package translator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fdddf/xcstrings-translator/internal/model"

	"github.com/go-resty/resty/v2"
)

// GoogleTranslator implements the TranslationProvider interface for Google Translate API
type GoogleTranslator struct {
	APIKey string
	Client *resty.Client
}

// GoogleTranslateRequest represents the request body for Google Translate API
type GoogleTranslateRequest struct {
	Contents       []string        `json:"contents"`
	SourceLanguage string          `json:"sourceLanguageCode,omitempty"`
	TargetLanguage string          `json:"targetLanguageCode"`
	MimeType       string          `json:"mimeType,omitempty"`
	Model          string          `json:"model,omitempty"`
	GlossaryConfig *GlossaryConfig `json:"glossaryConfig,omitempty"`
}

// GlossaryConfig represents the glossary configuration for Google Translate API
type GlossaryConfig struct {
	Glossary string `json:"glossary"`
}

// GoogleTranslateResponse represents the response from Google Translate API
type GoogleTranslateResponse struct {
	Translations []struct {
		TranslatedText         string `json:"translatedText"`
		DetectedSourceLanguage string `json:"detectedSourceLanguage,omitempty"`
	} `json:"translations"`
}

// NewGoogleTranslator creates a new Google Translator instance
func NewGoogleTranslator(apiKey string) *GoogleTranslator {
	client := resty.New()
	client.SetHeader("Content-Type", "application/json")
	client.SetAuthToken(apiKey)

	return &GoogleTranslator{
		APIKey: apiKey,
		Client: client,
	}
}

// Translate translates a string using Google Translate API
func (g *GoogleTranslator) Translate(ctx context.Context, req model.TranslationRequest) (model.TranslationResponse, error) {
	apiURL := "https://translation.googleapis.com/language/translate/v2"

	requestBody := GoogleTranslateRequest{
		Contents:       []string{req.Text},
		SourceLanguage: req.SourceLanguage,
		TargetLanguage: req.TargetLanguage,
		MimeType:       "text/plain",
	}

	resp, err := g.Client.R().
		SetContext(ctx).
		SetBody(requestBody).
		Post(apiURL)

	if err != nil {
		return model.TranslationResponse{
			Key:            req.Key,
			TargetLanguage: req.TargetLanguage,
			Error:          fmt.Errorf("request failed: %v", err),
		}, nil
	}

	if resp.StatusCode() != http.StatusOK {
		return model.TranslationResponse{
			Key:            req.Key,
			TargetLanguage: req.TargetLanguage,
			Error:          fmt.Errorf("API request failed with status code: %d, response: %s", resp.StatusCode(), resp.String()),
		}, nil
	}

	var translationResponse GoogleTranslateResponse
	err = json.Unmarshal(resp.Body(), &translationResponse)
	if err != nil {
		return model.TranslationResponse{
			Key:            req.Key,
			TargetLanguage: req.TargetLanguage,
			Error:          fmt.Errorf("failed to parse response: %v", err),
		}, nil
	}

	if len(translationResponse.Translations) == 0 {
		return model.TranslationResponse{
			Key:            req.Key,
			TargetLanguage: req.TargetLanguage,
			Error:          fmt.Errorf("no translation results"),
		}, nil
	}

	return model.TranslationResponse{
		Key:            req.Key,
		TargetLanguage: req.TargetLanguage,
		TranslatedText: translationResponse.Translations[0].TranslatedText,
	}, nil
}
