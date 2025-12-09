package translator

import (
	"bufio"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/fdddf/xcstrings-translator/internal/model"

	"github.com/go-resty/resty/v2"
)

// OpenAITranslator implements the TranslationProvider interface for OpenAI compatible APIs
type OpenAITranslator struct {
	APIKey      string
	APIBaseURL  string
	Model       string
	Client      *resty.Client
	Temperature float64
	MaxTokens   int
}

// OpenAIChatRequest represents the request body for OpenAI Chat API
type OpenAIChatRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	Temperature      float64              `json:"temperature,omitempty"`
	MaxTokens        int                  `json:"max_tokens,omitempty"`
	TopP             float64              `json:"top_p,omitempty"`
	FrequencyPenalty float64              `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64              `json:"presence_penalty,omitempty"`
	Stream           bool                 `json:"stream"`
	StreamOptions    *OpenAIStreamOptions `json:"stream_options,omitempty"`
}

// OpenAIStreamOptions represents stream_options to satisfy APIs that require it
// alongside the stream flag, even when streaming is disabled.
type OpenAIStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// OpenAIChatResponse represents the response from OpenAI Chat API
type OpenAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
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

func extractMessageText(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", fmt.Errorf("empty message content")
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return asString, nil
	}

	type contentPart struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		Content string `json:"content"`
		Value   string `json:"value"`
	}
	var parts []contentPart
	if err := json.Unmarshal(raw, &parts); err == nil {
		var builder strings.Builder
		for _, part := range parts {
			text := part.Text
			if text == "" {
				text = part.Content
			}
			if text == "" {
				text = part.Value
			}
			if text == "" {
				continue
			}
			if builder.Len() > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(text)
		}
		if builder.Len() > 0 {
			return builder.String(), nil
		}
	}

	var objectPart contentPart
	if err := json.Unmarshal(raw, &objectPart); err == nil {
		if objectPart.Text != "" {
			return objectPart.Text, nil
		}
		if objectPart.Content != "" {
			return objectPart.Content, nil
		}
		if objectPart.Value != "" {
			return objectPart.Value, nil
		}
	}

	// Return JSON string to aid debugging when we cannot understand the format.
	return "", fmt.Errorf("unsupported message content format: %s", string(raw))
}

func decodeResponseBody(resp *resty.Response) ([]byte, error) {
	body := resp.Body()
	if len(body) == 0 {
		return nil, fmt.Errorf("empty response body")
	}

	encodings := parseContentEncodings(resp.Header().Get("Content-Encoding"))
	data := body
	if len(encodings) > 0 {
		for i := len(encodings) - 1; i >= 0; i-- {
			encoding := encodings[i]
			var err error
			switch encoding {
			case "gzip", "x-gzip":
				data, err = ungzip(data)
			case "deflate":
				data, err = undeflate(data)
			case "br", "brotli", "x-brotli":
				data, err = unbrotli(data)
			case "identity":
				continue
			default:
				return nil, fmt.Errorf("unsupported content encoding: %s", encoding)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to decode %s content: %v", encoding, err)
			}
		}
		return data, nil
	}

	if bytes.HasPrefix(data, []byte{0x1f, 0x8b}) {
		return ungzip(data)
	}
	if looksLikeZlib(data) {
		if decoded, err := undeflate(data); err == nil {
			return decoded, nil
		}
	}

	return data, nil
}

func ungzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func undeflate(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err == nil {
		defer reader.Close()
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, reader); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	flateReader := flate.NewReader(bytes.NewReader(data))
	defer flateReader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, flateReader); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func unbrotli(data []byte) ([]byte, error) {
	reader := brotli.NewReader(bytes.NewReader(data))
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func parseContentEncodings(header string) []string {
	if header == "" {
		return nil
	}
	parts := strings.Split(header, ",")
	var encodings []string
	for _, part := range parts {
		enc := strings.ToLower(strings.TrimSpace(part))
		if enc == "" || enc == "identity" {
			continue
		}
		encodings = append(encodings, enc)
	}
	return encodings
}

func looksLikeZlib(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	cmf := data[0]
	flg := data[1]
	return (uint16(cmf)<<8|uint16(flg))%31 == 0 && cmf&0x0F == 8
}

// NewOpenAITranslator creates a new OpenAI Translator instance
func NewOpenAITranslator(apiKey, apiBaseURL, model string, temperature float64, maxTokens int) *OpenAITranslator {
	if apiBaseURL == "" {
		apiBaseURL = "https://api.openai.com"
	}
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	client := resty.New()
	client.SetHeader("Content-Type", "application/json")
	client.SetAuthToken(apiKey)

	return &OpenAITranslator{
		APIKey:      apiKey,
		APIBaseURL:  apiBaseURL,
		Model:       model,
		Client:      client,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}
}

func (o *OpenAITranslator) translateOnce(ctx context.Context, req model.TranslationRequest, stream bool) (string, error) {
	apiURL := fmt.Sprintf("%s/v1/chat/completions", o.APIBaseURL)

	// Create translation prompt
	prompt := fmt.Sprintf("Translate the following text from %s to %s:\n\n%s",
		req.SourceLanguage, req.TargetLanguage, req.Text)

	temperature := o.Temperature
	if temperature == 0 {
		temperature = 0.3
	}
	maxTokens := o.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}

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
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Stream:      stream,
	}

	// Some providers require stream_options to be present whenever stream is set (even false).
	requestBody.StreamOptions = &OpenAIStreamOptions{IncludeUsage: false}

	resp, err := o.Client.R().
		SetContext(ctx).
		SetBody(requestBody).
		Post(apiURL)

	if err != nil {
		return "", fmt.Errorf("request failed: %v", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("API request failed with status code: %d, response: %s", resp.StatusCode(), resp.String())
	}

	body, err := decodeResponseBody(resp)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	// If we requested streaming, try to parse SSE chunks first.
	if stream {
		if isStreamContent(resp.Header().Get("Content-Type"), body) {
			content, streamErr := parseStreamedContent(body)
			if streamErr == nil {
				return content, nil
			}
			// If the payload clearly looks like stream data, return the parsing error directly.
			if bytes.HasPrefix(bytes.TrimSpace(body), []byte("data:")) {
				return "", streamErr
			}
			// otherwise fall through to JSON parsing for robustness
		}
	}

	var translationResponse OpenAIChatResponse
	fmt.Printf("OpenAI Translation Response status: %d\n", resp.StatusCode())
	err = json.Unmarshal(body, &translationResponse)
	if err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if translationResponse.Error != nil {
		return "", fmt.Errorf("API error: %s", translationResponse.Error.Message)
	}

	if len(translationResponse.Choices) == 0 {
		return "", fmt.Errorf("no translation results")
	}

	content, err := extractMessageText(translationResponse.Choices[0].Message.Content)
	if err != nil {
		return "", fmt.Errorf("failed to parse message content: %v", err)
	}
	if content == "" {
		return "", fmt.Errorf("no translation results")
	}

	return content, nil
}

func isStreamContent(contentType string, body []byte) bool {
	if strings.Contains(strings.ToLower(contentType), "event-stream") {
		return true
	}
	trimmed := bytes.TrimSpace(body)
	return bytes.HasPrefix(trimmed, []byte("data:"))
}

func parseStreamedContent(body []byte) (string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	var builder strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return "", fmt.Errorf("failed to parse streamed chunk: %v", err)
		}
		if len(chunk.Choices) > 0 {
			builder.WriteString(chunk.Choices[0].Delta.Content)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read streamed response: %v", err)
	}

	if builder.Len() == 0 {
		return "", fmt.Errorf("no streamed content")
	}

	return builder.String(), nil
}

// Translate translates a string using OpenAI Chat API
func (o *OpenAITranslator) Translate(ctx context.Context, req model.TranslationRequest) (model.TranslationResponse, error) {
	translatedText, err := o.translateOnce(ctx, req, false)
	if err != nil && strings.Contains(err.Error(), "'stream' and 'stream_options' must be set together") {
		translatedText, err = o.translateOnce(ctx, req, true)
		if err != nil {
			err = fmt.Errorf("streaming retry failed: %w", err)
		}
	}

	if err != nil {
		return model.TranslationResponse{
			Key:            req.Key,
			TargetLanguage: req.TargetLanguage,
			Error:          err,
		}, nil
	}

	return model.TranslationResponse{
		Key:            req.Key,
		TargetLanguage: req.TargetLanguage,
		TranslatedText: translatedText,
	}, nil
}
