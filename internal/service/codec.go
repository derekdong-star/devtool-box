package service

import (
	"encoding/base64"
	"net/url"
)

type CodecService struct{}

func NewCodecService() *CodecService {
	return &CodecService{}
}

func (s *CodecService) Base64(text, mode string) (string, error) {
	if mode == "b64enc" {
		return base64.StdEncoding.EncodeToString([]byte(text)), nil
	}
	b, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *CodecService) URL(text, mode string) (string, error) {
	if mode == "urlenc" {
		return url.QueryEscape(text), nil
	}
	return url.QueryUnescape(text)
}
