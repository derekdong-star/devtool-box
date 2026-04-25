package service

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/google/uuid"

	"devtoolbox/internal/model"
)

const connFile = "db_conns.json"

// ConnStore 负责将数据库连接配置持久化到本地 JSON 文件
type ConnStore struct {
	mu    sync.RWMutex
	store *jsonStore[model.DBConn]
}

func NewConnStore() *ConnStore {
	dir := os.Getenv("DATA_DIR")
	if dir == "" {
		exe, err := os.Executable()
		dir = "."
		if err == nil {
			dir = filepath.Dir(exe)
		}
	}
	return &ConnStore{store: newJSONStore[model.DBConn](filepath.Join(dir, connFile))}
}

// List 返回所有已保存的连接，按保存时间倒序（最新在前）
func (s *ConnStore) List() ([]model.DBConn, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.store.load()
}

// ListDB 返回非 Redis 的数据库连接列表（业务过滤在 service 层）
func (s *ConnStore) ListDB() ([]model.DBConn, error) {
	all, err := s.List()
	if err != nil {
		return nil, err
	}
	result := make([]model.DBConn, 0, len(all))
	for _, c := range all {
		if c.Type != "redis" {
			result = append(result, c)
		}
	}
	return result, nil
}

// ListByType 返回指定 type 的连接列表
func (s *ConnStore) ListByType(connType string) ([]model.DBConn, error) {
	all, err := s.List()
	if err != nil {
		return nil, err
	}
	result := make([]model.DBConn, 0, len(all))
	for _, c := range all {
		if c.Type == connType {
			result = append(result, c)
		}
	}
	return result, nil
}

// Save 保存一条连接；若已存在相同 type+DSN 则直接返回，不重复写入
func (s *ConnStore) Save(dbType, dsn string) (model.DBConn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	conns, err := s.store.load()
	if err != nil {
		return model.DBConn{}, err
	}

	// 去重：同 type + DSN 已存在则直接返回
	for _, c := range conns {
		if c.Type == dbType && c.DSN == dsn {
			return c, nil
		}
	}

	conn := model.DBConn{
		ID:   uuid.New().String(),
		Name: buildName(dbType, dsn),
		Type: dbType,
		DSN:  dsn,
	}
	conns = append([]model.DBConn{conn}, conns...)
	return conn, s.store.save(conns)
}

// Delete 按 ID 删除一条连接
func (s *ConnStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conns, err := s.store.load()
	if err != nil {
		return err
	}
	filtered := conns[:0]
	for _, c := range conns {
		if c.ID != id {
			filtered = append(filtered, c)
		}
	}
	return s.store.save(filtered)
}

// buildName 从 DSN 提取人类可读的展示名
func truncate(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max]) + "…"
}

func buildName(dbType, dsn string) string {
	switch dbType {
	case "mysql":
		// root:pass@tcp(host:port)/dbname
		at := strings.LastIndex(dsn, "@")
		slash := strings.LastIndex(dsn, "/")
		if at >= 0 && slash > at {
			host := dsn[at+1 : slash]
			db := dsn[slash+1:]
			if strings.HasPrefix(host, "tcp(") {
				host = host[4 : len(host)-1]
			}
			return fmt.Sprintf("mysql@%s/%s", host, db)
		}
	case "postgres":
		return fmt.Sprintf("pg@%s", truncate(dsn, 40))
	case "sqlite3":
		return fmt.Sprintf("sqlite:%s", filepath.Base(dsn))
	case "redis":
		// addr?db=N[&password=xxx]  →  redis@addr/dbN
		q := strings.Index(dsn, "?")
		addr := dsn
		dbNum := "0"
		if q >= 0 {
			addr = dsn[:q]
			if v, err := url.ParseQuery(dsn[q+1:]); err == nil {
				if d := v.Get("db"); d != "" {
					dbNum = d
				}
			}
		}
		return fmt.Sprintf("redis@%s/db%s", addr, dbNum)
	}
	return fmt.Sprintf("%s:%s", dbType, truncate(dsn, 40))
}
