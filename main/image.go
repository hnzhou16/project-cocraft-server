package main

import (
	"fmt"
	"net/http"
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
