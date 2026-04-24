package service

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/securecookie"
)

type SessionService struct{}

func NewSessionService() *SessionService {
	return &SessionService{}
}

// Parse 从完整的 Cookie header 字符串中提取并解析 session 字段。
// sessionSecret 可选：
//   - 有值时：先用 gorilla/securecookie 验签解码（生产模式）
//   - 空值时：跳过验签，直接 base64url + gob 解码（调试模式）
func (s *SessionService) Parse(cookieHeader, sessionSecret string) (map[string]any, error) {
	// L-8: 使用标准 http.Header + http.Request 替换测试包 httptest
	header := http.Header{}
	header.Add("Cookie", cookieHeader)
	req := &http.Request{Header: header}

	cookie, err := req.Cookie("session")
	if err != nil {
		return nil, fmt.Errorf("session cookie not found: %w", err)
	}

	// 有 secret，走验签路径
	if sessionSecret != "" {
		var verified map[interface{}]interface{}
		codecs := securecookie.CodecsFromPairs([]byte(sessionSecret))
		if err := securecookie.DecodeMulti("session", cookie.Value, &verified, codecs...); err == nil {
			return normalizeMap(verified), nil
		}
	}

	// 无 secret 或验签失败，走纯解码路径
	// 格式：base64url(timestamp|base64url(gob_payload)|signature)
	outer, err := decodeBase64URL(cookie.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to decode session outer base64: %w", err)
	}

	parts := bytes.SplitN(outer, []byte("|"), 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("unexpected session format: expected 3 parts, got %d", len(parts))
	}

	payload, err := decodeBase64URL(string(parts[1]))
	if err != nil {
		return nil, fmt.Errorf("failed to decode session payload base64: %w", err)
	}

	var raw map[interface{}]interface{}
	if err := gob.NewDecoder(bytes.NewReader(payload)).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to gob-decode session payload: %w", err)
	}

	return normalizeMap(raw), nil
}

func decodeBase64URL(value string) ([]byte, error) {
	if mod := len(value) % 4; mod != 0 {
		value += strings.Repeat("=", 4-mod)
	}
	return base64.URLEncoding.DecodeString(value)
}

func normalizeMap(values map[interface{}]interface{}) map[string]any {
	out := make(map[string]any, len(values))
	for k, v := range values {
		out[fmt.Sprint(k)] = normalizeValue(v)
	}
	return out
}

func normalizeValue(value any) any {
	switch typed := value.(type) {
	case map[interface{}]interface{}:
		return normalizeMap(typed)
	case []interface{}:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, normalizeValue(item))
		}
		return out
	default:
		return value
	}
}
