package handler

import (
	"net/http"

	"devtoolbox/internal/model"
	"devtoolbox/internal/service"
)

type ConnStoreHandler struct {
	store *service.ConnStore
}

func NewConnStoreHandler(store *service.ConnStore) *ConnStoreHandler {
	return &ConnStoreHandler{store: store}
}

func (h *ConnStoreHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/db/conns", OnlyMethod(http.MethodGet, h.List))
	mux.HandleFunc("/api/db/conns/delete", OnlyMethod(http.MethodPost, h.Delete))
	mux.HandleFunc("/api/redis/conns", OnlyMethod(http.MethodGet, h.ListRedis))
}

// List GET → 返回非 redis 的数据库连接列表
func (h *ConnStoreHandler) List(w http.ResponseWriter, r *http.Request) {
	conns, err := h.store.ListDB() // M-3: 过滤逻辑下沉到 service 层
	if err != nil {
		Fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	OK(w, conns)
}

// ListRedis GET → 返回 redis 连接列表
func (h *ConnStoreHandler) ListRedis(w http.ResponseWriter, r *http.Request) {
	conns, err := h.store.ListByType("redis")
	if err != nil {
		Fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	OK(w, conns)
}

// Delete POST → 按 ID 删除
func (h *ConnStoreHandler) Delete(w http.ResponseWriter, r *http.Request) {
	var req model.DBConnDeleteReq
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.ID == "" {
		Fail(w, http.StatusBadRequest, "id is required")
		return
	}
	if err := h.store.Delete(req.ID); err != nil {
		Fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	OK(w, nil)
}
