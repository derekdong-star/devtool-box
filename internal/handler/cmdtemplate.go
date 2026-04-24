package handler

import (
	"net/http"

	"devtoolbox/internal/model"
	"devtoolbox/internal/service"
)

type CmdTemplateHandler struct {
	store *service.CmdTemplateStore
}

func NewCmdTemplateHandler(store *service.CmdTemplateStore) *CmdTemplateHandler {
	return &CmdTemplateHandler{store: store}
}

func (h *CmdTemplateHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/cmd-templates", OnlyMethod(http.MethodGet, h.List))
	mux.HandleFunc("/api/cmd-templates/save", OnlyMethod(http.MethodPost, h.Save))
	mux.HandleFunc("/api/cmd-templates/delete", OnlyMethod(http.MethodPost, h.Delete))
}

// List 返回模板列表，可通过 ?kind=sql 或 ?kind=redis 过滤
func (h *CmdTemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")
	templates, err := h.store.List(kind)
	if err != nil {
		Fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	OK(w, templates)
}

// Save 保存一条模板
func (h *CmdTemplateHandler) Save(w http.ResponseWriter, r *http.Request) {
	var req model.CmdTemplateSaveReq
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Kind != "sql" && req.Kind != "redis" {
		Fail(w, http.StatusBadRequest, "kind must be 'sql' or 'redis'")
		return
	}
	tmpl, err := h.store.Save(req.Name, req.Command, req.Kind)
	if err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	OK(w, tmpl)
}

// Delete 删除一条模板
func (h *CmdTemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	var req model.CmdTemplateDeleteReq
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
