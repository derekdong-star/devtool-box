package service

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"devtoolbox/internal/model"
)

const imageRequestTimeout = 300 * time.Second

// openaiImageResponse 用于解析 OpenAI Images API 的标准响应结构
// 在 callImageAPI 和 callImageEditAPI 中复用，避免匿名 struct 重复定义
type openaiImageResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		URL           string `json:"url"`
		B64JSON       string `json:"b64_json"`
		RevisedPrompt string `json:"revised_prompt"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ImageService 图片生成服务
type ImageService struct {
	cfgStore *ImageConfigStore
	client   *http.Client
}

// NewImageService 创建图片生成服务
func NewImageService(cfgStore *ImageConfigStore) *ImageService {
	return &ImageService{
		cfgStore: cfgStore,
		client:   &http.Client{Timeout: imageRequestTimeout},
	}
}

// ConfigStore 暴露配置存储（供 handler 读取模型列表）
func (s *ImageService) ConfigStore() *ImageConfigStore {
	return s.cfgStore
}

// GenerateImage 文生图
func (s *ImageService) GenerateImage(req model.ImageGenReq) (*model.ImageGenResp, error) {
	cfg, err := s.cfgStore.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	if shouldUseOpenAIImageSDK(cfg, req.Model) {
		return s.generateImageWithOpenAI(cfg, req)
	}
	if normalizeImageAPIBaseURL(cfg.APIURL) == "" {
		return nil, fmt.Errorf("API URL 未配置，请先保存配置")
	}

	body := map[string]interface{}{
		"prompt": req.Prompt,
		"model":  req.Model,
		"n":      defaultN(req.N),
	}
	maybeSetImageResponseFormat(body, req.Model)
	if req.Size != "" {
		body["size"] = req.Size
	}

	result, err := s.callImageAPI(cfg, body)
	if err != nil {
		log.Printf("[image] images/generations failed: %v, falling back to chat/completions", err)
		return s.callChatCompletions(cfg, req)
	}
	return result, nil
}

// GenerateWithImage 图生图（image edit/variation style）
func (s *ImageService) GenerateWithImage(file multipart.File, req model.ImageGenReq) (*model.ImageGenResp, error) {
	cfg, err := s.cfgStore.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	if shouldUseOpenAIImageSDK(cfg, req.Model) {
		return s.editImageWithOpenAI(cfg, file, req)
	}
	if normalizeImageAPIBaseURL(cfg.APIURL) == "" {
		return nil, fmt.Errorf("API URL 未配置，请先保存配置")
	}

	// 读取上传的图片并转为 base64 data URL
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}
	mimeType := http.DetectContentType(data)
	b64 := base64.StdEncoding.EncodeToString(data)
	imageDataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, b64)

	result, err := s.callImageEditAPI(cfg, data, mimeType, req)
	if err != nil {
		// Fallback: try chat completions streaming (for gateways like palebluedot)
		// Note: for image-to-image, we include the image URL in the prompt
		log.Printf("[image] GenerateWithImage fallback, image data URL len=%d, mime=%s", len(imageDataURL), mimeType)
		req.Prompt = req.Prompt + "\n\n[Image: " + imageDataURL + "]"
		return s.callChatCompletions(cfg, req)
	}
	return result, nil
}

func (s *ImageService) callImageEditAPI(cfg model.ImageConfig, imageData []byte, mimeType string, req model.ImageGenReq) (*model.ImageGenResp, error) {
	baseURL := normalizeImageAPIBaseURL(cfg.APIURL)
	if baseURL == "" {
		return nil, fmt.Errorf("API URL 未配置，请先保存配置")
	}

	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)

	if err := writer.WriteField("prompt", req.Prompt); err != nil {
		return nil, fmt.Errorf("write prompt: %w", err)
	}
	if err := writer.WriteField("model", req.Model); err != nil {
		return nil, fmt.Errorf("write model: %w", err)
	}
	if err := writer.WriteField("n", fmt.Sprintf("%d", defaultN(req.N))); err != nil {
		return nil, fmt.Errorf("write n: %w", err)
	}
	if req.Size != "" {
		if err := writer.WriteField("size", req.Size); err != nil {
			return nil, fmt.Errorf("write size: %w", err)
		}
	}

	fileWriter, err := writer.CreateFormFile("image", imageFilenameFromMimeType(mimeType))
	if err != nil {
		return nil, fmt.Errorf("create image form file: %w", err)
	}
	if _, err := fileWriter.Write(imageData); err != nil {
		return nil, fmt.Errorf("write image form file: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, baseURL+"/images/edits", &payload)
	if err != nil {
		return nil, fmt.Errorf("build edit request: %w", err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("edit request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read edit response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	if len(respBody) == 0 {
		return nil, fmt.Errorf("API returned empty response body (status 200)")
	}

	var openaiResp openaiImageResponse
	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		return nil, fmt.Errorf("decode edit response: %w (body: %s)", err, string(respBody))
	}
	if openaiResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", openaiResp.Error.Message)
	}

	result := &model.ImageGenResp{
		Created: openaiResp.Created,
		Data:    make([]model.ImageGenResult, 0, len(openaiResp.Data)),
	}
	for _, d := range openaiResp.Data {
		url := d.URL
		if url == "" && d.B64JSON != "" {
			url = "data:image/png;base64," + d.B64JSON
		}
		result.Data = append(result.Data, model.ImageGenResult{
			URL:           url,
			RevisedPrompt: d.RevisedPrompt,
		})
	}

	return result, nil
}

func (s *ImageService) callImageAPI(cfg model.ImageConfig, body map[string]interface{}) (*model.ImageGenResp, error) {
	baseURL := normalizeImageAPIBaseURL(cfg.APIURL)
	if baseURL == "" {
		return nil, fmt.Errorf("API URL 未配置，请先保存配置")
	}

	url := baseURL + "/images/generations"

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	if len(respBody) == 0 {
		return nil, fmt.Errorf("API returned empty response body (status 200)")
	}

	var openaiResp openaiImageResponse
	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w (body: %s)", err, string(respBody))
	}

	if openaiResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", openaiResp.Error.Message)
	}

	result := &model.ImageGenResp{
		Created: openaiResp.Created,
		Data:    make([]model.ImageGenResult, 0, len(openaiResp.Data)),
	}
	for _, d := range openaiResp.Data {
		url := d.URL
		if url == "" && d.B64JSON != "" {
			url = "data:image/png;base64," + d.B64JSON
		}
		result.Data = append(result.Data, model.ImageGenResult{
			URL:           url,
			RevisedPrompt: d.RevisedPrompt,
		})
	}

	return result, nil
}

// callChatCompletions 通过 chat completions streaming 生成图片（fallback for gateways like palebluedot）
func (s *ImageService) callChatCompletions(cfg model.ImageConfig, req model.ImageGenReq) (*model.ImageGenResp, error) {
	baseURL := normalizeImageAPIBaseURL(cfg.APIURL)
	if baseURL == "" {
		return nil, fmt.Errorf("API URL 未配置，请先保存配置")
	}

	url := baseURL + "/chat/completions"

	content := req.Prompt
	if req.Size != "" {
		content += "\n\n[Image size: " + req.Size + "]"
	}

	body := map[string]interface{}{
		"model":    req.Model,
		"messages": []map[string]string{{"role": "user", "content": content}},
		"stream":   true,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal chat request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build chat request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		log.Printf("[image] chat request network error: %v", err)
		return nil, fmt.Errorf("chat request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("[image] chat API error status=%d body=%s", resp.StatusCode, string(respBody))
		if resp.StatusCode == http.StatusGatewayTimeout {
			return nil, fmt.Errorf("上游服务响应超时 (504)，图片生成耗时较长，请稍后重试")
		}
		return nil, fmt.Errorf("chat API error (%d): %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[image] chat API status=%d, starting SSE parse", resp.StatusCode)

	// Parse SSE stream using bufio.Reader (no line-length limit vs scanner)
	var b64Chunks []string
	reader := bufio.NewReader(resp.Body)
	lineCount := 0
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			log.Printf("[image] stream read error after %d lines: %v", lineCount, readErr)
			return nil, fmt.Errorf("read stream: %w", readErr)
		}
		lineCount++
		line = strings.TrimRight(line, "\r\n")
		if !strings.HasPrefix(line, "data: ") {
			if lineCount <= 5 {
				log.Printf("[image] SSE line %d (non-data): %q", lineCount, line)
			}
			if readErr == io.EOF {
				break
			}
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			log.Printf("[image] SSE [DONE] received after %d lines", lineCount)
			break
		}
		if lineCount <= 3 {
			log.Printf("[image] SSE line %d: %s", lineCount, data)
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Images []struct {
						ImageURL struct {
							URL string `json:"url"`
						} `json:"image_url"`
					} `json:"images"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			if readErr == io.EOF {
				break
			}
			continue
		}

		if len(chunk.Choices) > 0 {
			for _, img := range chunk.Choices[0].Delta.Images {
				url := img.ImageURL.URL
				if strings.HasPrefix(url, "data:image/") {
					log.Printf("[image] found image data URL, len=%d", len(url))
					b64Chunks = append(b64Chunks, url)
				}
			}
		}
		if readErr == io.EOF {
			break
		}
	}

	log.Printf("[image] SSE parse complete: %d lines, %d b64 chunks", lineCount, len(b64Chunks))
	if len(b64Chunks) == 0 {
		return nil, fmt.Errorf("no image data found in stream")
	}

	return &model.ImageGenResp{
		Created: time.Now().Unix(),
		Data: []model.ImageGenResult{
			{URL: strings.Join(b64Chunks, "")},
		},
	}, nil
}

func normalizeImageAPIBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimRight(raw, "/")
	for _, suffix := range []string{"/images/generations", "/images/edits", "/chat/completions"} {
		if strings.HasSuffix(raw, suffix) {
			return strings.TrimSuffix(raw, suffix)
		}
	}
	return raw
}

func normalizeImageModelName(modelName string) string {
	modelName = strings.TrimSpace(modelName)
	modelName = strings.TrimPrefix(modelName, "openai/")
	return modelName
}

func isGPTImageModel(modelName string) bool {
	modelName = strings.ToLower(normalizeImageModelName(modelName))
	return strings.Contains(modelName, "gpt") && strings.Contains(modelName, "image")
}

func isOpenAIImageModel(modelName string) bool {
	modelName = normalizeImageModelName(modelName)
	return isGPTImageModel(modelName) || strings.HasPrefix(modelName, "dall-e-")
}

// shouldUseOpenAIImageSDK 判断是否使用官方 OpenAI Go SDK（而非通用 HTTP fallback）
// 条件 A：模型名是已知的 OpenAI 原生图像模型（gpt-image-* / dall-e-*）
// 条件 B：用户显式配置了 OpenAI 官方 endpoint，且模型前缀为 openai/*（用于兼容未来新模型）
func shouldUseOpenAIImageSDK(cfg model.ImageConfig, modelName string) bool {
	if isOpenAIImageModel(modelName) {
		return true
	}

	baseURL := normalizeImageAPIBaseURL(cfg.APIURL)
	return baseURL == "https://api.openai.com/v1" && strings.HasPrefix(strings.ToLower(modelName), "openai/")
}

func maybeSetImageResponseFormat(body map[string]interface{}, modelName string) {
	modelName = normalizeImageModelName(modelName)
	if isGPTImageModel(modelName) {
		return
	}
	if strings.HasPrefix(modelName, "dall-e-") {
		body["response_format"] = "b64_json"
	}
}

func defaultN(n int) int {
	if n <= 0 || n > 10 {
		return 1
	}
	return n
}

func imageFilenameFromMimeType(mimeType string) string {
	extensions, err := mime.ExtensionsByType(mimeType)
	if err == nil && len(extensions) > 0 {
		return "upload" + extensions[0]
	}
	return "upload.png"
}
