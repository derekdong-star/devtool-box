package handler

import (
	"net/http"

	"devtoolbox/internal/model"
	"devtoolbox/internal/service"
)

type JSONHandler struct {
	svc *service.JSONService
}

func NewJSONHandler() *JSONHandler {
	return &JSONHandler{svc: service.NewJSONService()}
}

func (h *JSONHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/format-json", OnlyMethod(http.MethodPost, h.Format))
}

func (h *JSONHandler) Format(w http.ResponseWriter, r *http.Request) {
	var req model.JSONFmtReq
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	out, err := h.svc.Format(req.JSON, req.Mode)
	if err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	OK(w, out)
}
