package model

// CookieParseReq 解析Cookie请求
type CookieParseReq struct {
	Cookie string `json:"cookie"`
}

// JWTDecodeReq JWT解码请求
type JWTDecodeReq struct {
	Token string `json:"token"`
}

// JWTDecodeResp JWT解码响应
type JWTDecodeResp struct {
	Header    map[string]interface{} `json:"header"`
	Payload   map[string]interface{} `json:"payload"`
	Signature string                 `json:"signature"`
}

// JSONFmtReq JSON格式化请求
type JSONFmtReq struct {
	JSON string `json:"json"`
	Mode string `json:"mode"`
}

// CodecReq 编解码请求
type CodecReq struct {
	Text string `json:"text"`
	Mode string `json:"mode"`
}

// DBQueryReq 数据库查询请求
type DBQueryReq struct {
	Type  string `json:"type"`
	DSN   string `json:"dsn"`
	Query string `json:"query"`
}

// DBConnReq 仅需连接信息的请求（列表表、描述表）
type DBConnReq struct {
	Type  string `json:"type"`
	DSN   string `json:"dsn"`
	Table string `json:"table,omitempty"`
}

// TSReq 时间戳请求
type TSReq struct {
	TS string `json:"ts"`
}

// TSResp 时间戳响应
type TSResp struct {
	Sec int64  `json:"sec"`
	Ms  int64  `json:"ms"`
	RFC string `json:"rfc"`
}

// UUIDReq UUID生成请求
type UUIDReq struct {
	N int `json:"n"`
}

// SessionParseReq Session解析请求
type SessionParseReq struct {
	Cookie string `json:"cookie"`
	Secret string `json:"secret"` // 可选，有值则验签
}

// DBConn 已保存的数据库连接配置
type DBConn struct {
	ID    string `json:"id"`    // 唯一 ID，用时间戳生成
	Name  string `json:"name"`  // 显示名（自动生成：type@host/db）
	Type  string `json:"type"`
	DSN   string `json:"dsn"`
}

// DBConnDeleteReq 删除连接请求
type DBConnDeleteReq struct {
	ID string `json:"id"`
}

// ColumnInfo 表结构中单列的描述信息
type ColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable string `json:"nullable"`
	Key      string `json:"key"`
	Default  string `json:"default"`
	Extra    string `json:"extra"`
}

// ── Redis ──────────────────────────────────────────────────────

// RedisConnReq Redis 连接信息
type RedisConnReq struct {
	Addr     string `json:"addr"`     // host:port
	Password string `json:"password"` // 可选
	DB       int    `json:"db"`       // 默认 0
}

// RedisKeysReq 前缀搜索 key
type RedisKeysReq struct {
	RedisConnReq
	Pattern string `json:"pattern"` // 支持 glob，如 user:*
	Count   int64  `json:"count"`   // scan 每批数量，默认 100
}

// RedisGetReq 获取单个 key
type RedisGetReq struct {
	RedisConnReq
	Key string `json:"key"`
}

// RedisExecReq 执行原始命令
type RedisExecReq struct {
	RedisConnReq
	Command string `json:"command"` // 如 "GET foo" / "HGETALL user:1"
}

// RedisKeyInfo 单个 key 的元信息
type RedisKeyInfo struct {
	Key  string `json:"key"`
	Type string `json:"type"`
	TTL  int64  `json:"ttl"` // 秒，-1 永不过期，-2 不存在
}

// RedisValueResp key 详情
type RedisValueResp struct {
	Key   string      `json:"key"`
	Type  string      `json:"type"`
	TTL   int64       `json:"ttl"`
	Value interface{} `json:"value"`
}

// ── 命令模板 ───────────────────────────────────────────────────

// CmdTemplate 保存的命令模板
type CmdTemplate struct {
	ID      string `json:"id"`      // uuid
	Name    string `json:"name"`    // 用户起的名字
	Command string `json:"command"` // 命令内容
	Kind    string `json:"kind"`    // "sql" | "redis"
}

// CmdTemplateSaveReq 保存模板请求
type CmdTemplateSaveReq struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Kind    string `json:"kind"`
}

// CmdTemplateDeleteReq 删除模板请求
type CmdTemplateDeleteReq struct {
	ID string `json:"id"`
}
