package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client interface {
	GenerateImage(string) (string, error)
	RefineImage(prompt, refinement string, iterations int) (string, error)
}

type ImageGenerator struct {
	apiKey      string
	apiUrl      string
	imageNumber int
	imageSize   string
	httpClient  *http.Client
}

func NewImageGenerator(apiKey, apiUrl, imageSize string, imageNumber int) *ImageGenerator {
	return &ImageGenerator{
		apiKey:      apiKey,
		apiUrl:      apiUrl,
		imageNumber: imageNumber,
		imageSize:   imageSize,
		httpClient:  &http.Client{},
	}
}

func (ig *ImageGenerator) GenerateImage(prompt string) (string, error) {
	requestBody, _ := json.Marshal(map[string]any{
		"prompt": prompt,
		"n":      ig.imageNumber, // num of image generated
		"size":   ig.imageSize,
	})

	req, _ := http.NewRequest("POST", ig.apiUrl, bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ig.apiKey)

	resp, err := ig.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to generate image: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	//log.Println(string(bodyBytes))

	var responseData struct {
		Data []struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &responseData); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(responseData.Data) == 0 {
		return "", fmt.Errorf("failed to generate image: no results returned")
	}

	return responseData.Data[0].URL, nil
}

func (ig *ImageGenerator) RefineImage(prompt, refinement string, iterations int) (string, error) {
	// Combine the original prompt with refinement instructions
	refinedPrompt := fmt.Sprintf("%s. %s", prompt, refinement)

	// For iterations, we can add specific instructions to improve quality
	if iterations > 1 {
		refinedPrompt = fmt.Sprintf("%s. High quality, detailed, refined version (iteration %d)", refinedPrompt, iterations)
	}

	requestBody, _ := json.Marshal(map[string]any{
		"prompt": refinedPrompt,
		"n":      ig.imageNumber,
		"size":   ig.imageSize,
	})

	req, _ := http.NewRequest("POST", ig.apiUrl, bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ig.apiKey)

	resp, err := ig.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to refine image: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	var responseData struct {
		Data []struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &responseData); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(responseData.Data) == 0 {
		return "", fmt.Errorf("failed to refine image: no results returned")
	}

	return responseData.Data[0].URL, nil
}
