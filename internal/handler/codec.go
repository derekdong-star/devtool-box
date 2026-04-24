package handler

import (
	"net/http"

	"devtoolbox/internal/model"
	"devtoolbox/internal/service"
)

type CodecHandler struct {
	svc *service.CodecService
}

func NewCodecHandler() *CodecHandler {
	return &CodecHandler{svc: service.NewCodecService()}
}

func (h *CodecHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/base64", OnlyMethod(http.MethodPost, h.Base64))
	mux.HandleFunc("/api/urlcodec", OnlyMethod(http.MethodPost, h.URLCodec))
}

func (h *CodecHandler) Base64(w http.ResponseWriter, r *http.Request) {
	var req model.CodecReq
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	out, err := h.svc.Base64(req.Text, req.Mode)
	if err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	OK(w, out)
}

func (h *CodecHandler) URLCodec(w http.ResponseWriter, r *http.Request) {
	var req model.CodecReq
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	out, err := h.svc.URL(req.Text, req.Mode)
	if err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	OK(w, out)
}
