package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
)

var allowedExtensions = map[string]bool{
	"jpeg": true,
	"jpg":  true,
	"png":  true,
	"gif":  true,
	"webp": true,
}

type PresignURLResponse struct {
	UploadURL string `json:"upload_url"`
	S3Key     string `json:"s3_key"`
}

func (app *application) generateUploadURLHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getUserFromCtx(r)
	userIDStr := user.ID.Hex()

	var req struct {
		Extension string `json:"extension"`
	}

	if err := ReadJSON(w, r, &req); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	if _, valid := allowedExtensions[req.Extension]; !valid {
		app.badRequestError(w, r, fmt.Errorf("extension '%s' is not allowed", req.Extension))
		return
	}

	presignedReq, key, err := app.awsPresigner.GenerateUploadURL(
		ctx,
		userIDStr,
		req.Extension,
		app.config.awsConfig.exp,
	)
	if err != nil {
		app.internalServerError(w, r, fmt.Errorf("failed to generate upload URL: %v", err))
		return
	}

	response := &PresignURLResponse{
		UploadURL: presignedReq.URL,
		S3Key:     key,
	}

	app.OutputJSON(w, http.StatusCreated, response)
}

func (app *application) deleteImageHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getUserFromCtx(r)
	userIDStr := user.ID.Hex()

	var req struct {
		ObjectKey string `json:"key"`
	}

	if err := ReadJSON(w, r, &req); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	log.Println(req.ObjectKey)

	// check if key starts with user's folder
	if !strings.HasPrefix(req.ObjectKey, fmt.Sprintf("user_uploads/%s/", userIDStr)) {
		app.unauthorizedError(w, r, errors.New("object key is not the correct format"))
		return
	}

	err := app.awsPresigner.DeleteImage(ctx, req.ObjectKey)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.OutputJSON(w, http.StatusOK, nil)
}
