package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"devtoolbox/internal/model"

	"github.com/redis/go-redis/v9"
)

// maxScanBatches 限制 SCAN 最大批次数，防止无限扫描 (L-5)
const maxScanBatches = 10

// RedisService 管理 Redis 连接池并封装常用操作
type RedisService struct {
	pools map[string]*redis.Client
	mu    sync.RWMutex
}

func NewRedisService() *RedisService {
	return &RedisService{pools: make(map[string]*redis.Client)}
}

func poolKey(addr string, db int) string {
	return fmt.Sprintf("%s/%d", addr, db)
}

func (s *RedisService) getClient(addr, password string, db int) (*redis.Client, error) {
	key := poolKey(addr, db)

	s.mu.RLock()
	c, ok := s.pools[key]
	s.mu.RUnlock()
	if ok {
		return c, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.pools[key]; ok {
		return c, nil
	}

	c = redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 5 * time.Second,
	})
	// M-8: 使用带超时的 context 进行 Ping，避免永久阻塞
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Ping(pingCtx).Err(); err != nil {
		c.Close()
		return nil, fmt.Errorf("connect %s db%d: %w", addr, db, err)
	}
	s.pools[key] = c
	return c, nil
}

// Ping 测试连通性，成功返回服务器版本
func (s *RedisService) Ping(ctx context.Context, addr, password string, db int) (string, error) {
	c, err := s.getClient(addr, password, db)
	if err != nil {
		return "", err
	}
	info, err := c.Info(ctx, "server").Result()
	if err != nil {
		return "PONG", nil
	}
	for _, line := range strings.Split(info, "\n") {
		if strings.HasPrefix(line, "redis_version:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "redis_version:")), nil
		}
	}
	return "PONG", nil
}

// ScanKeys 用 SCAN 按 pattern 搜索 key（不阻塞，安全）
func (s *RedisService) ScanKeys(ctx context.Context, addr, password string, db int, pattern string, limit int64) ([]model.RedisKeyInfo, error) {
	c, err := s.getClient(addr, password, db)
	if err != nil {
		return nil, err
	}
	if pattern == "" {
		pattern = "*"
	}
	if limit <= 0 {
		limit = 100
	}

	var keys []string
	var cursor uint64
	for {
		batch, next, err := c.Scan(ctx, cursor, pattern, limit).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 || int64(len(keys)) >= limit*maxScanBatches {
			break
		}
	}

	if int64(len(keys)) > limit {
		keys = keys[:limit]
	}

	pipe := c.Pipeline()
	typeCmds := make([]*redis.StatusCmd, len(keys))
	ttlCmds := make([]*redis.DurationCmd, len(keys))
	for i, k := range keys {
		typeCmds[i] = pipe.Type(ctx, k)
		ttlCmds[i] = pipe.TTL(ctx, k)
	}
	pipe.Exec(ctx) //nolint

	result := make([]model.RedisKeyInfo, len(keys))
	for i, k := range keys {
		t, _ := typeCmds[i].Result()
		ttl, _ := ttlCmds[i].Result()
		ttlSec := int64(ttl.Seconds())
		if ttl == -1*time.Second {
			ttlSec = -1
		} else if ttl < 0 {
			ttlSec = -2
		}
		result[i] = model.RedisKeyInfo{Key: k, Type: t, TTL: ttlSec}
	}
	return result, nil
}

// GetValue 获取 key 的完整值（根据类型自动选择命令）
func (s *RedisService) GetValue(ctx context.Context, addr, password string, db int, key string) (*model.RedisValueResp, error) {
	c, err := s.getClient(addr, password, db)
	if err != nil {
		return nil, err
	}

	keyType, err := c.Type(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	ttl, _ := c.TTL(ctx, key).Result()
	ttlSec := int64(ttl.Seconds())
	if ttl == -1*time.Second {
		ttlSec = -1
	} else if ttl < 0 {
		ttlSec = -2
	}

	resp := &model.RedisValueResp{Key: key, Type: keyType, TTL: ttlSec}

	switch keyType {
	case "string":
		v, err := c.Get(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		resp.Value = v
	case "list":
		v, err := c.LRange(ctx, key, 0, 199).Result()
		if err != nil {
			return nil, err
		}
		resp.Value = v
	case "hash":
		v, err := c.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		resp.Value = v
	case "set":
		v, err := c.SMembers(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		resp.Value = v
	case "zset":
		v, err := c.ZRangeWithScores(ctx, key, 0, 199).Result()
		if err != nil {
			return nil, err
		}
		type zItem struct {
			Member string  `json:"member"`
			Score  float64 `json:"score"`
		}
		items := make([]zItem, len(v))
		for i, z := range v {
			items[i] = zItem{Member: fmt.Sprint(z.Member), Score: z.Score}
		}
		resp.Value = items
	default:
		resp.Value = fmt.Sprintf("unsupported type: %s", keyType)
	}

	return resp, nil
}

// Exec 执行原始命令（拆分后调用 Do）
func (s *RedisService) Exec(ctx context.Context, addr, password string, db int, command string) (interface{}, error) {
	c, err := s.getClient(addr, password, db)
	if err != nil {
		return nil, err
	}
	parts := splitCmd(command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	args := make([]interface{}, len(parts))
	for i, p := range parts {
		args[i] = p
	}
	return c.Do(ctx, args...).Result()
}

// splitCmd 简单按空格拆分，支持双引号包裹含空格的参数
func splitCmd(s string) []string {
	var parts []string
	var cur strings.Builder
	inQ := false
	for _, r := range s {
		switch {
		case r == '"':
			inQ = !inQ
		case r == ' ' && !inQ:
			if cur.Len() > 0 {
				parts = append(parts, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}
