package app

import (
	"fmt"
	"net/http"

	"devtoolbox/internal/handler"
	"devtoolbox/internal/service"
)

type App struct {
	mux *http.ServeMux
}

// New 组装所有模块并返回 App 实例。
// M-9: 去掉无意义的 error 返回——当前所有操作均不会失败，
// 未来若需加载配置文件等可能失败的操作时再恢复 error。
func New() *App {
	mux := http.NewServeMux()

	// 页面与静态资源
	handler.NewPageHandler().Register(mux)

	// 共享连接存储
	connStore    := service.NewConnStore()
	templateStore := service.NewCmdTemplateStore()

	// 功能模块注册：新增模块只需在此追加一行
	modules := []handler.Handler{
		handler.NewCookieHandler(),
		handler.NewJSONHandler(),
		handler.NewCodecHandler(),
		handler.NewDBHandler(connStore),
		handler.NewRedisHandler(connStore),
		handler.NewConnStoreHandler(connStore),
		handler.NewCmdTemplateHandler(templateStore),
		handler.NewUtilHandler(),
	}
	for _, m := range modules {
		m.Register(mux)
	}

	return &App{mux: mux}
}

func (a *App) Run(addr string) error {
	fmt.Printf("DevToolbox running at http://localhost%s\n", addr)
	return http.ListenAndServe(addr, a.mux)
}
