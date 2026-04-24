package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"

	"devtoolbox/internal/model"
)

const templateFile = "cmd_templates.json"

// CmdTemplateStore 将命令模板持久化到本地 JSON 文件
type CmdTemplateStore struct {
	mu   sync.RWMutex
	path string
}

func NewCmdTemplateStore() *CmdTemplateStore {
	dir := os.Getenv("DATA_DIR")
	if dir == "" {
		exe, err := os.Executable()
		dir = "."
		if err == nil {
			dir = filepath.Dir(exe)
		}
	}
	return &CmdTemplateStore{path: filepath.Join(dir, templateFile)}
}

// List 返回所有模板，可按 kind 过滤（空字符串表示全部）
func (s *CmdTemplateStore) List(kind string) ([]model.CmdTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all, err := s.load()
	if err != nil {
		return nil, err
	}
	if kind == "" {
		return all, nil
	}
	result := make([]model.CmdTemplate, 0)
	for _, t := range all {
		if t.Kind == kind {
			result = append(result, t)
		}
	}
	return result, nil
}

// Save 新增一条模板；name + command + kind 完全相同时视为重复，直接返回已有记录
func (s *CmdTemplateStore) Save(name, command, kind string) (model.CmdTemplate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name    = strings.TrimSpace(name)
	command = strings.TrimSpace(command)
	if name == "" || command == "" {
		return model.CmdTemplate{}, &validationError{"name and command are required"}
	}

	all, err := s.load()
	if err != nil {
		return model.CmdTemplate{}, err
	}
	// 去重
	for _, t := range all {
		if t.Name == name && t.Command == command && t.Kind == kind {
			return t, nil
		}
	}

	tmpl := model.CmdTemplate{
		ID:      uuid.New().String(),
		Name:    name,
		Command: command,
		Kind:    kind,
	}
	all = append([]model.CmdTemplate{tmpl}, all...)
	return tmpl, s.write(all)
}

// Delete 按 ID 删除
func (s *CmdTemplateStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.load()
	if err != nil {
		return err
	}
	filtered := all[:0]
	for _, t := range all {
		if t.ID != id {
			filtered = append(filtered, t)
		}
	}
	return s.write(filtered)
}

// ── 内部方法 ──────────────────────────────────────────────────

func (s *CmdTemplateStore) load() ([]model.CmdTemplate, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return []model.CmdTemplate{}, nil
	}
	if err != nil {
		return nil, err
	}
	var templates []model.CmdTemplate
	if err := json.Unmarshal(data, &templates); err != nil {
		return nil, err
	}
	return templates, nil
}

func (s *CmdTemplateStore) write(templates []model.CmdTemplate) error {
	data, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// validationError 参数校验错误
type validationError struct{ msg string }

func (e *validationError) Error() string { return e.msg }
