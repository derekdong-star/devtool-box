package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const authCookieName = "dtb_auth"

type AuthService struct {
	user      string
	password  string
	secret    []byte
	sessionTTL time.Duration
}

type authClaims struct {
	User string `json:"user"`
	Exp  int64  `json:"exp"`
}

func NewAuthServiceFromEnv() *AuthService {
	user := os.Getenv("AUTH_USER")
	password := os.Getenv("AUTH_PASSWORD")
	secret := os.Getenv("SESSION_SECRET")
	if user == "" || password == "" || secret == "" {
		return &AuthService{}
	}
	ttl := 12 * time.Hour
	if raw := os.Getenv("AUTH_SESSION_TTL"); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			ttl = parsed
		}
	}
	return &AuthService{user: user, password: password, secret: []byte(secret), sessionTTL: ttl}
}

func (s *AuthService) Enabled() bool {
	return s.user != "" && s.password != "" && len(s.secret) > 0
}

func (s *AuthService) Login(user, password string) (*http.Cookie, error) {
	if !s.Enabled() {
		return nil, errors.New("auth is disabled")
	}
	if !constantTimeEqual(user, s.user) || !constantTimeEqual(password, s.password) {
		return nil, errors.New("invalid username or password")
	}
	expiresAt := time.Now().Add(s.sessionTTL)
	claims := authClaims{User: user, Exp: expiresAt.Unix()}
	value, err := s.signClaims(claims)
	if err != nil {
		return nil, err
	}
	return &http.Cookie{
		Name:     authCookieName,
		Value:    value,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(s.sessionTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}, nil
}

func (s *AuthService) ClearCookie() *http.Cookie {
	return &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func (s *AuthService) Authenticated(r *http.Request) bool {
	if !s.Enabled() {
		return true
	}
	cookie, err := r.Cookie(authCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	claims, err := s.verifyCookie(cookie.Value)
	if err != nil {
		return false
	}
	return claims.User == s.user && time.Now().Unix() < claims.Exp
}

func (s *AuthService) signClaims(claims authClaims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)
	sig := s.sign(payloadB64)
	return payloadB64 + "." + sig, nil
}

func (s *AuthService) verifyCookie(value string) (authClaims, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return authClaims{}, errors.New("invalid auth cookie")
	}
	if !constantTimeEqual(parts[1], s.sign(parts[0])) {
		return authClaims{}, errors.New("invalid auth signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return authClaims{}, err
	}
	var claims authClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return authClaims{}, err
	}
	return claims, nil
}

func (s *AuthService) sign(payload string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func constantTimeEqual(a, b string) bool {
	return hmac.Equal([]byte(a), []byte(b))
}

func AuthStatusFromEnv() string {
	if os.Getenv("AUTH_USER") == "" && os.Getenv("AUTH_PASSWORD") == "" && os.Getenv("SESSION_SECRET") == "" {
		return "disabled"
	}
	missing := make([]string, 0)
	for _, key := range []string{"AUTH_USER", "AUTH_PASSWORD", "SESSION_SECRET"} {
		if os.Getenv(key) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return fmt.Sprintf("disabled: missing %s", strings.Join(missing, ","))
	}
	if raw := os.Getenv("AUTH_SESSION_TTL"); raw != "" {
		if _, err := time.ParseDuration(raw); err != nil {
			return "enabled: invalid AUTH_SESSION_TTL " + strconv.Quote(raw) + ", fallback to 12h"
		}
	}
	return "enabled"
}
