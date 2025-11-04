package translator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fdddf/xcstrings-translator/internal/model"
)

// TranslationService manages the translation process with concurrency
type TranslationService struct {
	Provider    model.TranslationProvider
	Concurrency int
	Timeout     time.Duration
}

// NewTranslationService creates a new TranslationService instance
func NewTranslationService(provider model.TranslationProvider, concurrency int, timeout time.Duration) *TranslationService {
	return &TranslationService{
		Provider:    provider,
		Concurrency: concurrency,
		Timeout:     timeout,
	}
}

// TranslateBatch translates multiple strings concurrently
func (s *TranslationService) TranslateBatch(ctx context.Context, requests []model.TranslationRequest) ([]model.TranslationResponse, error) {
	if len(requests) == 0 {
		return nil, nil
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()

	// Create channels
	reqChan := make(chan model.TranslationRequest, len(requests))
	respChan := make(chan model.TranslationResponse, len(requests))

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < s.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			s.worker(ctx, reqChan, respChan, workerID)
		}(i)
	}

	// Send requests to the channel
	go func() {
		for _, req := range requests {
			select {
			case reqChan <- req:
			case <-ctx.Done():
				return
			}
		}
		close(reqChan)
	}()

	// Collect responses
	go func() {
		wg.Wait()
		close(respChan)
	}()

	// Process responses
	var responses []model.TranslationResponse
	for resp := range respChan {
		responses = append(responses, resp)
	}

	if ctx.Err() != nil {
		return responses, fmt.Errorf("translation timed out: %v", ctx.Err())
	}

	return responses, nil
}

// worker processes translation requests from the channel
func (s *TranslationService) worker(ctx context.Context, reqChan <-chan model.TranslationRequest, respChan chan<- model.TranslationResponse, workerID int) {
	for req := range reqChan {
		select {
		case <-ctx.Done():
			return
		default:
			resp, err := s.Provider.Translate(ctx, req)
			if err != nil {
				resp.Error = err
			}
			respChan <- resp
		}
	}
}

// CreateTranslationRequests creates translation requests from xcstrings data
func CreateTranslationRequests(xcstrings *model.XCStrings, targetLanguages []string) []model.TranslationRequest {
	var requests []model.TranslationRequest

	for key, entry := range xcstrings.Strings {
		if entry.ShouldTranslate != nil && *entry.ShouldTranslate == false {
			continue
		}
		// Get source text (from source language)
		sourceText := ""
		if sourceLangEntry, ok := entry.Localizations[xcstrings.SourceLanguage]; ok {
			sourceText = sourceLangEntry.StringUnit.Value
		}

		if key != "" && sourceText == "" {
			sourceText = key
		}

		if sourceText == "" {
			continue
		}

		// Create request for each target language
		for _, targetLang := range targetLanguages {
			// Skip if already has translation
			if _, ok := entry.Localizations[targetLang]; ok {
				continue
			}

			requests = append(requests, model.TranslationRequest{
				Key:            key,
				Text:           sourceText,
				SourceLanguage: xcstrings.SourceLanguage,
				TargetLanguage: targetLang,
			})
		}
	}

	return requests
}

// ApplyTranslations applies translated responses to the xcstrings data
func ApplyTranslations(xcstrings *model.XCStrings, responses []model.TranslationResponse) {
	for _, resp := range responses {
		if resp.Error != nil {
			fmt.Printf("Error translating key %s: %v\n", resp.Key, resp.Error)
			continue
		}

		if entry, ok := xcstrings.Strings[resp.Key]; ok {
			if entry.Localizations == nil {
				entry.Localizations = make(map[string]model.Localization)
			}

			entry.Localizations[resp.TargetLanguage] = model.Localization{
				StringUnit: model.StringUnit{
					State: "translated",
					Value: resp.TranslatedText,
				},
			}

			xcstrings.Strings[resp.Key] = entry
		}
	}
}
