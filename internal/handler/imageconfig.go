package handler

import (
	"net/http"
	"strings"

	"devtoolbox/internal/model"
	"devtoolbox/internal/service"
)

// ImageConfigHandler 图片 API 配置接口
type ImageConfigHandler struct {
	store *service.ImageConfigStore
}

// NewImageConfigHandler 创建配置 handler
func NewImageConfigHandler(store *service.ImageConfigStore) *ImageConfigHandler {
	return &ImageConfigHandler{store: store}
}

func (h *ImageConfigHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/config/image", OnlyMethod(http.MethodGet, h.Get))
	mux.HandleFunc("/api/config/image/save", OnlyMethod(http.MethodPost, h.Save))
}

func (h *ImageConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.store.Load()
	if err != nil {
		Fail(w, http.StatusInternalServerError, "读取配置失败: "+err.Error())
		return
	}
	OK(w, cfg)
}

func (h *ImageConfigHandler) Save(w http.ResponseWriter, r *http.Request) {
	var req model.ImageConfig
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	// 清理模型列表空白行
	models := make([]string, 0, len(req.Models))
	for _, m := range req.Models {
		m = strings.TrimSpace(m)
		if m != "" {
			models = append(models, m)
		}
	}
	req.Models = models

	if err := h.store.Save(req); err != nil {
		Fail(w, http.StatusInternalServerError, "保存配置失败: "+err.Error())
		return
	}
	OK(w, nil)
}
