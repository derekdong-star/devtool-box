package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"devtoolbox/internal/model"
)

const uploadConfigFile = "upload_config.json"

// UploadConfigStore COS 上传配置持久化
type UploadConfigStore struct {
	mu     sync.RWMutex
	path   string
	cache  model.UploadConfig
	loaded bool
}

func NewUploadConfigStore() *UploadConfigStore {
	return &UploadConfigStore{path: filepath.Join(getDataDir(), uploadConfigFile)}
}

func (s *UploadConfigStore) Load() (model.UploadConfig, error) {
	s.mu.RLock()
	if s.loaded {
		cfg := s.cache
		s.mu.RUnlock()
		return cfg, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loaded {
		return s.cache, nil
	}

	cfg, err := loadUploadConfig(s.path)
	if err != nil {
		return model.UploadConfig{}, err
	}
	s.cache = cfg
	s.loaded = true
	return cfg, nil
}

func (s *UploadConfigStore) Save(cfg model.UploadConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg = normalizeUploadConfig(cfg)
	if err := saveUploadConfig(s.path, cfg); err != nil {
		return err
	}
	s.cache = cfg
	s.loaded = true
	return nil
}

func loadUploadConfig(path string) (model.UploadConfig, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return defaultUploadConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return model.UploadConfig{}, err
	}

	var cfg model.UploadConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return model.UploadConfig{}, err
	}
	return normalizeUploadConfig(cfg), nil
}

func saveUploadConfig(path string, cfg model.UploadConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func defaultUploadConfig() model.UploadConfig {
	return model.UploadConfig{
		PathPrefix:          "uploads",
		UseSignedURL:        false,
		SignedURLExpireSecs: 3600,
	}
}

func normalizeUploadConfig(cfg model.UploadConfig) model.UploadConfig {
	cfg.SecretID = strings.TrimSpace(cfg.SecretID)
	cfg.SecretKey = strings.TrimSpace(cfg.SecretKey)
	cfg.Bucket = strings.TrimSpace(cfg.Bucket)
	cfg.Region = strings.TrimSpace(cfg.Region)
	cfg.Domain = strings.TrimSpace(cfg.Domain)
	cfg.PublicBaseURL = strings.TrimSpace(strings.TrimRight(cfg.PublicBaseURL, "/"))
	cfg.PathPrefix = strings.Trim(strings.TrimSpace(cfg.PathPrefix), "/")
	if cfg.SignedURLExpireSecs <= 0 {
		cfg.SignedURLExpireSecs = 3600
	}
	return cfg
}
