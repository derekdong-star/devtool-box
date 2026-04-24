package handler

import (
	"io"
	"net/http"

	"devtoolbox/internal/model"
	"devtoolbox/internal/service"
)

type UtilHandler struct {
	svc *service.UtilService
}

func NewUtilHandler() *UtilHandler {
	return &UtilHandler{svc: service.NewUtilService()}
}

func (h *UtilHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/timestamp", OnlyMethod(http.MethodPost, h.Timestamp))
	mux.HandleFunc("/api/uuid", OnlyMethod(http.MethodPost, h.UUID))
}

func (h *UtilHandler) Timestamp(w http.ResponseWriter, r *http.Request) {
	var req model.TSReq
	// M-2: 区分"空 body"（合法，表示查当前时间）与"格式错误 JSON"（非法）
	if err := BindJSON(r, &req); err != nil && err != io.EOF {
		Fail(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.TS == "" {
		sec, ms, rfc := h.svc.Now()
		OK(w, model.TSResp{Sec: sec, Ms: ms, RFC: rfc})
		return
	}

	sec, ms, rfc, err := h.svc.ParseTS(req.TS)
	if err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	OK(w, model.TSResp{Sec: sec, Ms: ms, RFC: rfc})
}

func (h *UtilHandler) UUID(w http.ResponseWriter, r *http.Request) {
	var req model.UUIDReq
	// M-2: 同上，空 body 时使用默认值，格式错误才报 400
	if err := BindJSON(r, &req); err != nil && err != io.EOF {
		Fail(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.N <= 0 || req.N > 100 {
		req.N = 5
	}
	OK(w, h.svc.GenUUID(req.N))
}
