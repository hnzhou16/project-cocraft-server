package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type errResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func statusAllowsBody(status int) bool {
	if status >= 100 && status < 200 || status == http.StatusNoContent || status == http.StatusNotModified {
		return false
	}
	return true
}

// WriteJSON - no need to return error (just log it), since http response already send
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if !statusAllowsBody(status) {
		w.WriteHeader(status)
		return
	}

	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Println("Failed to write JSON response: ", err)
	}
}

// ReadJSON - read and decode JSON from request body into the provided data struct
// there could be error due to bad formatting, return it
func ReadJSON(w http.ResponseWriter, r *http.Request, data any) error {
	maxBytes := 1024 * 1024 // 1MB
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	decoder := json.NewDecoder(r.Body) // create decoder to read from request body
	decoder.DisallowUnknownFields()    // fail if extra fields in request body

	err := decoder.Decode(data) // decode into data struct
	if err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func WriteJSONError(w http.ResponseWriter, status int, code string, message string) {
	type envelope struct {
		Error errResponse `json:"error"`
	}
	WriteJSON(w, status, envelope{Error: errResponse{
		Code:    code,
		Message: message,
	}})
}

func (app *application) OutputJSON(w http.ResponseWriter, status int, data any) {
	type envelope struct {
		Data any `json:"data"`
	}
	WriteJSON(w, status, &envelope{Data: data})
}
