package response

import (
	"encoding/json"
	"net/http"
)

type Result struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

func JSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func OK(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, Result{Code: 0, Data: data})
}

func Fail(w http.ResponseWriter, status int, err string) {
	JSON(w, status, Result{Code: status, Msg: err})
}
