package service

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"devtoolbox/internal/model"
)

type DBService struct {
	pools map[string]*sql.DB
	mu    sync.RWMutex
}

func NewDBService() *DBService {
	return &DBService{pools: make(map[string]*sql.DB)}
}

func (s *DBService) getDB(dbType, dsn string) (*sql.DB, error) {
	key := fmt.Sprintf("%s://%s", dbType, dsn)

	s.mu.RLock()
	db, ok := s.pools[key]
	s.mu.RUnlock()
	if ok {
		return db, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if db, ok := s.pools[key]; ok {
		return db, nil
	}
	db, err := sql.Open(dbType, dsn)
	if err != nil {
		return nil, err
	}
	// M-7: 连接池参数限制，防止资源泄漏
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(10 * time.Minute)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	s.pools[key] = db
	return db, nil
}

// Query 执行任意 SQL，返回行数组
func (s *DBService) Query(dbType, dsn, query string) ([]map[string]interface{}, error) {
	db, err := s.getDB(dbType, dsn)
	if err != nil {
		return nil, err
	}
	return s.scanRows(db.Query(query))
}

// ListTables 列出当前数据库所有用户表名
func (s *DBService) ListTables(dbType, dsn string) ([]string, error) {
	db, err := s.getDB(dbType, dsn)
	if err != nil {
		return nil, err
	}

	var q string
	switch dbType {
	case "mysql":
		q = "SHOW TABLES"
	case "postgres":
		q = "SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename"
	case "sqlite3":
		q = "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name"
	default:
		return nil, fmt.Errorf("unsupported db type: %s", dbType)
	}

	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

// DescribeTable 返回表结构（列信息）
func (s *DBService) DescribeTable(dbType, dsn, table string) ([]model.ColumnInfo, error) {
	db, err := s.getDB(dbType, dsn)
	if err != nil {
		return nil, err
	}

	switch dbType {
	case "mysql":
		return s.describeMySQL(db, table)
	case "postgres":
		return s.describePostgres(db, table)
	case "sqlite3":
		return s.describeSQLite(db, table)
	default:
		return nil, fmt.Errorf("unsupported db type: %s", dbType)
	}
}

func (s *DBService) describeMySQL(db *sql.DB, table string) ([]model.ColumnInfo, error) {
	rows, err := db.Query(fmt.Sprintf("DESCRIBE `%s`", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []model.ColumnInfo
	for rows.Next() {
		var c model.ColumnInfo
		var nullable, key, def, extra sql.NullString
		if err := rows.Scan(&c.Name, &c.Type, &nullable, &key, &def, &extra); err != nil {
			return nil, err
		}
		c.Nullable = nullable.String
		c.Key = key.String
		c.Default = def.String
		c.Extra = extra.String
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

func (s *DBService) describePostgres(db *sql.DB, table string) ([]model.ColumnInfo, error) {
	q := `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable,
			COALESCE(
				(SELECT 'PRI' FROM information_schema.table_constraints tc
				 JOIN information_schema.key_column_usage kcu
				   ON tc.constraint_name = kcu.constraint_name
				  AND tc.table_name = kcu.table_name
				 WHERE tc.constraint_type = 'PRIMARY KEY'
				   AND kcu.table_name = c.table_name
				   AND kcu.column_name = c.column_name
				 LIMIT 1), '') AS key,
			COALESCE(c.column_default, '') AS col_default,
			'' AS extra
		FROM information_schema.columns c
		WHERE c.table_name = $1 AND c.table_schema = 'public'
		ORDER BY c.ordinal_position`
	rows, err := db.Query(q, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []model.ColumnInfo
	for rows.Next() {
		var c model.ColumnInfo
		if err := rows.Scan(&c.Name, &c.Type, &c.Nullable, &c.Key, &c.Default, &c.Extra); err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

func (s *DBService) describeSQLite(db *sql.DB, table string) ([]model.ColumnInfo, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(`%s`)", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []model.ColumnInfo
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dfltVal sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltVal, &pk); err != nil {
			return nil, err
		}
		nullable := "YES"
		if notNull == 1 {
			nullable = "NO"
		}
		key := ""
		if pk > 0 {
			key = "PRI"
		}
		cols = append(cols, model.ColumnInfo{
			Name:     name,
			Type:     colType,
			Nullable: nullable,
			Key:      key,
			Default:  dfltVal.String,
		})
	}
	return cols, rows.Err()
}

// scanRows 通用行扫描
func (s *DBService) scanRows(rows *sql.Rows, err error) ([]map[string]interface{}, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{})
		for i, c := range cols {
			switch v := vals[i].(type) {
			case []byte:
				row[c] = string(v)
			default:
				row[c] = v
			}
		}
		results = append(results, row)
	}
	return results, rows.Err()
}
