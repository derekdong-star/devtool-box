package handler

import (
	"net/http"
	"strings"

	"devtoolbox/internal/model"
	"devtoolbox/internal/service"
)

type DBHandler struct {
	svc   *service.DBService
	store *service.ConnStore
}

func NewDBHandler(store *service.ConnStore) *DBHandler {
	return &DBHandler{
		svc:   service.NewDBService(),
		store: store,
	}
}

func (h *DBHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/query-db", OnlyMethod(http.MethodPost, h.Query))
	mux.HandleFunc("/api/db/tables", OnlyMethod(http.MethodPost, h.ListTables))
	mux.HandleFunc("/api/db/describe", OnlyMethod(http.MethodPost, h.DescribeTable))
}

func (h *DBHandler) Query(w http.ResponseWriter, r *http.Request) {
	var req model.DBQueryReq
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.Query) == "" {
		Fail(w, http.StatusBadRequest, "query is empty")
		return
	}
	res, err := h.svc.Query(req.Type, req.DSN, req.Query)
	if err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	OK(w, res)
}

func (h *DBHandler) ListTables(w http.ResponseWriter, r *http.Request) {
	var req model.DBConnReq
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	tables, err := h.svc.ListTables(req.Type, req.DSN)
	if err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	// 连接成功 → 自动保存（去重由 ConnStore 内部处理）
	h.store.Save(req.Type, req.DSN) //nolint
	OK(w, tables)
}

func (h *DBHandler) DescribeTable(w http.ResponseWriter, r *http.Request) {
	var req model.DBConnReq
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.Table) == "" {
		Fail(w, http.StatusBadRequest, "table name is required")
		return
	}
	cols, err := h.svc.DescribeTable(req.Type, req.DSN, req.Table)
	if err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	OK(w, cols)
}
