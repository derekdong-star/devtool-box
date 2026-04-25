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

// Redis 连接与扫描常量
const (
	maxScanBatches     = 10
	defaultScanLimit   = 100
	zsetPreviewCount   = 199
	listPreviewCount   = 199

	redisDialTimeout   = 5 * time.Second
	redisReadTimeout   = 10 * time.Second
	redisWriteTimeout  = 5 * time.Second
	redisPingTimeout   = 5 * time.Second
)

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
		DialTimeout:  redisDialTimeout,
		ReadTimeout:  redisReadTimeout,
		WriteTimeout: redisWriteTimeout,
	})
	pingCtx, cancel := context.WithTimeout(context.Background(), redisPingTimeout)
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
		limit = defaultScanLimit
	}

	keys, err := s.scanWithLimit(ctx, c, pattern, limit)
	if err != nil {
		return nil, err
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
		result[i] = model.RedisKeyInfo{Key: k, Type: t, TTL: ttlToSeconds(ttl)}
	}
	return result, nil
}

// scanWithLimit 执行 SCAN 并限制返回数量
func (s *RedisService) scanWithLimit(ctx context.Context, c *redis.Client, pattern string, limit int64) ([]string, error) {
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
	return keys, nil
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

	resp := &model.RedisValueResp{Key: key, Type: keyType, TTL: ttlToSeconds(ttl)}
	val, err := s.getValueByType(ctx, c, keyType, key)
	if err != nil {
		return nil, err
	}
	resp.Value = val
	return resp, nil
}

// getValueByType 根据 Redis 类型选择对应的读取命令
func (s *RedisService) getValueByType(ctx context.Context, c *redis.Client, keyType, key string) (interface{}, error) {
	switch keyType {
	case "string":
		return c.Get(ctx, key).Result()
	case "list":
		return c.LRange(ctx, key, 0, listPreviewCount).Result()
	case "hash":
		return c.HGetAll(ctx, key).Result()
	case "set":
		return c.SMembers(ctx, key).Result()
	case "zset":
		return s.getZSetValue(ctx, c, key)
	default:
		return fmt.Sprintf("unsupported type: %s", keyType), nil
	}
}

// getZSetValue 读取 zset 并转换为可 JSON 序列化的结构
func (s *RedisService) getZSetValue(ctx context.Context, c *redis.Client, key string) ([]zItem, error) {
	members, err := c.ZRangeWithScores(ctx, key, 0, zsetPreviewCount).Result()
	if err != nil {
		return nil, err
	}
	items := make([]zItem, len(members))
	for i, z := range members {
		items[i] = zItem{Member: fmt.Sprint(z.Member), Score: z.Score}
	}
	return items, nil
}

// zItem 用于 JSON 序列化的 zset 元素
type zItem struct {
	Member string  `json:"member"`
	Score  float64 `json:"score"`
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

// ttlToSeconds 将 Redis TTL Duration 转换为秒表示：
// -1 = 永不过期, -2 = 已过期/不存在, >=0 = 剩余秒数
func ttlToSeconds(d time.Duration) int64 {
	if d == -1*time.Second {
		return -1
	}
	if d < 0 {
		return -2
	}
	return int64(d.Seconds())
}
