package handler

import "net/http"

// Handler 定义功能模块路由注册接口
type Handler interface {
	Register(mux *http.ServeMux)
}
