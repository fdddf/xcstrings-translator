package server

import (
	"fmt"
	"sync"
	"testing"

	"github.com/fdddf/xcstrings-translator/internal/model"
)

// newTestState builds a ServerState seeded with n source strings.
func newTestState(n int) *ServerState {
	strs := make(map[string]model.StringEntry, n)
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("key_%d", i)
		strs[key] = model.StringEntry{
			Localizations: map[string]model.Localization{
				"en": {StringUnit: model.StringUnit{State: "translated", Value: key}},
			},
		}
	}
	return &ServerState{
		fileName:  "Localizable.xcstrings",
		xcstrings: &model.XCStrings{SourceLanguage: "en", Strings: strs},
		job:       &Job{ID: "test", Status: "running", Total: n},
	}
}

// TestConcurrentApplyAndRead reproduces the scenario where a background
// translation job writes translations (applyResponse) while HTTP handlers read
// the shared state (buildPayload / job snapshot). Run with `go test -race` to
// detect data races on s.xcstrings / s.job.
func TestConcurrentApplyAndRead(t *testing.T) {
	const n = 200
	s := newTestState(n)

	var wg sync.WaitGroup

	// Writer: simulates the translation goroutine applying results under lock.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			s.applyResponse(model.TranslationResponse{
				Key:            fmt.Sprintf("key_%d", i),
				TargetLanguage: "fr",
				TranslatedText: "traduction",
			})
			s.incrementJob(1)
		}
		s.finishJob("done", "")
	}()

	// Readers: simulate /progress, /strings and the job snapshot hammering the
	// state concurrently with the writer.
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < n; i++ {
				if p := s.buildPayload(nil); p != nil {
					_ = p.TotalStrings
				}
				s.mu.RLock()
				if s.job != nil {
					jobCopy := *s.job
					_ = jobCopy
				}
				s.mu.RUnlock()
			}
		}()
	}

	wg.Wait()

	// Sanity: every string should now have a French translation applied.
	payload := s.buildPayload([]string{"fr"})
	if payload == nil {
		t.Fatal("expected payload, got nil")
	}
	for _, e := range payload.Entries {
		if e.Translations["fr"] == "" {
			t.Fatalf("key %q missing fr translation", e.Key)
		}
	}
}
