package model

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// XCStrings represents the structure of a Localizable.xcstrings file
type XCStrings struct {
	SourceLanguage string                 `json:"sourceLanguage"`
	Strings        map[string]StringEntry `json:"strings"`
	Version        string                 `json:"version"`
}

// StringEntry represents a single string entry with its localizations
type StringEntry struct {
	Localizations map[string]Localization `json:"localizations"`
}

// Localization represents a localization for a specific language
type Localization struct {
	StringUnit StringUnit `json:"stringUnit"`
}

// StringUnit contains the translation state and value
type StringUnit struct {
	State string `json:"state"`
	Value string `json:"value"`
}

// TranslationRequest represents a request to translate a string
type TranslationRequest struct {
	Key            string
	Text           string
	SourceLanguage string
	TargetLanguage string
}

// TranslationResponse represents a response from a translation provider
type TranslationResponse struct {
	Key            string
	TargetLanguage string
	TranslatedText string
	Error          error
}

// TranslationProvider defines the interface for translation providers
type TranslationProvider interface {
	Translate(ctx context.Context, req TranslationRequest) (TranslationResponse, error)
}

// LoadXCStrings loads an xcstrings file from disk
func LoadXCStrings(filePath string) (*XCStrings, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var xcstrings XCStrings
	err = json.Unmarshal(data, &xcstrings)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return &xcstrings, nil
}

// SaveXCStrings saves an xcstrings file to disk
func SaveXCStrings(filePath string, xcstrings *XCStrings) error {
	data, err := json.MarshalIndent(xcstrings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}
