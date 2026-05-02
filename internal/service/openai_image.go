package service

import (
	"context"
	"fmt"
	"mime/multipart"
	"strings"

	"devtoolbox/internal/model"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const openAIImageAPIBaseURL = "https://api.openai.com/v1"

var openAIGenerateSizes = map[string]map[string]struct{}{
	"gpt-image": {"1024x1024": {}, "1024x1536": {}, "1536x1024": {}},
	"dall-e-3":  {"1024x1024": {}, "1024x1792": {}, "1792x1024": {}},
	"dall-e-2":  {"256x256": {}, "512x512": {}, "1024x1024": {}},
}

var openAIEditSizes = map[string]map[string]struct{}{
	"gpt-image": {"1024x1024": {}, "1024x1536": {}, "1536x1024": {}},
	"dall-e-2":  {"256x256": {}, "512x512": {}, "1024x1024": {}},
}

func (s *ImageService) generateImageWithOpenAI(cfg model.ImageConfig, req model.ImageGenReq) (*model.ImageGenResp, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("API Key 未配置，请先保存配置")
	}

	modelName := normalizeImageModelName(req.Model)
	apiModelName := effectiveImageModelName(cfg, req.Model)
	size, err := validateOpenAIImageGenerateSize(modelName, req.Size)
	if err != nil {
		return nil, err
	}

	params := openai.ImageGenerateParams{
		Prompt: req.Prompt,
		Model:  openai.ImageModel(apiModelName),
		N:      openai.Int(int64(defaultN(req.N))),
	}
	if size != "" {
		params.Size = openai.ImageGenerateParamsSize(size)
	}
	if strings.HasPrefix(modelName, "dall-e-") {
		params.ResponseFormat = openai.ImageGenerateParamsResponseFormatB64JSON
	}

	ctx, cancel := context.WithTimeout(context.Background(), imageRequestTimeout)
	defer cancel()

	client := s.newOpenAIClient(cfg)
	resp, err := client.Images.Generate(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI images generate failed: %w", err)
	}
	return toImageGenResp(resp), nil
}

func (s *ImageService) editImageWithOpenAI(cfg model.ImageConfig, file multipart.File, req model.ImageGenReq) (*model.ImageGenResp, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("API Key 未配置，请先保存配置")
	}

	modelName := normalizeImageModelName(req.Model)
	apiModelName := effectiveImageModelName(cfg, req.Model)
	size, err := validateOpenAIImageEditSize(modelName, req.Size)
	if err != nil {
		return nil, err
	}

	params := openai.ImageEditParams{
		Image:  openai.ImageEditParamsImageUnion{OfFile: file},
		Prompt: req.Prompt,
		Model:  openai.ImageModel(apiModelName),
		N:      openai.Int(int64(defaultN(req.N))),
	}
	if size != "" {
		params.Size = openai.ImageEditParamsSize(size)
	}
	if modelName == string(openai.ImageModelDallE2) {
		params.ResponseFormat = openai.ImageEditParamsResponseFormatB64JSON
	}

	ctx, cancel := context.WithTimeout(context.Background(), imageRequestTimeout)
	defer cancel()

	client := s.newOpenAIClient(cfg)
	resp, err := client.Images.Edit(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI images edit failed: %w", err)
	}
	return toImageGenResp(resp), nil
}

func (s *ImageService) newOpenAIClient(cfg model.ImageConfig) openai.Client {
	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
		option.WithHTTPClient(s.client),
	}

	baseURL := normalizeImageAPIBaseURL(cfg.APIURL)
	if baseURL == "" {
		baseURL = openAIImageAPIBaseURL
	}
	opts = append(opts, option.WithBaseURL(baseURL))

	return openai.NewClient(opts...)
}

func effectiveImageModelName(cfg model.ImageConfig, modelName string) string {
	modelName = strings.TrimSpace(modelName)
	if normalizeImageAPIBaseURL(cfg.APIURL) == openAIImageAPIBaseURL {
		return normalizeImageModelName(modelName)
	}
	return modelName
}

func toImageGenResp(resp *openai.ImagesResponse) *model.ImageGenResp {
	if resp == nil {
		return &model.ImageGenResp{Data: []model.ImageGenResult{}}
	}

	format := "png"
	if resp.OutputFormat != "" {
		format = string(resp.OutputFormat)
	}

	result := &model.ImageGenResp{
		Created: resp.Created,
		Data:    make([]model.ImageGenResult, 0, len(resp.Data)),
	}
	for _, item := range resp.Data {
		url := item.URL
		if url == "" && item.B64JSON != "" {
			url = "data:image/" + format + ";base64," + item.B64JSON
		}
		result.Data = append(result.Data, model.ImageGenResult{
			URL:           url,
			RevisedPrompt: item.RevisedPrompt,
		})
	}
	return result
}

func validateOpenAIImageGenerateSize(modelName, size string) (string, error) {
	return validateOpenAIImageSize(modelName, size, openAIGenerateSizes, "文生图")
}

func validateOpenAIImageEditSize(modelName, size string) (string, error) {
	return validateOpenAIImageSize(modelName, size, openAIEditSizes, "图生图")
}

func validateOpenAIImageSize(modelName, size string, supported map[string]map[string]struct{}, op string) (string, error) {
	size = strings.TrimSpace(size)
	if size == "" {
		return "", nil
	}

	family := openAIImageModelFamily(modelName)
	if family == "" {
		return "", fmt.Errorf("暂不支持的 OpenAI 图片模型: %s", modelName)
	}
	allowed, ok := supported[family]
	if !ok {
		return "", fmt.Errorf("%s 暂不支持模型 %s", op, modelName)
	}
	if _, ok := allowed[size]; !ok {
		return "", fmt.Errorf("模型 %s 不支持尺寸 %s", modelName, size)
	}
	return size, nil
}

func openAIImageModelFamily(modelName string) string {
	switch {
	case isGPTImageModel(modelName):
		return "gpt-image"
	case modelName == string(openai.ImageModelDallE3):
		return "dall-e-3"
	case modelName == string(openai.ImageModelDallE2):
		return "dall-e-2"
	default:
		return ""
	}
}
