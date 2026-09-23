package api

import (
	"encoding/json"
	"log"
	"net/http"
)

type errorBody struct {
	Error string `json:"error"`
}

func OKResponse(w http.ResponseWriter, data any) {
	JSONResponse(w, http.StatusOK, data)
}

func ErrorResponse(w http.ResponseWriter, status int, message string) {
	JSONResponse(w, status, errorBody{Error: message})
}

// InternalErrorResponse logs err and responds with a generic 500 so internal
// details such as DB errors never reach the client.
func InternalErrorResponse(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	ErrorResponse(w, http.StatusInternalServerError, "internal server error")
}

func JSONResponse(w http.ResponseWriter, status int, data any) {
	body, err := json.Marshal(data)
	if err != nil {
		log.Printf("encoding response failed: %v", err)
		status = http.StatusInternalServerError
		body = []byte(`{"error":"internal server error"}`)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		log.Printf("writing response failed: %v", err)
	}
}
