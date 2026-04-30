package handler

import (
	"net/http"

	"devtoolbox/internal/model"
	"devtoolbox/internal/service"
)

// ImageHandler 图片生成接口
type ImageHandler struct {
	svc *service.ImageService
}

// NewImageHandler 创建图片生成 handler
func NewImageHandler(svc *service.ImageService) *ImageHandler {
	return &ImageHandler{svc: svc}
}

func (h *ImageHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/image/models", OnlyMethod(http.MethodGet, h.Models))
	mux.HandleFunc("/api/image/generate", OnlyMethod(http.MethodPost, h.Generate))
	mux.HandleFunc("/api/image/generate-with-image", OnlyMethod(http.MethodPost, h.GenerateWithImage))
}

// Models 返回可用模型列表
func (h *ImageHandler) Models(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.svc.ConfigStore().Load()
	if err != nil {
		Fail(w, http.StatusInternalServerError, "读取配置失败: "+err.Error())
		return
	}
	OK(w, cfg.Models)
}

// Generate 文生图
func (h *ImageHandler) Generate(w http.ResponseWriter, r *http.Request) {
	var req model.ImageGenReq
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if req.Prompt == "" {
		Fail(w, http.StatusBadRequest, "prompt 不能为空")
		return
	}
	if req.Model == "" {
		Fail(w, http.StatusBadRequest, "model 不能为空")
		return
	}

	resp, err := h.svc.GenerateImage(req)
	if err != nil {
		Fail(w, http.StatusBadGateway, err.Error())
		return
	}
	OK(w, resp)
}

// GenerateWithImage 图生图
func (h *ImageHandler) GenerateWithImage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		Fail(w, http.StatusBadRequest, "解析表单失败: "+err.Error())
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		Fail(w, http.StatusBadRequest, "请上传图片文件")
		return
	}
	defer file.Close()

	req := model.ImageGenReq{
		Prompt: r.FormValue("prompt"),
		Model:  r.FormValue("model"),
		Size:   r.FormValue("size"),
	}
	if req.Prompt == "" {
		Fail(w, http.StatusBadRequest, "prompt 不能为空")
		return
	}
	if req.Model == "" {
		Fail(w, http.StatusBadRequest, "model 不能为空")
		return
	}

	resp, err := h.svc.GenerateWithImage(file, req)
	if err != nil {
		Fail(w, http.StatusBadGateway, err.Error())
		return
	}
	OK(w, resp)
}
