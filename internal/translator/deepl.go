package translator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"xcstrings-translator/internal/model"

	"github.com/go-resty/resty/v2"
)

// DeepLTranslator implements the TranslationProvider interface for DeepL API
type DeepLTranslator struct {
	APIKey string
	IsFree bool
	Client *resty.Client
}

// DeepLTranslateRequest represents the request body for DeepL API
type DeepLTranslateRequest struct {
	Text               []string `json:"text"`
	TargetLang         string   `json:"target_lang"`
	SourceLang         string   `json:"source_lang,omitempty"`
	SplitSentences     string   `json:"split_sentences,omitempty"`
	PreserveFormatting bool     `json:"preserve_formatting,omitempty"`
	Formality          string   `json:"formality,omitempty"`
}

// DeepLTranslateResponse represents the response from DeepL API
type DeepLTranslateResponse struct {
	Translations []struct {
		DetectedSourceLanguage string `json:"detected_source_language"`
		Text                   string `json:"text"`
	} `json:"translations"`
}

// NewDeepLTranslator creates a new DeepL Translator instance
func NewDeepLTranslator(apiKey string, isFree bool) *DeepLTranslator {
	client := resty.New()
	client.SetHeader("Content-Type", "application/json")
	client.SetAuthToken(fmt.Sprintf("DeepL-Auth-Key %s", apiKey))

	return &DeepLTranslator{
		APIKey: apiKey,
		IsFree: isFree,
		Client: client,
	}
}

// Translate translates a string using DeepL API
func (d *DeepLTranslator) Translate(ctx context.Context, req model.TranslationRequest) (model.TranslationResponse, error) {
	baseURL := "https://api.deepl.com"
	if d.IsFree {
		baseURL = "https://api-free.deepl.com"
	}
	apiURL := fmt.Sprintf("%s/v2/translate", baseURL)

	// Convert language codes to DeepL format (e.g., zh-Hans -> ZH)
	targetLang := strings.ToUpper(req.TargetLanguage)
	if strings.HasPrefix(targetLang, "ZH-") {
		if targetLang == "ZH-HANS" {
			targetLang = "ZH"
		} else if targetLang == "ZH-HANT" {
			targetLang = "ZH-TW"
		}
	}

	requestBody := DeepLTranslateRequest{
		Text:       []string{req.Text},
		TargetLang: targetLang,
		Formality:  "default",
	}

	if req.SourceLanguage != "" {
		sourceLang := strings.ToUpper(req.SourceLanguage)
		if strings.HasPrefix(sourceLang, "ZH-") {
			if sourceLang == "ZH-HANS" {
				sourceLang = "ZH"
			} else if sourceLang == "ZH-HANT" {
				sourceLang = "ZH-TW"
			}
		}
		requestBody.SourceLang = sourceLang
	}

	resp, err := d.Client.R().
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

	var translationResponse DeepLTranslateResponse
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
		TranslatedText: translationResponse.Translations[0].Text,
	}, nil
}
