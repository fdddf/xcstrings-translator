package controllers

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	appcontext "github.com/fdddf/opentrans/internal/context"
	"github.com/fdddf/opentrans/internal/model"
	"github.com/fdddf/opentrans/internal/services"
	"github.com/fdddf/opentrans/internal/translator"
	"github.com/google/uuid"
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
)

// FileController handles file upload/export requests
type FileController struct {
	state *ServerState
}

// NewFileController creates a new FileController
func NewFileController() *FileController {
	state := &ServerState{}
	// Initialize Llama translator for Hunyuan model
	if err := initLlamaTranslator(state); err != nil {
		fmt.Printf("Failed to initialize Llama translator: %v\n", err)
	}
	return &FileController{
		state: state,
	}
}

// ServerState holds the in-memory working copy of the xcstrings data
type ServerState struct {
	mu              sync.RWMutex
	fileName        string
	xcstrings       *model.XCStrings
	targetLanguages []string
	llamaTranslator *translator.LlamaTranslator
	job             *Job
}

// Payload represents the data returned to the UI
type Payload struct {
	FileName           string           `json:"fileName"`
	SourceLanguage     string           `json:"sourceLanguage"`
	AvailableLanguages []string         `json:"availableLanguages"`
	TotalStrings       int              `json:"totalStrings"`
	Entries            []UILocalization `json:"entries"`
	Warning            string           `json:"warning,omitempty"`
}

// UILocalization is a flattened view for the table UI
type UILocalization struct {
	Key          string            `json:"key"`
	Source       string            `json:"source"`
	State        string            `json:"state"`
	Translations map[string]string `json:"translations"`
	Missing      []string          `json:"missing"`
}

// TranslateTextRequest describes the request for translating a single text
type TranslateTextRequest struct {
	Text             string `json:"text"`
	SourceLanguage   string `json:"sourceLanguage"`
	TargetLanguage   string `json:"targetLanguage"`
	ProviderConfigID *uint  `json:"providerConfigId"`
}

// HandleUpload handles file upload
func (ctrl *FileController) HandleUpload(c *fiber.Ctx) error {
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
		if err := services.ValidateLanguages([]string{source}); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		xcstrings.SourceLanguage = source
	}

	ctrl.state.mu.Lock()
	ctrl.state.xcstrings = xcstrings
	ctrl.state.fileName = fileHeader.Filename
	ctrl.state.targetLanguages = nil
	ctrl.state.mu.Unlock()

	payload := ctrl.buildPayload(nil)
	return c.JSON(payload)
}

// HandleProgress handles progress request
func (ctrl *FileController) HandleProgress(c *fiber.Ctx) error {
	ctrl.state.mu.RLock()
	job := ctrl.state.job
	payload := ctrl.buildPayload(nil)
	ctrl.state.mu.RUnlock()

	return c.JSON(fiber.Map{
		"job":     job,
		"payload": payload,
	})
}

// HandleStrings handles strings request
func (ctrl *FileController) HandleStrings(c *fiber.Ctx) error {
	payload := ctrl.buildPayload(nil)
	if payload == nil {
		return fiber.NewError(fiber.StatusNotFound, "no xcstrings loaded")
	}
	return c.JSON(payload)
}

// HandleExport handles export request
func (ctrl *FileController) HandleExport(c *fiber.Ctx) error {
	ctrl.state.mu.RLock()
	xc := ctrl.state.xcstrings
	name := ctrl.state.fileName
	ctrl.state.mu.RUnlock()

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

// HandleTranslate handles translate request
func (ctrl *FileController) HandleTranslate(c *fiber.Ctx) error {
	var req TranslateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if err := services.ValidateLanguages(req.TargetLanguages); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if len(req.TargetLanguages) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "targetLanguages is required")
	}

	ctrl.state.mu.RLock()
	xc := ctrl.state.xcstrings
	ctrl.state.mu.RUnlock()

	if xc == nil {
		return fiber.NewError(fiber.StatusBadRequest, "upload a xcstrings file first")
	}

	if req.SourceLanguage != "" {
		xc.SourceLanguage = req.SourceLanguage
	}

	requests := translator.CreateTranslationRequests(xc, req.TargetLanguages)
	job := ctrl.startJob(len(requests))

	// If nothing to do, finish immediately.
	if len(requests) == 0 {
		ctrl.finishJob("done", "")
		return c.JSON(fiber.Map{"jobId": job.ID})
	}

	go ctrl.runTranslation(job, xc, req)

	return c.JSON(fiber.Map{"jobId": job.ID})
}

// HandleGetSupportedLanguages returns the supported language codes with metadata
func (ctrl *FileController) HandleGetSupportedLanguages(c *fiber.Ctx) error {
	// Return Apple Connect supported languages (37 languages)
	languages := services.GetAppleConnectLanguages()
	return c.JSON(fiber.Map{
		"success":   true,
		"languages": languages,
	})
}

// HandleTranslateText translates a single text using the configured Llama model
func (ctrl *FileController) HandleTranslateText(c *fiber.Ctx) error {
	var req TranslateTextRequest
	if err := c.BodyParser(&req); err != nil {
		return c.JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}

	if req.Text == "" {
		return c.JSON(fiber.Map{"success": false, "error": "text is required"})
	}

	if req.TargetLanguage == "" {
		return c.JSON(fiber.Map{"success": false, "error": "targetLanguage is required"})
	}

	// Create a translation request
	translationReq := model.TranslationRequest{
		Key:            "localization",
		Text:           req.Text,
		SourceLanguage: req.SourceLanguage,
		TargetLanguage: req.TargetLanguage,
	}

	// Resolve the translation provider. When a saved provider configuration is referenced
	// (e.g. an OpenAI-compatible endpoint), build a translator from it; otherwise fall back to
	// the local Llama/Hunyuan model.
	var provider model.TranslationProvider
	if req.ProviderConfigID != nil {
		userID, ok := appcontext.GetUserIDFromContext(c)
		if !ok {
			return c.JSON(fiber.Map{"success": false, "error": "user not authenticated"})
		}

		providerType, configData, err := services.ResolveProviderConfigForUser(userID, *req.ProviderConfigID)
		if err != nil {
			return c.JSON(fiber.Map{"success": false, "error": err.Error()})
		}

		provider, err = services.CreateTranslationProvider(providerType, configData)
		if err != nil {
			return c.JSON(fiber.Map{"success": false, "error": fmt.Sprintf("failed to initialize provider: %v", err)})
		}
	} else {
		// Use global Llama translator
		if globalLlamaTranslator == nil {
			return c.JSON(fiber.Map{"success": false, "error": "translation service not available - please ensure the Hunyuan model is properly configured"})
		}
		provider = globalLlamaTranslator
	}

	// Translate the text directly using the resolved provider
	response, err := provider.Translate(c.Context(), translationReq)
	if err != nil {
		return c.JSON(fiber.Map{"success": false, "error": fmt.Sprintf("translation failed: %v", err)})
	}

	if response.Error != nil {
		return c.JSON(fiber.Map{"success": false, "error": fmt.Sprintf("translation failed: %v", response.Error)})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"text":    response.TranslatedText,
	})
}

// buildPayload builds the payload for the UI
func (ctrl *FileController) buildPayload(targets []string) *Payload {
	ctrl.state.mu.RLock()
	xc := ctrl.state.xcstrings
	name := ctrl.state.fileName
	rememberedTargets := ctrl.state.targetLanguages
	ctrl.state.mu.RUnlock()

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

// startJob starts a new job
func (ctrl *FileController) startJob(total int) *Job {
	job := &Job{
		ID:        uuid.NewString(),
		Status:    "running",
		Done:      0,
		Total:     total,
		UpdatedAt: time.Now(),
	}
	ctrl.state.mu.Lock()
	ctrl.state.job = job
	ctrl.state.mu.Unlock()
	return job
}

// incrementJob increments the job progress
func (ctrl *FileController) incrementJob(delta int) {
	ctrl.state.mu.Lock()
	defer ctrl.state.mu.Unlock()
	if ctrl.state.job == nil {
		return
	}
	ctrl.state.job.Done += delta
	if ctrl.state.job.Done > ctrl.state.job.Total && ctrl.state.job.Total > 0 {
		ctrl.state.job.Done = ctrl.state.job.Total
	}
	ctrl.state.job.UpdatedAt = time.Now()
}

// finishJob finishes a job
func (ctrl *FileController) finishJob(status, msg string) {
	ctrl.state.mu.Lock()
	defer ctrl.state.mu.Unlock()
	if ctrl.state.job == nil {
		return
	}
	ctrl.state.job.Status = status
	ctrl.state.job.Message = msg
	ctrl.state.job.UpdatedAt = time.Now()
}

// runTranslation runs the translation process
func (ctrl *FileController) runTranslation(job *Job, xc *model.XCStrings, req TranslateRequest) {
	provider, err := buildProvider(strings.ToLower(req.Provider), req.Config)
	if err != nil {
		ctrl.finishJob("error", err.Error())
		return
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

	progressBuilder := func(target string, total int) translator.ProgressReporter {
		return func(done, total int, resp model.TranslationResponse) {
			if resp.Error == nil {
				ctrl.applyResponse(resp)
			}
			ctrl.incrementJob(1)
		}
	}

	responses, translateErr := translator.TranslatePerLanguage(ctx, xc, req.TargetLanguages, service, progressBuilder)
	translator.ApplyTranslations(xc, responses)

	if len(req.TargetLanguages) > 0 {
		ctrl.state.mu.Lock()
		ctrl.state.targetLanguages = dedupe(req.TargetLanguages)
		ctrl.state.mu.Unlock()
	}

	if translateErr != nil {
		ctrl.finishJob("error", translateErr.Error())
		return
	}

	ctrl.finishJob("done", "")
}

// applyResponse applies a single successful translation response
func (ctrl *FileController) applyResponse(resp model.TranslationResponse) {
	if resp.Error != nil {
		return
	}
	ctrl.state.mu.Lock()
	defer ctrl.state.mu.Unlock()
	translator.ApplyTranslations(ctrl.state.xcstrings, []model.TranslationResponse{resp})
}

// Helper functions
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
	case "llama", "hunyuan":
		if globalLlamaTranslator == nil {
			if err := initLlamaTranslator(&ServerState{}); err != nil {
				return nil, fmt.Errorf("failed to initialize hunyuan model: %v", err)
			}
		}
		if globalLlamaTranslator == nil {
			return nil, fmt.Errorf("hunyuan model not available")
		}
		return globalLlamaTranslator, nil
	default:
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("apiKey required for OpenAI provider")
		}
		return translator.NewOpenAITranslator(cfg.APIKey, cfg.APIBaseURL, cfg.Model, cfg.Temperature, cfg.MaxTokens), nil
	}
}

// Global Llama translator instance for Hunyuan model
var (
	globalLlamaTranslator *translator.LlamaTranslator
	llamaInitialized      bool
	llamaInitMu           sync.Mutex
)

// initLlamaTranslator initializes the Llama translator for Hunyuan model
func initLlamaTranslator(state *ServerState) error {
	llamaInitMu.Lock()
	defer llamaInitMu.Unlock()

	// Check if already initialized
	if llamaInitialized {
		return nil
	}

	// Load config.yaml if not already loaded
	if !viper.IsSet("llama.lib_path") {
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./backend")
		if err := viper.ReadInConfig(); err != nil {
			return fmt.Errorf("failed to read config.yaml: %v", err)
		}
	}

	// Get the directory containing libllama.so
	libPath := viper.GetString("llama.lib_path")
	libDir := filepath.Dir(libPath)

	// Set LD_LIBRARY_PATH to help find libggml.so
	if oldPath := os.Getenv("LD_LIBRARY_PATH"); oldPath != "" {
		os.Setenv("LD_LIBRARY_PATH", libDir+":"+oldPath)
	} else {
		os.Setenv("LD_LIBRARY_PATH", libDir)
	}

	// Use default llama config for Hunyuan model
	options := translator.LlamaOptions{
		ModelPath:        viper.GetString("llama.model_path"),
		Threads:          viper.GetInt("llama.threads"),
		Seed:             viper.GetInt("llama.seed"),
		Tokens:           viper.GetInt("llama.tokens"),
		TopK:             viper.GetInt("llama.top_k"),
		Tfs:              viper.GetFloat64("llama.tfs"),
		TopP:             viper.GetFloat64("llama.top_p"),
		MinP:             viper.GetFloat64("llama.min_p"),
		TypicalP:         viper.GetFloat64("llama.typical_p"),
		RepeatLastN:      viper.GetInt("llama.repeat_last_n"),
		RepeatPenalty:    viper.GetFloat64("llama.repeat_penalty"),
		FrequencyPenalty: viper.GetFloat64("llama.frequency_penalty"),
		PresencePenalty:  viper.GetFloat64("llama.presence_penalty"),
		Temperature:      viper.GetFloat64("llama.temperature"),
		Verbose:          viper.GetBool("llama.verbose"),
	}

	// Check if model file exists
	if _, err := os.Stat(options.ModelPath); os.IsNotExist(err) {
		return fmt.Errorf("model file not found: %s", options.ModelPath)
	}

	// Initialize llama library
	if err := translator.InitLlamaLibrary(libPath); err != nil {
		return fmt.Errorf("failed to initialize llama library: %v", err)
	}

	// Create llama translator
	llamaTrans, err := translator.NewLlamaTranslator(options)
	if err != nil {
		return fmt.Errorf("failed to create llama translator: %v", err)
	}

	globalLlamaTranslator = llamaTrans
	state.llamaTranslator = llamaTrans
	llamaInitialized = true
	fmt.Printf("Llama translator initialized successfully with Hunyuan model from %s\n", options.ModelPath)
	return nil
}
