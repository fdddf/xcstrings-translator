package translator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fdddf/xcstrings-translator/internal/model"

	"github.com/go-resty/resty/v2"
)

// OpenAITranslator implements the TranslationProvider interface for OpenAI compatible APIs
type OpenAITranslator struct {
	APIKey     string
	APIBaseURL string
	Model      string
	Client     *resty.Client
}

// OpenAIChatRequest represents the request body for OpenAI Chat API
type OpenAIChatRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	Temperature      float64 `json:"temperature,omitempty"`
	MaxTokens        int     `json:"max_tokens,omitempty"`
	TopP             float64 `json:"top_p,omitempty"`
	FrequencyPenalty float64 `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64 `json:"presence_penalty,omitempty"`
}

// OpenAIChatResponse represents the response from OpenAI Chat API
type OpenAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param,omitempty"`
		Code    string `json:"code,omitempty"`
	} `json:"error,omitempty"`
}

// NewOpenAITranslator creates a new OpenAI Translator instance
func NewOpenAITranslator(apiKey, apiBaseURL, model string) *OpenAITranslator {
	if apiBaseURL == "" {
		apiBaseURL = "https://api.openai.com"
	}
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	client := resty.New()
	client.SetHeader("Content-Type", "application/json")
	client.SetAuthToken(fmt.Sprintf("Bearer %s", apiKey))

	return &OpenAITranslator{
		APIKey:     apiKey,
		APIBaseURL: apiBaseURL,
		Model:      model,
		Client:     client,
	}
}

// Translate translates a string using OpenAI Chat API
func (o *OpenAITranslator) Translate(ctx context.Context, req model.TranslationRequest) (model.TranslationResponse, error) {
	apiURL := fmt.Sprintf("%s/v1/chat/completions", o.APIBaseURL)

	// Create translation prompt
	prompt := fmt.Sprintf("Translate the following text from %s to %s:\n\n%s",
		req.SourceLanguage, req.TargetLanguage, req.Text)

	requestBody := OpenAIChatRequest{
		Model: o.Model,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{
				Role:    "system",
				Content: "You are a professional translator. Translate the text accurately without adding extra information.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.3,
		MaxTokens:   1024,
	}

	resp, err := o.Client.R().
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

	var translationResponse OpenAIChatResponse
	err = json.Unmarshal(resp.Body(), &translationResponse)
	if err != nil {
		return model.TranslationResponse{
			Key:            req.Key,
			TargetLanguage: req.TargetLanguage,
			Error:          fmt.Errorf("failed to parse response: %v", err),
		}, nil
	}

	if translationResponse.Error != nil {
		return model.TranslationResponse{
			Key:            req.Key,
			TargetLanguage: req.TargetLanguage,
			Error:          fmt.Errorf("API error: %s", translationResponse.Error.Message),
		}, nil
	}

	if len(translationResponse.Choices) == 0 || translationResponse.Choices[0].Message.Content == "" {
		return model.TranslationResponse{
			Key:            req.Key,
			TargetLanguage: req.TargetLanguage,
			Error:          fmt.Errorf("no translation results"),
		}, nil
	}

	return model.TranslationResponse{
		Key:            req.Key,
		TargetLanguage: req.TargetLanguage,
		TranslatedText: translationResponse.Choices[0].Message.Content,
	}, nil
}
