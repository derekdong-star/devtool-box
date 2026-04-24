package handler

import (
	"net/http"
	"net/url"
	"strconv"

	"devtoolbox/internal/model"
	"devtoolbox/internal/service"
)

type RedisHandler struct {
	svc   *service.RedisService
	store *service.ConnStore
}

func NewRedisHandler(store *service.ConnStore) *RedisHandler {
	return &RedisHandler{
		svc:   service.NewRedisService(),
		store: store,
	}
}

func (h *RedisHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/redis/ping", OnlyMethod(http.MethodPost, h.Ping))
	mux.HandleFunc("/api/redis/keys", OnlyMethod(http.MethodPost, h.Keys))
	mux.HandleFunc("/api/redis/get", OnlyMethod(http.MethodPost, h.Get))
	mux.HandleFunc("/api/redis/exec", OnlyMethod(http.MethodPost, h.Exec))
}

// Ping 测试连接并自动保存
func (h *RedisHandler) Ping(w http.ResponseWriter, r *http.Request) {
	var req model.RedisConnReq
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	version, err := h.svc.Ping(r.Context(), req.Addr, req.Password, req.DB)
	if err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	// 连接成功 → 自动保存
	h.store.Save("redis", redisConnDSN(req)) //nolint
	OK(w, map[string]string{"version": version})
}

// Keys 前缀/pattern 搜索
func (h *RedisHandler) Keys(w http.ResponseWriter, r *http.Request) {
	var req model.RedisKeysReq
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	keys, err := h.svc.ScanKeys(r.Context(), req.Addr, req.Password, req.DB, req.Pattern, req.Count)
	if err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	OK(w, keys)
}

// Get 获取单个 key 完整值
func (h *RedisHandler) Get(w http.ResponseWriter, r *http.Request) {
	var req model.RedisGetReq
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Key == "" {
		Fail(w, http.StatusBadRequest, "key is required")
		return
	}
	resp, err := h.svc.GetValue(r.Context(), req.Addr, req.Password, req.DB, req.Key)
	if err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	OK(w, resp)
}

// Exec 执行原始命令
func (h *RedisHandler) Exec(w http.ResponseWriter, r *http.Request) {
	var req model.RedisExecReq
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Command == "" {
		Fail(w, http.StatusBadRequest, "command is required")
		return
	}
	result, err := h.svc.Exec(r.Context(), req.Addr, req.Password, req.DB, req.Command)
	if err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	OK(w, result)
}

// redisConnDSN 生成 Redis 连接的 DSN 字符串，用于 ConnStore
// 格式：addr?db=N[&password=xxx]
func redisConnDSN(req model.RedisConnReq) string {
	q := url.Values{}
	q.Set("db", strconv.Itoa(req.DB))
	if req.Password != "" {
		q.Set("password", req.Password)
	}
	return req.Addr + "?" + q.Encode()
}
