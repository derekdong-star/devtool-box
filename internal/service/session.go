package service

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
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
	cookieName, cookieValue, err := extractSessionCookie(cookieHeader)
	if err != nil {
		return nil, err
	}

	// 有 secret，走验签路径
	if sessionSecret != "" {
		var verified map[interface{}]interface{}
		codecs := securecookie.CodecsFromPairs([]byte(sessionSecret))
		if err := securecookie.DecodeMulti(cookieName, cookieValue, &verified, codecs...); err == nil {
			return normalizeMap(verified), nil
		}
	}

	// 无 secret 或验签失败，走纯解码路径
	// 格式：base64url(timestamp|base64url(gob_payload)|signature)
	outer, err := decodeBase64URL(cookieValue)
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

var exactSessionCookieNames = map[string]struct{}{
	"session":    {},
	"sessionid":  {},
	"session_id": {},
	"connect.sid": {},
	"connectsid": {},
	"jsessionid": {},
	"phpsessid":  {},
	"sid":        {},
}

var setCookieAttrs = map[string]struct{}{
	"path":       {},
	"domain":     {},
	"expires":    {},
	"max-age":    {},
	"httponly":   {},
	"secure":     {},
	"samesite":   {},
	"partitioned": {},
	"priority":   {},
	"comment":    {},
	"version":    {},
}

type sessionCookieCandidate struct {
	name  string
	value string
}

func extractSessionCookie(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("session cookie not found: empty cookie input")
	}

	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "cookie:"):
		raw = strings.TrimSpace(raw[len("cookie:"):])
	case strings.HasPrefix(lower, "set-cookie:"):
		raw = strings.TrimSpace(raw[len("set-cookie:"):])
	}

	if !strings.Contains(raw, "=") && !strings.Contains(raw, ";") {
		return "session", raw, nil
	}

	var first sessionCookieCandidate
	var candidates []sessionCookieCandidate

	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		name := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		if name == "" {
			continue
		}

		lowerName := strings.ToLower(name)
		if _, ok := setCookieAttrs[lowerName]; ok {
			continue
		}

		item := sessionCookieCandidate{name: name, value: value}
		if first.name == "" {
			first = item
		}
		candidates = append(candidates, item)
		if _, ok := exactSessionCookieNames[lowerName]; ok {
			return item.name, item.value, nil
		}
	}

	for _, item := range candidates {
		lowerName := strings.ToLower(item.name)
		if strings.Contains(lowerName, "session") || strings.HasSuffix(lowerName, "sid") {
			return item.name, item.value, nil
		}
	}

	if first.name != "" {
		return "", "", fmt.Errorf("session cookie not found: available cookies [%s]", joinCookieNames(candidates))
	}

	return "", "", fmt.Errorf("session cookie not found: no valid cookie pair found")
}

func joinCookieNames(candidates []sessionCookieCandidate) string {
	names := make([]string, 0, len(candidates))
	for _, item := range candidates {
		names = append(names, item.name)
	}
	return strings.Join(names, ", ")
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
