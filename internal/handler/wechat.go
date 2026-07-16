package handler

import (
	"net/http"

	"devtoolbox/internal/model"
	"devtoolbox/internal/service"
)

type WeChatHandler struct {
	svc *service.WeChatService
}

func NewWeChatHandler() *WeChatHandler {
	return &WeChatHandler{svc: service.NewWeChatService()}
}

func (h *WeChatHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/wechat/themes", OnlyMethod(http.MethodGet, h.Themes))
	mux.HandleFunc("/api/wechat/format", OnlyMethod(http.MethodPost, h.Format))
}

func (h *WeChatHandler) Themes(w http.ResponseWriter, r *http.Request) {
	OK(w, h.svc.Themes())
}

func (h *WeChatHandler) Format(w http.ResponseWriter, r *http.Request) {
	var req model.WeChatFormatReq
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}

	out, err := h.svc.Format(req.Markdown, req.Theme)
	if err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	OK(w, out)
}
