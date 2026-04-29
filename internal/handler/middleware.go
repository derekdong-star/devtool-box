package handler

import (
	"net/http"
	"strings"

	"devtoolbox/internal/service"
)

func RequireAuth(auth *service.AuthService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !auth.Enabled() || isPublicAuthPath(r.URL.Path) || auth.Authenticated(r) {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			Fail(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		http.Redirect(w, r, "/login", http.StatusFound)
	})
}

func isPublicAuthPath(path string) bool {
	return path == "/login" || path == "/api/auth/login" || strings.HasPrefix(path, "/static/")
}
