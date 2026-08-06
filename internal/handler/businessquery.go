package handler

import (
	"net/http"

	"devtoolbox/internal/model"
	"devtoolbox/internal/service"
)

type BusinessQueryHandler struct {
	svc *service.BusinessQueryService
}

func NewBusinessQueryHandler(connStore *service.ConnStore) *BusinessQueryHandler {
	return &BusinessQueryHandler{svc: service.NewBusinessQueryService(connStore)}
}

func (h *BusinessQueryHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/business-query/tokenrouter/user-overview", OnlyMethod(http.MethodPost, h.TokenRouterUserOverview))
}

func (h *BusinessQueryHandler) TokenRouterUserOverview(w http.ResponseWriter, r *http.Request) {
	var req model.BusinessQueryReq
	if err := BindJSON(r, &req); err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.svc.TokenRouterUserOverview(r.Context(), req)
	if err != nil {
		Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	OK(w, result)
}
