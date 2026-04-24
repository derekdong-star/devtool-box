package service

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"devtoolbox/internal/model"
)

type JWTService struct{}

func NewJWTService() *JWTService {
	return &JWTService{}
}

func (s *JWTService) Decode(token string) (*model.JWTDecodeResp, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT format")
	}

	decode := func(s string) (map[string]interface{}, error) {
		if i := len(s) % 4; i != 0 {
			s += strings.Repeat("=", 4-i)
		}
		b, err := base64.URLEncoding.DecodeString(s)
		if err != nil {
			return nil, err
		}
		var out map[string]interface{}
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, err
		}
		return out, nil
	}

	header, err := decode(parts[0])
	if err != nil {
		return nil, err
	}
	payload, err := decode(parts[1])
	if err != nil {
		return nil, err
	}
	return &model.JWTDecodeResp{
		Header:    header,
		Payload:   payload,
		Signature: parts[2],
	}, nil
}
