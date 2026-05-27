package service

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	cos "github.com/tencentyun/cos-go-sdk-v5"

	"devtoolbox/internal/model"
)

const (
	MaxUploadFileSize     = 50 << 20
	maxUploadBodyOverhead = 1 << 20
	MaxUploadRequestSize  = MaxUploadFileSize + maxUploadBodyOverhead
	uploadTimeout         = 5 * time.Minute
)

type UploadService struct {
	store *UploadConfigStore
}

type UploadFileInput struct {
	FileName     string
	Directory    string
	ObjectName   string
	ContentType  string
	UseSignedURL bool
	Content      []byte
}

func NewUploadService(store *UploadConfigStore) *UploadService {
	return &UploadService{store: store}
}

func (s *UploadService) ConfigStore() *UploadConfigStore {
	return s.store
}

func (s *UploadService) Upload(ctx context.Context, input UploadFileInput) (model.UploadResult, error) {
	cfg, err := s.store.Load()
	if err != nil {
		return model.UploadResult{}, fmt.Errorf("读取上传配置失败: %w", err)
	}
	if err := validateUploadConfig(cfg); err != nil {
		return model.UploadResult{}, err
	}
	if len(input.Content) == 0 {
		return model.UploadResult{}, fmt.Errorf("文件内容为空")
	}
	if len(input.Content) > MaxUploadFileSize {
		return model.UploadResult{}, fmt.Errorf("文件过大，最大支持 %dMB", MaxUploadFileSize>>20)
	}

	contentType := detectContentType(input.ContentType, input.FileName, input.Content)
	objectKey := buildObjectKey(cfg.PathPrefix, input.Directory, input.ObjectName, input.FileName)
	client, bucketURL, err := newCOSClient(cfg)
	if err != nil {
		return model.UploadResult{}, err
	}

	putOpt := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentType: contentType,
		},
	}

	uploadCtx, cancel := context.WithTimeout(ctx, uploadTimeout)
	defer cancel()

	if _, err := client.Object.Put(uploadCtx, objectKey, bytes.NewReader(input.Content), putOpt); err != nil {
		return model.UploadResult{}, fmt.Errorf("COS 上传失败: %w", err)
	}

	resourceURL, err := resolveObjectURL(client, cfg, bucketURL, objectKey, input.UseSignedURL)
	if err != nil {
		return model.UploadResult{}, err
	}

	return model.UploadResult{
		Key:          objectKey,
		URL:          resourceURL,
		Size:         int64(len(input.Content)),
		ContentType:  contentType,
		OriginalName: input.FileName,
	}, nil
}

func validateUploadConfig(cfg model.UploadConfig) error {
	switch {
	case cfg.SecretID == "":
		return fmt.Errorf("COS SecretId 未配置")
	case cfg.SecretKey == "":
		return fmt.Errorf("COS SecretKey 未配置")
	case cfg.Bucket == "":
		return fmt.Errorf("COS Bucket 未配置")
	case cfg.Region == "":
		return fmt.Errorf("COS Region 未配置")
	default:
		return nil
	}
}

func newCOSClient(cfg model.UploadConfig) (*cos.Client, *url.URL, error) {
	host := strings.TrimSpace(cfg.Domain)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimRight(host, "/")
	if host == "" {
		host = fmt.Sprintf("%s.cos.%s.myqcloud.com", cfg.Bucket, cfg.Region)
	}

	bucketURL, err := url.Parse("https://" + host)
	if err != nil {
		return nil, nil, fmt.Errorf("无效 COS 域名配置: %w", err)
	}

	base := &cos.BaseURL{BucketURL: bucketURL}
	client := cos.NewClient(base, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  cfg.SecretID,
			SecretKey: cfg.SecretKey,
		},
	})
	return client, bucketURL, nil
}

func resolveObjectURL(client *cos.Client, cfg model.UploadConfig, bucketURL *url.URL, key string, forceSigned bool) (string, error) {
	if cfg.PublicBaseURL != "" {
		return cfg.PublicBaseURL + "/" + quoteObjectKey(key), nil
	}
	if forceSigned || cfg.UseSignedURL {
		presignedURL, err := client.Object.GetPresignedURL(
			context.Background(),
			http.MethodGet,
			key,
			cfg.SecretID,
			cfg.SecretKey,
			time.Duration(cfg.SignedURLExpireSecs)*time.Second,
			nil,
		)
		if err != nil {
			return "", fmt.Errorf("生成预签名 URL 失败: %w", err)
		}
		return presignedURL.String(), nil
	}
	return strings.TrimRight(bucketURL.String(), "/") + "/" + quoteObjectKey(key), nil
}

func buildObjectKey(prefix, directory, objectName, originalName string) string {
	now := time.Now()
	parts := make([]string, 0, 4)
	if prefix != "" {
		parts = append(parts, strings.Trim(prefix, "/"))
	}
	if dir := strings.Trim(directory, "/ "); dir != "" {
		parts = append(parts, sanitizePathPart(dir))
	}
	parts = append(parts, now.Format("20060102"))

	filename := strings.TrimSpace(objectName)
	if filename == "" {
		filename = strings.TrimSpace(originalName)
	}
	parts = append(parts, buildStoredFileName(filename))
	return path.Join(parts...)
}

func buildStoredFileName(name string) string {
	name = path.Base(strings.TrimSpace(name))
	ext := strings.ToLower(filepath.Ext(name))
	base := strings.TrimSuffix(name, filepath.Ext(name))
	base = sanitizeFilePart(base)
	if base == "" {
		base = "file"
	}
	if ext == "" {
		return base + "-" + uuid.NewString()
	}
	return base + "-" + uuid.NewString() + ext
}

func sanitizePathPart(value string) string {
	replacer := strings.NewReplacer("\\", "/", "..", "", " ", "-", ":", "-", "*", "", "?", "", "\"", "", "<", "", ">", "", "|", "")
	value = replacer.Replace(strings.TrimSpace(value))
	segments := strings.FieldsFunc(value, func(r rune) bool { return r == '/' })
	cleaned := make([]string, 0, len(segments))
	for _, seg := range segments {
		seg = strings.Trim(seg, ".-_")
		if seg != "" {
			cleaned = append(cleaned, seg)
		}
	}
	return strings.Join(cleaned, "/")
}

func sanitizeFilePart(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer("/", "-", "\\", "-", "..", "", " ", "-", ":", "-", "*", "", "?", "", "\"", "", "<", "", ">", "", "|", "")
	value = replacer.Replace(value)
	value = strings.Trim(value, ".-_")
	return value
}

func detectContentType(contentType, fileName string, content []byte) string {
	if t := strings.TrimSpace(contentType); t != "" {
		return t
	}
	if ext := strings.ToLower(filepath.Ext(fileName)); ext != "" {
		if byExt := mime.TypeByExtension(ext); byExt != "" {
			return byExt
		}
	}
	return http.DetectContentType(content)
}

func quoteObjectKey(key string) string {
	parts := strings.Split(key, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}
