package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"devtoolbox/internal/model"
)

const imageConfigFile = "image_config.json"

// ImageConfigStore 图片 API 配置的持久化管理
type ImageConfigStore struct {
	mu     sync.RWMutex
	path   string
	cache  model.ImageConfig
	loaded bool
}

// NewImageConfigStore 创建配置存储，使用运行时数据目录
func NewImageConfigStore() *ImageConfigStore {
	dataDir := getDataDir()
	return &ImageConfigStore{path: filepath.Join(dataDir, imageConfigFile)}
}

func getDataDir() string {
	if d := os.Getenv("DATA_DIR"); d != "" {
		return d
	}
	return "./data"
}

// Load 读取配置，无文件时返回默认空配置（带预设模型）
func (s *ImageConfigStore) Load() (model.ImageConfig, error) {
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

	cfg, err := loadImageConfig(s.path)
	if err != nil {
		return model.ImageConfig{}, err
	}
	s.cache = cfg
	s.loaded = true
	return cfg, nil
}

// Save 保存配置到文件
func (s *ImageConfigStore) Save(cfg model.ImageConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := saveImageConfig(s.path, cfg); err != nil {
		return err
	}
	s.cache = cfg
	s.loaded = true
	return nil
}

func loadImageConfig(path string) (model.ImageConfig, error) {
	var cfg model.ImageConfig
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return defaultImageConfig(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return model.ImageConfig{}, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return model.ImageConfig{}, err
	}
	if cfg.Models == nil {
		cfg.Models = []string{}
	}
	return cfg, nil
}

func saveImageConfig(path string, cfg model.ImageConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func defaultImageConfig() model.ImageConfig {
	return model.ImageConfig{
		APIURL: "",
		APIKey: "",
		Models: []string{"gpt-5.4-image-2"},
	}
}
