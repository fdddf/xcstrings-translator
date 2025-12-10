package server

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fdddf/xcstrings-translator/internal/model"
	"github.com/fdddf/xcstrings-translator/internal/translator"
	"github.com/fdddf/xcstrings-translator/webui"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

// Payload represents the data returned to the UI.
type Payload struct {
	FileName           string           `json:"fileName"`
	SourceLanguage     string           `json:"sourceLanguage"`
	AvailableLanguages []string         `json:"availableLanguages"`
	TotalStrings       int              `json:"totalStrings"`
	Entries            []UILocalization `json:"entries"`
	Warning            string           `json:"warning,omitempty"`
}

// UILocalization is a flattened view for the table UI.
type UILocalization struct {
	Key          string            `json:"key"`
	Source       string            `json:"source"`
	State        string            `json:"state"`
	Translations map[string]string `json:"translations"`
	Missing      []string          `json:"missing"`
}

// TranslateRequest describes the batch translate payload from the UI.
type TranslateRequest struct {
	Provider        string         `json:"provider"`
	TargetLanguages []string       `json:"targetLanguages"`
	SourceLanguage  string         `json:"sourceLanguage"`
	Concurrency     int            `json:"concurrency"`
	TimeoutSeconds  int            `json:"timeoutSeconds"`
	Config          ProviderConfig `json:"config"`
}

// ProviderConfig is the union of provider-specific options we support.
type ProviderConfig struct {
	APIKey      string  `json:"apiKey"`
	APIBaseURL  string  `json:"apiBaseUrl"`
	Model       string  `json:"model"`
	Glossary    string  `json:"glossary"`
	AppID       string  `json:"appId"`
	AppSecret   string  `json:"appSecret"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"maxTokens"`
	Formality   string  `json:"formality"`
	IsFree      bool    `json:"isFree"`
}

// ServerState holds the in-memory working copy of the xcstrings data.
type ServerState struct {
	mu              sync.RWMutex
	fileName        string
	xcstrings       *model.XCStrings
	targetLanguages []string
}

// Serve starts the Fiber server using the embedded UI assets.
func Serve(addr string) error {
	distFS, err := fs.Sub(webui.EmbeddedFS, "dist")
	if err != nil {
		return fmt.Errorf("embedded UI missing: %w", err)
	}

	state := &ServerState{}

	app := fiber.New()
	app.Use(logger.New())
	app.Use(cors.New())

	api := app.Group("/api")
	api.Post("/upload", state.handleUpload)
	api.Get("/strings", state.handleStrings)
	api.Post("/translate", state.handleTranslate)
	api.Get("/export", state.handleExport)

	app.Use("/", filesystem.New(filesystem.Config{
		Root:         http.FS(distFS),
		Browse:       false,
		Index:        "index.html",
		NotFoundFile: "index.html",
	}))

	return app.Listen(addr)
}

func (s *ServerState) handleUpload(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "file is required")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("failed to read file: %v", err))
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("failed to read file: %v", err))
	}

	if len(data) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "empty file")
	}

	xcstrings, err := model.ParseXCStrings(data)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("invalid xcstrings: %v", err))
	}

	if source := c.FormValue("sourceLanguage"); source != "" {
		xcstrings.SourceLanguage = source
	}

	s.mu.Lock()
	s.xcstrings = xcstrings
	s.fileName = fileHeader.Filename
	s.targetLanguages = nil
	s.mu.Unlock()

	payload := s.buildPayload(nil)
	return c.JSON(payload)
}

func (s *ServerState) handleStrings(c *fiber.Ctx) error {
	payload := s.buildPayload(nil)
	if payload == nil {
		return fiber.NewError(fiber.StatusNotFound, "no xcstrings loaded")
	}
	return c.JSON(payload)
}

func (s *ServerState) handleExport(c *fiber.Ctx) error {
	s.mu.RLock()
	xc := s.xcstrings
	name := s.fileName
	s.mu.RUnlock()

	if xc == nil {
		return fiber.NewError(fiber.StatusNotFound, "no xcstrings loaded")
	}

	data, err := model.MarshalXCStrings(xc)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	fileName := name
	if fileName == "" {
		fileName = "Localizable_translated.xcstrings"
	}
	c.Attachment(fileName)
	c.Set("Content-Type", "application/json")
	return c.Send(data)
}

func (s *ServerState) handleTranslate(c *fiber.Ctx) error {
	var req TranslateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if len(req.TargetLanguages) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "targetLanguages is required")
	}

	s.mu.RLock()
	xc := s.xcstrings
	s.mu.RUnlock()

	if xc == nil {
		return fiber.NewError(fiber.StatusBadRequest, "upload a xcstrings file first")
	}

	if req.SourceLanguage != "" {
		xc.SourceLanguage = req.SourceLanguage
	}

	provider, err := buildProvider(strings.ToLower(req.Provider), req.Config)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = 4
	}

	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 300 * time.Second
	}

	service := translator.NewTranslationService(provider, concurrency, timeout)
	ctx := context.Background()

	responses, translateErr := translator.TranslatePerLanguage(ctx, xc, req.TargetLanguages, service, nil)
	translator.ApplyTranslations(xc, responses)

	if len(req.TargetLanguages) > 0 {
		s.mu.Lock()
		s.targetLanguages = dedupe(req.TargetLanguages)
		s.mu.Unlock()
	}

	payload := s.buildPayload(nil)
	if translateErr != nil {
		payload.Warning = translateErr.Error()
	}

	return c.JSON(payload)
}

func (s *ServerState) buildPayload(targets []string) *Payload {
	s.mu.RLock()
	xc := s.xcstrings
	name := s.fileName
	rememberedTargets := s.targetLanguages
	s.mu.RUnlock()

	if xc == nil {
		return nil
	}

	languages := collectLanguages(xc)

	targetSet := dedupe(targets)
	if len(targetSet) == 0 {
		targetSet = rememberedTargets
	}
	if len(targetSet) == 0 {
		for _, lang := range languages {
			if lang != xc.SourceLanguage {
				targetSet = append(targetSet, lang)
			}
		}
		targetSet = dedupe(targetSet)
	}

	entries := flattenEntries(xc, targetSet)

	return &Payload{
		FileName:           name,
		SourceLanguage:     xc.SourceLanguage,
		AvailableLanguages: languages,
		TotalStrings:       len(xc.Strings),
		Entries:            entries,
	}
}

func flattenEntries(xc *model.XCStrings, targets []string) []UILocalization {
	keys := make([]string, 0, len(xc.Strings))
	for key := range xc.Strings {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	entries := make([]UILocalization, 0, len(keys))
	for _, key := range keys {
		entry := xc.Strings[key]
		translations := make(map[string]string)
		for lang, loc := range entry.Localizations {
			translations[lang] = loc.StringUnit.Value
		}

		sourceText := translations[xc.SourceLanguage]
		if sourceText == "" {
			sourceText = key
		}

		state := ""
		if sourceLoc, ok := entry.Localizations[xc.SourceLanguage]; ok {
			state = sourceLoc.StringUnit.State
		}

		missing := []string{}
		for _, target := range targets {
			if translations[target] == "" {
				missing = append(missing, target)
			}
		}

		entries = append(entries, UILocalization{
			Key:          key,
			Source:       sourceText,
			State:        state,
			Translations: translations,
			Missing:      missing,
		})
	}
	return entries
}

func collectLanguages(xc *model.XCStrings) []string {
	langSet := map[string]struct{}{}
	if xc.SourceLanguage != "" {
		langSet[xc.SourceLanguage] = struct{}{}
	}
	for _, entry := range xc.Strings {
		for lang := range entry.Localizations {
			langSet[lang] = struct{}{}
		}
	}

	langs := make([]string, 0, len(langSet))
	for lang := range langSet {
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	return langs
}

func buildProvider(provider string, cfg ProviderConfig) (model.TranslationProvider, error) {
	switch provider {
	case "google":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("apiKey required for Google provider")
		}
		return translator.NewGoogleTranslator(cfg.APIKey), nil
	case "deepl":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("apiKey required for DeepL provider")
		}
		return translator.NewDeepLTranslator(cfg.APIKey, cfg.IsFree), nil
	case "baidu":
		if cfg.AppID == "" || cfg.AppSecret == "" {
			return nil, fmt.Errorf("appId and appSecret are required for Baidu provider")
		}
		return translator.NewBaiduTranslator(cfg.AppID, cfg.AppSecret), nil
	default:
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("apiKey required for OpenAI provider")
		}
		return translator.NewOpenAITranslator(cfg.APIKey, cfg.APIBaseURL, cfg.Model, cfg.Temperature, cfg.MaxTokens), nil
	}
}

func dedupe(list []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, item := range list {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
