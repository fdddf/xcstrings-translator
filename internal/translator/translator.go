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

// ProgressReporter reports translation progress as responses are produced.
// done is the number of completed requests, total is the total number of requests,
// and last is the most recent response.
type ProgressReporter func(done, total int, last model.TranslationResponse)

// NewTranslationService creates a new TranslationService instance
func NewTranslationService(provider model.TranslationProvider, concurrency int, timeout time.Duration) *TranslationService {
	return &TranslationService{
		Provider:    provider,
		Concurrency: concurrency,
		Timeout:     timeout,
	}
}

// TranslateBatch translates multiple strings concurrently with optional progress reporting.
func (s *TranslationService) TranslateBatch(ctx context.Context, requests []model.TranslationRequest, progress ProgressReporter) ([]model.TranslationResponse, error) {
	if len(requests) == 0 {
		return nil, nil
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()

	// Create channels
	bufferSize := s.bufferSize(len(requests))
	reqChan := make(chan model.TranslationRequest, bufferSize)
	respChan := make(chan model.TranslationResponse, bufferSize)

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
	completed := 0
	var firstErr error
	for resp := range respChan {
		responses = append(responses, resp)
		if resp.Error != nil && firstErr == nil {
			firstErr = fmt.Errorf("translation failed for key %s to %s: %w", resp.Key, resp.TargetLanguage, resp.Error)
			cancel()
		}
		completed++
		if progress != nil {
			progress(completed, len(requests), resp)
		}
	}

	if firstErr != nil {
		return responses, firstErr
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

// bufferSize decides the channel buffer size to avoid allocating an excessively large queue.
func (s *TranslationService) bufferSize(total int) int {
	if total == 0 {
		return 0
	}

	// Keep buffers small to avoid large allocations when total is huge,
	// but give workers enough slack to stay busy.
	concurrency := s.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}

	limit := concurrency * 2
	if total < limit {
		return total
	}
	return limit
}

// CreateTranslationRequests creates translation requests from xcstrings data
func CreateTranslationRequests(xcstrings *model.XCStrings, targetLanguages []string) []model.TranslationRequest {
	var requests []model.TranslationRequest

	for _, targetLang := range targetLanguages {
		requests = append(requests, CreateTranslationRequestsForLanguage(xcstrings, targetLang)...)
	}

	return requests
}

// CreateTranslationRequestsForLanguage builds requests only for the given target language.
func CreateTranslationRequestsForLanguage(xcstrings *model.XCStrings, targetLanguage string) []model.TranslationRequest {
	var requests []model.TranslationRequest

	for key, entry := range xcstrings.Strings {
		if entry.ShouldTranslate != nil && *entry.ShouldTranslate == false {
			continue
		}

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

		// Skip if already has translation for this target
		if _, ok := entry.Localizations[targetLanguage]; ok {
			continue
		}

		requests = append(requests, model.TranslationRequest{
			Key:            key,
			Text:           sourceText,
			SourceLanguage: xcstrings.SourceLanguage,
			TargetLanguage: targetLanguage,
		})
	}

	return requests
}

// TranslatePerLanguage runs translations language by language to avoid building a massive request list.
// The progressBuilder can be nil; when provided it produces a ProgressReporter for each language.
func TranslatePerLanguage(
	ctx context.Context,
	xcstrings *model.XCStrings,
	targetLanguages []string,
	service *TranslationService,
	progressBuilder func(target string, total int) ProgressReporter,
) ([]model.TranslationResponse, error) {
	var allResponses []model.TranslationResponse
	var translateErr error

	for _, target := range targetLanguages {
		requests := CreateTranslationRequestsForLanguage(xcstrings, target)
		if len(requests) == 0 {
			continue
		}

		progress := ProgressReporter(nil)
		if progressBuilder != nil {
			progress = progressBuilder(target, len(requests))
		}

		responses, err := service.TranslateBatch(ctx, requests, progress)
		allResponses = append(allResponses, responses...)
		if err != nil {
			translateErr = fmt.Errorf("translation to %s failed: %w", target, err)
			break
		}
	}

	return allResponses, translateErr
}

// NewVerboseProgressReporter prints coarse progress for a single language when verbose is true.
func NewVerboseProgressReporter(target string, total int, verbose bool) ProgressReporter {
	if !verbose || total == 0 {
		return nil
	}

	// Print roughly every 10% (but at least once) to avoid noisy output.
	step := total / 10
	if step == 0 {
		step = 1
	}

	return func(done, total int, last model.TranslationResponse) {
		if done != total && done%step != 0 {
			return
		}

		status := "ok"
		if last.Error != nil {
			status = "err"
		}
		fmt.Printf("  [%s] %d/%d (%s)\n", target, done, total, status)
	}
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
