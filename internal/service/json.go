package service

import (
	"encoding/json"
	"strconv"
)

type JSONService struct{}

func NewJSONService() *JSONService {
	return &JSONService{}
}

func (s *JSONService) Format(input, mode string) (string, error) {
	var raw interface{}
	if err := json.Unmarshal([]byte(input), &raw); err != nil {
		return "", err
	}

	var out []byte
	var err error
	switch mode {
	case "compact":
		out, err = json.Marshal(raw)
	case "escape":
		b, err := json.Marshal(raw)
		if err != nil {
			return "", err
		}
		out = []byte(strconv.Quote(string(b)))
	default:
		out, err = json.MarshalIndent(raw, "", "  ")
	}
	if err != nil {
		return "", err
	}
	return string(out), nil
}
