package main

import (
	"net/http"
)

func (app *application) internalServerError(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Errorw("internal server error", "method", r.Method, "path", r.URL.Path, "error", err.Error())
	WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}

func (app *application) badRequestError(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Errorw("bad request error", "method", r.Method, "path", r.URL.Path, "error", err.Error())
	WriteJSONError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
}

func (app *application) notFoundError(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Errorw("not found error", "method", r.Method, "path", r.URL.Path, "error", err.Error())
	WriteJSONError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
}

func (app *application) conflictError(w http.ResponseWriter, r *http.Request, code string, err error) {
	app.logger.Errorw("conflict error", "method", r.Method, "path", r.URL.Path, "error", err.Error())
	WriteJSONError(w, http.StatusConflict, code, err.Error())
}

func (app *application) unauthorizedError(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Errorw("unauthorized error", "method", r.Method, "path", r.URL.Path, "error", err.Error())
	WriteJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
}

func (app *application) forbiddenError(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Errorw("unauthorized error", "method", r.Method, "path", r.URL.Path, "error", err.Error())
	WriteJSONError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
}
