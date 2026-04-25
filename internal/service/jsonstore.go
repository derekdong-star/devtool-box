package service

import (
	"encoding/json"
	"os"
)

// jsonStore 提供基于本地 JSON 文件的泛型切片持久化。
// 用于消除 ConnStore 与 CmdTemplateStore 中重复的 load/write 逻辑。
// 注意：jsonStore 本身不加锁，调用方（如 ConnStore）负责保证并发安全。
type jsonStore[T any] struct {
	path string
}

func newJSONStore[T any](path string) *jsonStore[T] {
	return &jsonStore[T]{path: path}
}

func (s *jsonStore[T]) load() ([]T, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return []T{}, nil
	}
	if err != nil {
		return nil, err
	}
	var items []T
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *jsonStore[T]) save(items []T) error {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}
