package main

import "net/http"

type imageRequest struct {
	Prompt string `json:"prompt" validate:"required,min=1,max=500"`
}

func (app *application) generateImageHandler(w http.ResponseWriter, r *http.Request) {
	var payload imageRequest

	if err := ReadJSON(w, r, &payload); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	imageUrl, err := app.aiImage.GenerateImage(payload.Prompt)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	app.OutputJSON(w, http.StatusOK, map[string]string{"image_url": imageUrl})
}
