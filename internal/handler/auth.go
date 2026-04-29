package handler

import (
	"net/http"
	"strings"

	"devtoolbox/internal/model"
	"devtoolbox/internal/service"
	"devtoolbox/internal/web"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/login", h.LoginPage)
	mux.HandleFunc("/api/auth/login", OnlyMethod(http.MethodPost, h.Login))
	mux.HandleFunc("/api/auth/logout", OnlyMethod(http.MethodPost, h.Logout))
}

func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/login" {
		http.NotFound(w, r)
		return
	}
	if h.auth.Authenticated(r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	b, err := web.FS.ReadFile("login.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(b)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginReq
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	cookie, err := h.auth.Login(strings.TrimSpace(req.User), req.Password)
	if err != nil {
		Fail(w, http.StatusUnauthorized, err.Error())
		return
	}
	http.SetCookie(w, cookie)
	OK(w, map[string]string{"user": req.User})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, h.auth.ClearCookie())
	OK(w, nil)
}
