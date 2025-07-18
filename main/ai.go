package main

import "net/http"

type imageRequest struct {
	Prompt      string `json:"prompt" validate:"required,min=1,max=500"`
	Refinement  string `json:"refinement,omitempty" validate:"omitempty,max=300"`
	Iterations  int    `json:"iterations,omitempty" validate:"omitempty,min=1,max=5"`
}

type refineImageRequest struct {
	Prompt      string `json:"prompt" validate:"required,min=1,max=500"`
	Refinement  string `json:"refinement" validate:"required,min=1,max=300"`
	Iterations  int    `json:"iterations" validate:"min=1,max=5"`
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

	var imageUrl string
	var err error

	// Check if refinement is provided - if so, use RefineImage
	if payload.Refinement != "" {
		// Default iterations to 1 if not provided
		if payload.Iterations == 0 {
			payload.Iterations = 1
		}
		
		imageUrl, err = app.aiImage.RefineImage(payload.Prompt, payload.Refinement, payload.Iterations)
		if err != nil {
			app.internalServerError(w, r, err)
			return
		}

		response := map[string]interface{}{
			"image_url":  imageUrl,
			"prompt":     payload.Prompt,
			"refinement": payload.Refinement,
			"iterations": payload.Iterations,
		}
		app.OutputJSON(w, http.StatusOK, response)
	} else {
		// Basic image generation
		imageUrl, err = app.aiImage.GenerateImage(payload.Prompt)
		if err != nil {
			app.internalServerError(w, r, err)
			return
		}

		response := map[string]interface{}{
			"image_url": imageUrl,
			"prompt":    payload.Prompt,
		}
		app.OutputJSON(w, http.StatusOK, response)
	}
}

func (app *application) refineImageHandler(w http.ResponseWriter, r *http.Request) {
	var payload refineImageRequest

	if err := ReadJSON(w, r, &payload); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestError(w, r, err)
		return
	}

	// Default iterations to 1 if not provided
	if payload.Iterations == 0 {
		payload.Iterations = 1
	}

	imageUrl, err := app.aiImage.RefineImage(payload.Prompt, payload.Refinement, payload.Iterations)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	response := map[string]interface{}{
		"image_url":  imageUrl,
		"prompt":     payload.Prompt,
		"refinement": payload.Refinement,
		"iterations": payload.Iterations,
	}

	app.OutputJSON(w, http.StatusOK, response)
}
