package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"devtoolbox/internal/model"
	"devtoolbox/internal/service"
)

const uploadMultipartMemoryLimit = 32 << 20

type UploadHandler struct {
	store *service.UploadConfigStore
	svc   *service.UploadService
}

func NewUploadHandler(store *service.UploadConfigStore, svc *service.UploadService) *UploadHandler {
	return &UploadHandler{store: store, svc: svc}
}

func (h *UploadHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/config/upload", OnlyMethod(http.MethodGet, h.GetConfig))
	mux.HandleFunc("/api/config/upload/save", OnlyMethod(http.MethodPost, h.SaveConfig))
	mux.HandleFunc("/api/upload/file", OnlyMethod(http.MethodPost, h.Upload))
}

func (h *UploadHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.store.Load()
	if err != nil {
		Fail(w, http.StatusInternalServerError, "读取配置失败: "+err.Error())
		return
	}
	OK(w, cfg)
}

func (h *UploadHandler) SaveConfig(w http.ResponseWriter, r *http.Request) {
	var req model.UploadConfig
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if err := h.store.Save(req); err != nil {
		Fail(w, http.StatusInternalServerError, "保存配置失败: "+err.Error())
		return
	}
	OK(w, nil)
}

func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, service.MaxUploadRequestSize)
	if err := r.ParseMultipartForm(uploadMultipartMemoryLimit); err != nil {
		if isUploadTooLargeError(err) {
			Fail(w, http.StatusBadRequest, maxUploadFileSizeMessage())
			return
		}
		Fail(w, http.StatusBadRequest, "解析表单失败: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		Fail(w, http.StatusBadRequest, "请选择要上传的文件")
		return
	}
	defer file.Close()

	if header.Size > service.MaxUploadFileSize {
		Fail(w, http.StatusBadRequest, maxUploadFileSizeMessage())
		return
	}

	content, err := io.ReadAll(io.LimitReader(file, service.MaxUploadFileSize+1))
	if err != nil {
		Fail(w, http.StatusBadRequest, "读取上传文件失败: "+err.Error())
		return
	}
	if len(content) > service.MaxUploadFileSize {
		Fail(w, http.StatusBadRequest, maxUploadFileSizeMessage())
		return
	}

	useSignedURL, _ := strconv.ParseBool(strings.TrimSpace(r.FormValue("use_signed_url")))
	result, err := h.svc.Upload(r.Context(), service.UploadFileInput{
		FileName:     header.Filename,
		Directory:    r.FormValue("directory"),
		ObjectName:   r.FormValue("name"),
		ContentType:  header.Header.Get("Content-Type"),
		UseSignedURL: useSignedURL,
		Content:      content,
	})
	if err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	OK(w, result)
}

func isUploadTooLargeError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "http: request body too large")
}

func maxUploadFileSizeMessage() string {
	return fmt.Sprintf("文件过大，最大支持 %dMB", service.MaxUploadFileSize>>20)
}
