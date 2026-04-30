package service

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"devtoolbox/internal/model"
)

const imageRequestTimeout = 300 * time.Second

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
	if cfg.APIURL == "" {
		return nil, fmt.Errorf("API URL 未配置，请先保存配置")
	}

	body := map[string]interface{}{
		"prompt":          req.Prompt,
		"model":           req.Model,
		"n":               defaultN(req.N),
		"response_format": "b64_json",
	}
	if req.Size != "" {
		body["size"] = req.Size
	}

	result, err := s.callImageAPI(cfg, body)
	if err != nil {
		log.Printf("[image] images/generations failed: %v, falling back to chat/completions", err)
		// Fallback: try chat completions streaming (for gateways like palebluedot)
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
	if cfg.APIURL == "" {
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

	body := map[string]interface{}{
		"prompt": req.Prompt,
		"model":  req.Model,
		"image":  imageDataURL,
		"n":      defaultN(req.N),
	}
	if req.Size != "" {
		body["size"] = req.Size
	}

	result, err := s.callImageAPI(cfg, body)
	if err != nil {
		// Fallback: try chat completions streaming (for gateways like palebluedot)
		// Note: for image-to-image, we include the image URL in the prompt
		log.Printf("[image] GenerateWithImage fallback, image data URL len=%d, mime=%s", len(imageDataURL), mimeType)
		req.Prompt = req.Prompt + "\n\n[Image: " + imageDataURL + "]"
		return s.callChatCompletions(cfg, req)
	}
	return result, nil
}

func (s *ImageService) callImageAPI(cfg model.ImageConfig, body map[string]interface{}) (*model.ImageGenResp, error) {
	url := strings.TrimRight(cfg.APIURL, "/") + "/images/generations"

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

	var openaiResp struct {
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
			// 如果返回的是 base64，包装成 data URL
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
	url := strings.TrimRight(cfg.APIURL, "/") + "/chat/completions"

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
			continue // skip non-image chunks
		}

		if len(chunk.Choices) > 0 {
			for _, img := range chunk.Choices[0].Delta.Images {
				url := img.ImageURL.URL
				if strings.HasPrefix(url, "data:image/") {
					log.Printf("[image] found image data URL, len=%d", len(url))
					// Keep full data URL including MIME type (e.g. webp, png)
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

	// b64Chunks now contains full data URLs with correct MIME types
	b64Data := strings.Join(b64Chunks, "")
	return &model.ImageGenResp{
		Created: time.Now().Unix(),
		Data: []model.ImageGenResult{
			{URL: b64Data},
		},
	}, nil
}

func defaultN(n int) int {
	if n <= 0 || n > 10 {
		return 1
	}
	return n
}
