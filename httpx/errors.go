package httpx

import (
	"encoding/json"
	"net/http"
	"time"
)

const RequestTimeout = 3 * time.Second

type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func WriteError(w http.ResponseWriter, status int, message, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: message,
		Code:  code,
	})
}
