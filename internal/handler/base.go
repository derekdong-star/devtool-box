package handler

import (
	"encoding/json"
	"net/http"

	"devtoolbox/pkg/response"
)

func BindJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func OK(w http.ResponseWriter, data interface{}) {
	response.OK(w, data)
}

func Fail(w http.ResponseWriter, status int, err string) {
	response.Fail(w, status, err)
}

// OnlyMethod 限制 HTTP 方法的中间件，不匹配时返回 405 (L-6)
func OnlyMethod(method string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h(w, r)
	}
}
