package handler

import (
	"net/http"

	"devtoolbox/internal/model"
	"devtoolbox/internal/service"
)

type CookieHandler struct {
	cookieSvc  *service.CookieService
	jwtSvc     *service.JWTService
	sessionSvc *service.SessionService
}

func NewCookieHandler() *CookieHandler {
	return &CookieHandler{
		cookieSvc:  service.NewCookieService(),
		jwtSvc:     service.NewJWTService(),
		sessionSvc: service.NewSessionService(),
	}
}

func (h *CookieHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/parse-cookie", OnlyMethod(http.MethodPost, h.ParseCookie))
	mux.HandleFunc("/api/parse-jwt", OnlyMethod(http.MethodPost, h.ParseJWT))
	mux.HandleFunc("/api/parse-session", OnlyMethod(http.MethodPost, h.ParseSession))
}

func (h *CookieHandler) ParseCookie(w http.ResponseWriter, r *http.Request) {
	var req model.CookieParseReq
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	OK(w, h.cookieSvc.Parse(req.Cookie))
}

func (h *CookieHandler) ParseJWT(w http.ResponseWriter, r *http.Request) {
	var req model.JWTDecodeReq
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.jwtSvc.Decode(req.Token)
	if err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	OK(w, resp)
}

func (h *CookieHandler) ParseSession(w http.ResponseWriter, r *http.Request) {
	var req model.SessionParseReq
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.sessionSvc.Parse(req.Cookie, req.Secret)
	if err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	OK(w, result)
}
