package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fdddf/opentrans/internal/dao/query"
	"github.com/fdddf/opentrans/internal/database"
	"github.com/fdddf/opentrans/internal/model"
	"github.com/fdddf/opentrans/internal/translator"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// QueueService handles translation queue operations
type QueueService struct {
	DB                     *database.Database
	Query                  *query.Query
	AppService             *AppService
	AppLocalizationService *AppLocalizationService
	ProjectService         *ProjectService
	TranslationService     *TranslationService
	SubscriptionService    *SubscriptionService
	ProviderService        *ProviderService
	AppProviderConfigService *AppProviderConfigService

	mu    sync.RWMutex
	queue map[uint]*database.TranslationQueue // In-memory cache of active jobs
}

var queueServiceInstance *QueueService

// GetQueueService returns a singleton instance of QueueService
func GetQueueService() *QueueService {
	if queueServiceInstance == nil {
		queueServiceInstance = &QueueService{
			queue: make(map[uint]*database.TranslationQueue),
		}
	}
	return queueServiceInstance
}

// SetQueueServiceInstance sets the global QueueService instance (used by FX)
func SetQueueServiceInstance(qs *QueueService) {
	queueServiceInstance = qs
}

// SubmitTranslationJob submits a new translation job to the queue
func (qs *QueueService) SubmitTranslationJob(userID uint, projectID *uint, appID *uint, jobType, providerType, sourceLanguage string, targetLanguages []string, configData map[string]interface{}) (*database.TranslationQueue, error) {
	// Check user's subscription and usage limits
	if projectID != nil {
		// This is related to a project, check if user has access
		project, err := qs.ProjectService.GetProject(*projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to get project: %v", err)
		}

		if project.UserID != userID {
			return nil, errors.New("user does not have access to this project")
		}
	} else if appID != nil {
		// This is related to an app, check if user has access
		hasAccess, _, err := qs.AppService.CheckUserAccessToApp(*appID, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to check user access to app: %v", err)
		}

		if !hasAccess {
			return nil, errors.New("user does not have access to this app")
		}
	}

	// Create the queue entry
	queueEntry := &database.TranslationQueue{
		UserID:          userID,
		ProjectID:       projectID,
		AppID:           appID,
		Type:            jobType,
		Status:          "pending",
		ProviderType:    providerType,
		SourceLanguage:  sourceLanguage,
		TargetLanguages: targetLanguages,
		ConfigData:      configData,
		Progress:        0,
		Total:           0, // Will be set when processing starts
		Done:            0,
	}

	if err := qs.Query.TranslationQueue.Create(queueEntry); err != nil {
		return nil, fmt.Errorf("failed to submit job to queue: %v", err)
	}

	// Add to in-memory cache
	qs.mu.Lock()
	qs.queue[queueEntry.ID] = queueEntry
	qs.mu.Unlock()

	return queueEntry, nil
}

// GetQueueJob retrieves a specific queue job
func (qs *QueueService) GetQueueJob(jobID uint) (*database.TranslationQueue, error) {
	// Check in-memory cache first
	qs.mu.RLock()
	job, exists := qs.queue[jobID]
	qs.mu.RUnlock()

	if exists {
		return job, nil
	}

	// If not in cache, fetch from database
	queueJob, err := qs.Query.TranslationQueue.Where(
		qs.Query.TranslationQueue.ID.Eq(jobID),
	).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("queue job not found")
	}

	if err != nil {
		return nil, fmt.Errorf("database error: %v", err)
	}

	// Add to cache
	qs.mu.Lock()
	qs.queue[jobID] = queueJob
	qs.mu.Unlock()

	return queueJob, nil
}

// GetQueueJobsByUser retrieves all queue jobs for a user
func (qs *QueueService) GetQueueJobsByUser(userID uint) ([]database.TranslationQueue, error) {
	queueJobs, err := qs.Query.TranslationQueue.Where(
		qs.Query.TranslationQueue.UserID.Eq(userID),
	).Order(qs.Query.TranslationQueue.CreatedAt.Desc()).Find()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve queue jobs: %v", err)
	}

	// Update cache with these jobs
	qs.mu.Lock()
	for _, job := range queueJobs {
		qs.queue[job.ID] = job
	}
	qs.mu.Unlock()

	// Convert slice of pointers to slice of values
	result := make([]database.TranslationQueue, len(queueJobs))
	for i, job := range queueJobs {
		result[i] = *job
	}

	return result, nil
}

// GetPendingJobs retrieves all pending jobs that need processing
func (qs *QueueService) GetPendingJobs() ([]database.TranslationQueue, error) {
	queueJobs, err := qs.Query.TranslationQueue.Where(
		qs.Query.TranslationQueue.Status.Eq("pending"),
	).Order(qs.Query.TranslationQueue.Priority.Desc(), qs.Query.TranslationQueue.CreatedAt.Asc()).Find()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve pending jobs: %v", err)
	}

	// Update cache with these jobs
	qs.mu.Lock()
	for _, job := range queueJobs {
		qs.queue[job.ID] = job
	}
	qs.mu.Unlock()

	// Convert slice of pointers to slice of values
	result := make([]database.TranslationQueue, len(queueJobs))
	for i, job := range queueJobs {
		result[i] = *job
	}

	return result, nil
}

// UpdateQueueJob updates a queue job
func (qs *QueueService) UpdateQueueJob(jobID uint, updates map[string]interface{}) error {
	result, err := qs.Query.TranslationQueue.Where(
		qs.Query.TranslationQueue.ID.Eq(jobID),
	).Updates(updates)
	if err != nil {
		return fmt.Errorf("failed to update queue job: %v", err)
	}

	if result.RowsAffected == 0 {
		return errors.New("queue job not found")
	}

	// Update in-memory cache
	qs.mu.Lock()
	if job, exists := qs.queue[jobID]; exists {
		for key, value := range updates {
			switch key {
			case "Status":
				if s, ok := value.(string); ok {
					job.Status = s
				}
			case "Progress":
				job.Progress = safeIntConversion(value)
			case "Total":
				job.Total = safeIntConversion(value)
			case "Done":
				job.Done = safeIntConversion(value)
			case "Error":
				if s, ok := value.(string); ok {
					job.Error = s
				}
			case "ResultData":
				if m, ok := value.(map[string]interface{}); ok {
					job.ResultData = m
				}
			}
		}
	}
	qs.mu.Unlock()

	return nil
}

// safeIntConversion safely converts various integer types to int
func safeIntConversion(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return 0
	}
}

// ProcessNextJob processes the next available job in the queue
func (qs *QueueService) ProcessNextJob() error {
	// Get the next pending job - use raw DB session to avoid logging spam
	var nextJob database.TranslationQueue
	result := qs.DB.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
		Where("status = ?", "pending").
		Order("priority DESC, created_at ASC").
		First(&nextJob)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// No pending jobs - this is normal, don't log
		return nil
	}

	if result.Error != nil {
		return fmt.Errorf("failed to get next job: %v", result.Error)
	}

	// Update status to processing
	err := qs.UpdateQueueJob(nextJob.ID, map[string]interface{}{
		"Status":   "processing",
		"Progress": 0,
	})
	if err != nil {
		return fmt.Errorf("failed to update job status: %v", err)
	}

	// Process the job based on type
	switch nextJob.Type {
	case "xcstrings":
		err = qs.processXCStringsJob(&nextJob)
	case "app_localization":
		err = qs.processAppLocalizationJob(&nextJob)
	default:
		err = errors.New("unknown job type: " + nextJob.Type)
	}

	if err != nil {
		// Mark as failed
		failureErr := qs.UpdateQueueJob(nextJob.ID, map[string]interface{}{
			"Status":   "failed",
			"Error":    err.Error(),
			"Progress": 100,
		})
		if failureErr != nil {
			return fmt.Errorf("failed to mark job as failed: %v, original error: %v", failureErr, err)
		}
		return err
	} else {
		// Mark as completed
		err = qs.UpdateQueueJob(nextJob.ID, map[string]interface{}{
			"Status":   "completed",
			"Progress": 100,
		})
		if err != nil {
			return fmt.Errorf("failed to mark job as completed: %v", err)
		}
	}

	return nil
}

// processXCStringsJob processes an xcstrings translation job
func (qs *QueueService) processXCStringsJob(job *database.TranslationQueue) error {
	if job.ProjectID == nil {
		return errors.New("project ID is required for xcstrings job")
	}

	// Get the project
	project, err := qs.ProjectService.GetProject(*job.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %v", err)
	}

	// Parse the project content
	xcstrings, err := qs.ProjectService.ParseProjectContent(project)
	if err != nil {
		return fmt.Errorf("failed to parse project content: %v", err)
	}

	// Create translation provider
	provider, err := createProviderFromConfig(job.ProviderType, job.ConfigData)
	if err != nil {
		return fmt.Errorf("failed to create provider: %v", err)
	}

	// Get the actual translation provider interface
	translationProvider, ok := provider.(model.TranslationProvider)
	if !ok {
		return errors.New("invalid provider type")
	}

	// Create translation service
	concurrency := 4
	if c, ok := job.ConfigData["concurrency"].(int); ok && c > 0 {
		concurrency = c
	}

	timeout := 300 * time.Second
	if t, ok := job.ConfigData["timeout"].(int); ok && t > 0 {
		timeout = time.Duration(t) * time.Second
	}

	service := translator.NewTranslationService(translationProvider, concurrency, timeout)

	// Filter out unsupported languages for Hunyuan provider
	supportedTargetLanguages := job.TargetLanguages
	if job.ProviderType == "llama" {
		supportedTargetLanguages = make([]string, 0, len(job.TargetLanguages))
		for _, lang := range job.TargetLanguages {
			if isLanguageSupportedByHunyuan(lang) {
				// Language is supported
				supportedTargetLanguages = append(supportedTargetLanguages, lang)
			} else {
				// Language not supported, skip it
				fmt.Printf("Skipping unsupported language: %s (not supported by Hunyuan model)\n", lang)
			}
		}

		if len(supportedTargetLanguages) == 0 {
			// No supported languages to translate
			fmt.Printf("No supported languages found for xcstrings translation job %d\n", job.ID)
			err = qs.UpdateQueueJob(job.ID, map[string]interface{}{
				"Done":     0,
				"Progress": 100,
				"Status":   "completed",
			})
			if err != nil {
				return fmt.Errorf("failed to update job progress: %v", err)
			}
			return nil
		}
	}

	// Create translation requests
	requests := translator.CreateTranslationRequests(xcstrings, supportedTargetLanguages)

	// Update job with total count
	err = qs.UpdateQueueJob(job.ID, map[string]interface{}{
		"Total": len(requests),
	})
	if err != nil {
		return fmt.Errorf("failed to update job total: %v", err)
	}

	if len(requests) == 0 {
		// Nothing to translate
		err = qs.UpdateQueueJob(job.ID, map[string]interface{}{
			"Done":     0,
			"Progress": 100,
		})
		if err != nil {
			return fmt.Errorf("failed to update job progress: %v", err)
		}
		return nil
	}

	// Execute translations with progress tracking
	ctx := context.Background()
	doneCount := 0

	progressBuilder := func(target string, total int) translator.ProgressReporter {
		return func(done, total int, resp model.TranslationResponse) {
			if resp.Error == nil {
				// Apply translation to xcstrings
				translator.ApplyTranslations(xcstrings, []model.TranslationResponse{resp})
			}

			// Update progress
			doneCount++
			progress := int(float64(doneCount) / float64(len(requests)) * 100)

			qs.UpdateQueueJob(job.ID, map[string]interface{}{
				"Done":     doneCount,
				"Progress": progress,
			})
		}
	}

	// Execute translation
	_, translateErr := translator.TranslatePerLanguage(ctx, xcstrings, supportedTargetLanguages, service, progressBuilder)

	// Save translations to database
	if translateErr == nil {
		err = qs.ProjectService.UpdateTranslationsFromXCStrings(project.ID, xcstrings, job.ProviderType)
		if err != nil {
			return fmt.Errorf("failed to save translations: %v", err)
		}

		// Update project content
		err = qs.ProjectService.SaveProjectContent(project.ID, xcstrings)
		if err != nil {
			return fmt.Errorf("failed to update project content: %v", err)
		}

		// Update user usage
		err = qs.UpdateUserUsage(job.UserID, len(requests))
		if err != nil {
			return fmt.Errorf("failed to update user usage: %v", err)
		}
	}

	// Mark job as completed or failed
	if translateErr != nil {
		return fmt.Errorf("translation failed: %v", translateErr)
	}

	err = qs.UpdateQueueJob(job.ID, map[string]interface{}{
		"Done":     len(requests),
		"Progress": 100,
	})
	if err != nil {
		return fmt.Errorf("failed to update job progress: %v", err)
	}

	return nil
}

// processAppLocalizationJob processes an app localization job
func (qs *QueueService) processAppLocalizationJob(job *database.TranslationQueue) error {
	if job.AppID == nil {
		return errors.New("app ID is required for app localization job")
	}

	fmt.Printf("Starting app localization translation job %d for app %d\n", job.ID, *job.AppID)

	// Get the app
	app, err := qs.AppService.GetApp(*job.AppID)
	if err != nil {
		return fmt.Errorf("failed to get app: %v", err)
	}

	// Get existing localizations
	localizations, err := qs.AppLocalizationService.GetAppLocalizations(*job.AppID)
	if err != nil {
		return fmt.Errorf("failed to get app localizations: %v", err)
	}

	// Get the primary locale as the source
	sourceLoc, err := qs.AppLocalizationService.GetAppLocalization(*job.AppID, app.PrimaryLocale)
	if err != nil {
		return fmt.Errorf("failed to get source localization: %v", err)
	}

	// Log source localization info
	fmt.Printf("Source localization (%s): Name=%q (%d chars), Description=%d chars, Subtitle=%q (%d chars)\n", 
		app.PrimaryLocale, sourceLoc.Name, len(sourceLoc.Name), len(sourceLoc.Description), 
		sourceLoc.Subtitle, len(sourceLoc.Subtitle))

	// Create translation provider
	provider, err := createProviderFromConfig(job.ProviderType, job.ConfigData)
	if err != nil {
		return fmt.Errorf("failed to create provider: %v", err)
	}

	// Get the actual translation provider interface
	translationProvider, ok := provider.(model.TranslationProvider)
	if !ok {
		return errors.New("invalid provider type")
	}

	// Create translation service with concurrency 1 for Llama to avoid thread safety issues
	concurrency := 1
	if job.ProviderType != "llama" {
		concurrency = 4
	}

	timeout := 300 * time.Second
	if t, ok := job.ConfigData["timeout"].(int); ok && t > 0 {
		timeout = time.Duration(t) * time.Second
	}

	service := translator.NewTranslationService(translationProvider, concurrency, timeout)

	// Define fields to translate with their max length for chunking
	// Note: Apple Connect API supports these fields in appStoreVersionLocalizations
	fieldsToTranslate := []struct {
		name       string
		maxLength  int
		sourceFunc func(*database.AppLocalization) string
		targetFunc func(*database.AppLocalization, string)
	}{
		{"Name", 30, func(l *database.AppLocalization) string { return l.Name }, func(l *database.AppLocalization, v string) { l.Name = v }},
		{"Subtitle", 30, func(l *database.AppLocalization) string { return l.Subtitle }, func(l *database.AppLocalization, v string) { l.Subtitle = v }},
		{"Description", 4000, func(l *database.AppLocalization) string { return l.Description }, func(l *database.AppLocalization, v string) { l.Description = v }},
		{"Keywords", 100, func(l *database.AppLocalization) string { return l.Keywords }, func(l *database.AppLocalization, v string) { l.Keywords = v }},
		{"WhatsNew", 4000, func(l *database.AppLocalization) string { return l.WhatsNew }, func(l *database.AppLocalization, v string) { l.WhatsNew = v }},
		{"PromotionalText", 170, func(l *database.AppLocalization) string { return l.PromotionalText }, func(l *database.AppLocalization, v string) { l.PromotionalText = v }},
	}

	// Check if onlyTranslateWhatsNew option is set
	// This option translates both What's New and Promotional Text (update-related content)
	if onlyWhatsNew, ok := job.ConfigData["onlyTranslateWhatsNew"].(bool); ok && onlyWhatsNew {
		fmt.Printf("Only translating What's New and Promotional Text fields as requested\n")
		// Filter to only include WhatsNew and PromotionalText fields
		filteredFields := make([]struct {
			name       string
			maxLength  int
			sourceFunc func(*database.AppLocalization) string
			targetFunc func(*database.AppLocalization, string)
		}, 0)
		for _, f := range fieldsToTranslate {
			if f.name == "WhatsNew" || f.name == "PromotionalText" {
				filteredFields = append(filteredFields, f)
			}
		}
		fieldsToTranslate = filteredFields
	}

	totalFields := len(job.TargetLanguages) * len(fieldsToTranslate)

	// Update job with total count
	err = qs.UpdateQueueJob(job.ID, map[string]interface{}{
		"Total": totalFields,
	})
	if err != nil {
		return fmt.Errorf("failed to update job total: %v", err)
	}

	// Process each target language
	// Use a semaphore to limit concurrency for non-llama providers
	maxConcurrentLangs := 1 // Default to 1 for llama
	if job.ProviderType != "llama" {
		maxConcurrentLangs = 3 // Allow up to 3 languages concurrently for other providers
	}

	sem := make(chan struct{}, maxConcurrentLangs)
	var wg sync.WaitGroup
	var mu sync.Mutex // Protect doneCount, successCount, errorCount

	doneCount := 0
	ctx := context.Background()
	successCount := 0
	errorCount := 0

	// Create a map to store language results
	type languageResult struct {
		lang      string
		targetLoc *database.AppLocalization
		exists    bool
		err       error
	}
	results := make(chan languageResult, len(job.TargetLanguages))

	// Process languages concurrently (with semaphore limit)
	for _, targetLang := range job.TargetLanguages {
		wg.Add(1)
		go func(lang string) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			fmt.Printf("Processing language: %s\n", lang)

			// Check if the language is supported by Hunyuan (for llama provider)
			if job.ProviderType == "llama" && !isLanguageSupportedByHunyuan(lang) {
				// Language not supported, skip it
				fmt.Printf("Skipping unsupported language: %s (not supported by Hunyuan model)\n", lang)
				// Update progress for all fields of this language
				mu.Lock()
				for i := 0; i < len(fieldsToTranslate); i++ {
					doneCount++
					progress := int(float64(doneCount) / float64(totalFields) * 100)
					qs.UpdateQueueJob(job.ID, map[string]interface{}{
						"Done":     doneCount,
						"Progress": progress,
					})
				}
				mu.Unlock()
				results <- languageResult{lang: lang, err: errors.New("unsupported language")}
				return
			}

			// Check if localization already exists
			var targetLoc *database.AppLocalization
			exists := false
			for _, loc := range localizations {
				if loc.LanguageCode == lang {
					exists = true
					targetLoc = &loc
					break
				}
			}

			// If it doesn't exist, create one from source
			if !exists {
				targetLoc = &database.AppLocalization{
					AppID:           *job.AppID,
					LanguageCode:    lang,
					Name:            sourceLoc.Name,
					Subtitle:        sourceLoc.Subtitle,
					PrivacyURL:      sourceLoc.PrivacyURL,
					MarketingURL:    sourceLoc.MarketingURL,
					SupportURL:      sourceLoc.SupportURL,
					Description:     sourceLoc.Description,
					Keywords:        sourceLoc.Keywords,
					WhatsNew:        sourceLoc.WhatsNew,
					PromotionalText: sourceLoc.PromotionalText,
				}
			}

			// Check if skipExisting option is set (skip fields that already have translated content)
			skipExisting := false
			if skip, ok := job.ConfigData["skipExisting"].(bool); ok && skip {
				skipExisting = true
			}

			// Translate each field (serial within a language to avoid context conflicts)
			for _, fieldDef := range fieldsToTranslate {
				sourceText := fieldDef.sourceFunc(sourceLoc)

				// Skip empty fields
				if sourceText == "" {
					fmt.Printf("  Field %s is empty, skipping\n", fieldDef.name)
					mu.Lock()
					doneCount++
					progress := int(float64(doneCount) / float64(totalFields) * 100)
					qs.UpdateQueueJob(job.ID, map[string]interface{}{
						"Done":     doneCount,
						"Progress": progress,
					})
					mu.Unlock()
					continue
				}

				// Skip if field already has content and skipExisting is true
				existingText := fieldDef.sourceFunc(targetLoc)
				if skipExisting && exists && existingText != "" {
					fmt.Printf("  Field %s already has content, skipping (skipExisting=true)\n", fieldDef.name)
					mu.Lock()
					doneCount++
					progress := int(float64(doneCount) / float64(totalFields) * 100)
					qs.UpdateQueueJob(job.ID, map[string]interface{}{
						"Done":     doneCount,
						"Progress": progress,
					})
					mu.Unlock()
					continue
				}

				fmt.Printf("  Translating field %s (%d chars)\n", fieldDef.name, len(sourceText))

				// Split long text into chunks if necessary
				chunks := qs.splitTextForTranslation(sourceText, fieldDef.maxLength, lang)

				var translatedText string
				translationSuccess := true

				if len(chunks) > 1 {
					// Translate chunks and concatenate
					fmt.Printf("    Splitting into %d chunks for translation\n", len(chunks))
					for i, chunk := range chunks {
						fmt.Printf("      Chunk %d: %d chars - %q\n", i, len(chunk), truncateStringForLog(chunk, 50))
					}
					var translatedChunks []string
					for i, chunk := range chunks {
						req := model.TranslationRequest{
							Key:            fmt.Sprintf("%s.%s.chunk%d", lang, fieldDef.name, i),
							Text:           chunk,
							SourceLanguage: job.SourceLanguage,
							TargetLanguage: lang,
						}

						responses, err := service.TranslateBatch(ctx, []model.TranslationRequest{req}, nil)
						if err != nil || len(responses) == 0 || responses[0].Error != nil {
							fmt.Printf("    Error translating chunk %d: %v\n", i, err)
							translationSuccess = false
							break
						}
						fmt.Printf("      Chunk %d translated: %d chars - %q\n", i, len(responses[0].TranslatedText), truncateStringForLog(responses[0].TranslatedText, 50))
						translatedChunks = append(translatedChunks, responses[0].TranslatedText)
					}
					if translationSuccess {
						// Always join with newline to preserve paragraph structure
						// Each chunk represents a complete thought/paragraph
						translatedText = strings.Join(translatedChunks, "\n")
						fmt.Printf("    Merged result: %d chars, %d chunks\n", len(translatedText), len(translatedChunks))
					}
				} else {
					// Single chunk translation
					req := model.TranslationRequest{
						Key:            fmt.Sprintf("%s.%s", lang, fieldDef.name),
						Text:           sourceText,
						SourceLanguage: job.SourceLanguage,
						TargetLanguage: lang,
					}

					responses, err := service.TranslateBatch(ctx, []model.TranslationRequest{req}, nil)
					if err != nil || len(responses) == 0 || responses[0].Error != nil {
						var errMsg string
						if err != nil {
							errMsg = err.Error()
						} else if len(responses) > 0 && responses[0].Error != nil {
							errMsg = responses[0].Error.Error()
						}
						fmt.Printf("  Error translating %s: %s\n", fieldDef.name, errMsg)
						translationSuccess = false
					} else {
						translatedText = responses[0].TranslatedText
						fmt.Printf("  Successfully translated %s\n", fieldDef.name)
					}
				}

				// Update the field with translated text if successful
				mu.Lock()
				if translationSuccess {
					fieldDef.targetFunc(targetLoc, translatedText)
					// Log translated text info for debugging
					newlineCount := strings.Count(translatedText, "\n")
					fmt.Printf("  Field %s translation: %d chars, %d newlines\n", fieldDef.name, len(translatedText), newlineCount)
					if newlineCount > 0 && len(translatedText) < 500 {
						fmt.Printf("    Sample: %q\n", translatedText)
					}
					// Log Name field specifically since it's required
					if fieldDef.name == "Name" {
						fmt.Printf("  Name set to: %q\n", translatedText)
					}
					successCount++
				} else {
					errorCount++
					// Keep original text if translation failed
				}

				doneCount++
				progress := int(float64(doneCount) / float64(totalFields) * 100)
				qs.UpdateQueueJob(job.ID, map[string]interface{}{
					"Done":     doneCount,
					"Progress": progress,
				})
				mu.Unlock()
			}

			// Send result
			results <- languageResult{
				lang:      lang,
				targetLoc: targetLoc,
				exists:    exists,
			}
		}(targetLang)
	}

	// Wait for all languages to complete
	wg.Wait()
	close(results)

	// Process results and save to database
	for result := range results {
		if result.err != nil {
			continue
		}

		// Save or update the localization
			updates := map[string]interface{}{
					"name":              result.targetLoc.Name,
					"subtitle":          result.targetLoc.Subtitle,
					"privacy_url":       result.targetLoc.PrivacyURL,
					"marketing_url":     result.targetLoc.MarketingURL,
					"support_url":       result.targetLoc.SupportURL,
					"description":  result.targetLoc.Description,
					"keywords":          result.targetLoc.Keywords,
					"whats_new":         result.targetLoc.WhatsNew,
					"promotional_text":  result.targetLoc.PromotionalText,
				}
				
				// Log description before saving
				descLen := len(result.targetLoc.Description)
				descNewlines := strings.Count(result.targetLoc.Description, "\n")
				nameLen := len(result.targetLoc.Name)
				fmt.Printf("  Saving localization for %s: name=%q (%d chars), description %d chars, %d newlines\n", 
					result.lang, result.targetLoc.Name, nameLen, descLen, descNewlines)
		
				if result.exists {
					err = qs.AppLocalizationService.UpdateAppLocalizationWithValidation(*job.AppID, result.lang, updates)
				} else {
					_, err = qs.AppLocalizationService.CreateAppLocalization(*job.AppID, result.lang,
						result.targetLoc.Name, result.targetLoc.Subtitle, result.targetLoc.PrivacyURL,
						result.targetLoc.MarketingURL, result.targetLoc.SupportURL, result.targetLoc.Description,
						result.targetLoc.Keywords,
						result.targetLoc.WhatsNew, result.targetLoc.PromotionalText, "",
					)
				}
		if err != nil {
			fmt.Printf("Failed to save app localization for %s: %v\n", result.lang, err)
		}
	}

	// Mark job as completed
	err = qs.UpdateQueueJob(job.ID, map[string]interface{}{
		"Done":     totalFields,
		"Progress": 100,
	})
	if err != nil {
		return fmt.Errorf("failed to update job progress: %v", err)
	}

	fmt.Printf("App localization translation job %d completed. Success: %d, Errors: %d\n", job.ID, successCount, errorCount)
	return nil
}

// truncateStringForLog truncates a string for logging purposes
func truncateStringForLog(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// splitTextForTranslation splits text into chunks for translation, respecting paragraph and word boundaries
func (qs *QueueService) splitTextForTranslation(text string, maxLength int, targetLang string) []string {
	// If text is short enough, return as single chunk
	if len(text) <= maxLength {
		return []string{text}
	}

	// Split by paragraphs first (preserving paragraph structure)
	paragraphs := strings.Split(text, "\n")

	var chunks []string
	var currentChunk strings.Builder
	currentLength := 0

	for _, paragraph := range paragraphs {
		// Trim the paragraph to remove leading/trailing whitespace
		paragraph = strings.TrimSpace(paragraph)

		// Skip empty paragraphs
		if paragraph == "" {
			continue
		}

		// Check if this paragraph alone exceeds the max length
		if len(paragraph) > maxLength {
			// Split long paragraph by words
			words := strings.Fields(paragraph)

			for _, word := range words {
				wordLen := len(word)
				newLength := currentLength + wordLen

				// Add space if not the first word in chunk
				if currentLength > 0 {
					newLength++
				}

				if newLength > maxLength && currentLength > 0 {
					// Add current chunk and start a new one
					chunks = append(chunks, currentChunk.String())
					currentChunk.Reset()
					currentLength = 0
				}

				// Add word to current chunk
				if currentLength > 0 {
					currentChunk.WriteString(" ")
				}
				currentChunk.WriteString(word)
				currentLength += wordLen + 1
			}
		} else {
			// Paragraph fits in a single chunk
			paragraphLen := len(paragraph)

			// Check if adding this paragraph would exceed the max length
			if currentLength+paragraphLen > maxLength && currentLength > 0 {
				// Add current chunk and start a new one with this paragraph
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
				currentLength = 0
			}

			// Add paragraph to current chunk
			if currentLength > 0 {
				currentChunk.WriteString("\n")
			}
			currentChunk.WriteString(paragraph)
			currentLength += paragraphLen + 1
		}
	}

	// Add the last chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// CreateTranslationRequestsForLanguages creates translation requests for specific languages
func CreateTranslationRequestsForLanguages(xcstrings *model.XCStrings, targetLanguages []string) []interface{} { // Using interface{} as a placeholder
	// This function would create specific translation requests based on missing keys
	// For now, returning an empty slice - actual implementation would analyze the xcstrings
	// content and create requests for missing translations
	return make([]interface{}, 0)
}

// CreateTranslationProvider builds a translation provider from a provider type and its config data.
// It is the exported entry point used by controllers that need to translate synchronously (outside
// the queue worker) such as the single-text localization translation endpoint.
func CreateTranslationProvider(providerType string, configData map[string]interface{}) (model.TranslationProvider, error) {
	return createProviderFromConfig(providerType, configData)
}

// createProviderFromConfig creates a provider based on configuration data
func createProviderFromConfig(providerType string, configData map[string]interface{}) (model.TranslationProvider, error) {
	providerType = strings.ToLower(providerType)

	switch providerType {
	case "google":
		apiKey, ok := configData["apiKey"].(string)
		if !ok || apiKey == "" {
			return nil, errors.New("apiKey required for Google provider")
		}
		return translator.NewGoogleTranslator(apiKey), nil

	case "deepl":
		apiKey, ok := configData["apiKey"].(string)
		if !ok || apiKey == "" {
			return nil, errors.New("apiKey required for DeepL provider")
		}
		isFree := false
		if free, ok := configData["isFree"].(bool); ok {
			isFree = free
		}
		return translator.NewDeepLTranslator(apiKey, isFree), nil

	case "baidu":
		appID, ok := configData["appId"].(string)
		if !ok || appID == "" {
			return nil, errors.New("appId required for Baidu provider")
		}
		appSecret, ok := configData["appSecret"].(string)
		if !ok || appSecret == "" {
			return nil, errors.New("appSecret required for Baidu provider")
		}
		return translator.NewBaiduTranslator(appID, appSecret), nil

	case "openai":
		apiKey, ok := configData["apiKey"].(string)
		if !ok || apiKey == "" {
			return nil, errors.New("apiKey required for OpenAI provider")
		}
		apiBaseURL := "https://api.openai.com"
		if url, ok := configData["apiBaseUrl"].(string); ok && url != "" {
			apiBaseURL = url
		}
		model := "gpt-3.5-turbo"
		if m, ok := configData["model"].(string); ok && m != "" {
			model = m
		}
		temperature := 0.3
		if t, ok := configData["temperature"].(float64); ok {
			temperature = t
		}
		maxTokens := 1024
		// JSON numbers decode to float64, so accept both float64 and int for robustness.
		if mt, ok := configData["maxTokens"].(float64); ok && mt > 0 {
			maxTokens = int(mt)
		} else if mt, ok := configData["maxTokens"].(int); ok && mt > 0 {
			maxTokens = mt
		}
		return translator.NewOpenAITranslator(apiKey, apiBaseURL, model, temperature, maxTokens), nil

	case "llama":
		// Get default config paths if not provided in configData
		modelPath, ok := configData["modelPath"].(string)
		if !ok || modelPath == "" {
			// Use default from config.yaml
			modelPath = getLlamaConfigPath("model_path", "/data/dev/opentrans/backend/models/HY-MT1.5-1.8B-Q4_K_M.gguf")
			if modelPath == "" {
				return nil, errors.New("modelPath required for Llama provider")
			}
		}

		libPath, ok := configData["libPath"].(string)
		if !ok || libPath == "" {
			// Use default from config.yaml
			libPath = getLlamaConfigPath("lib_path", "/data/dev/yzma/lib/")
			if libPath == "" {
				return nil, errors.New("libPath required for Llama provider")
			}
		}

		// Build Llama options from config data with defaults from config
		options := translator.LlamaOptions{
			ModelPath:   modelPath,
			GrammarPath: "",
			Threads:     getLlamaConfigInt("threads", 4),
			Seed:        getLlamaConfigInt("seed", -1),
			Tokens:      getLlamaConfigInt("tokens", 4096),
			TopK:        getLlamaConfigInt("top_k", 20),
			Tfs:         getLlamaConfigFloat("tfs", 1.0),
			TopP:        getLlamaConfigFloat("top_p", 0.6),
			MinP:        getLlamaConfigFloat("min_p", 0.1),
			TypicalP:    getLlamaConfigFloat("typical_p", 1.0),
			RepeatLastN: getLlamaConfigInt("repeat_last_n", 64),
			RepeatPenalty: getLlamaConfigFloat("repeat_penalty", 1.05),
			FrequencyPenalty: getLlamaConfigFloat("frequency_penalty", 0.0),
			PresencePenalty: getLlamaConfigFloat("presence_penalty", 0.0),
			Temperature: getLlamaConfigFloat("temperature", 0.7),
			Verbose:     getLlamaConfigBool("verbose", false),
		}

		// Override with config values if present
		if t, ok := configData["threads"].(int); ok && t > 0 {
			options.Threads = t
		}
		if s, ok := configData["seed"].(int); ok {
			options.Seed = s
		}
		if tok, ok := configData["tokens"].(int); ok && tok > 0 {
			options.Tokens = tok
		}
		if tk, ok := configData["topK"].(int); ok && tk > 0 {
			options.TopK = tk
		}
		if tf, ok := configData["tfs"].(float64); ok {
			options.Tfs = tf
		}
		if tp, ok := configData["topP"].(float64); ok {
			options.TopP = tp
		}
		if mp, ok := configData["minP"].(float64); ok {
			options.MinP = mp
		}
		if typ, ok := configData["typicalP"].(float64); ok {
			options.TypicalP = typ
		}
		if rln, ok := configData["repeatLastN"].(int); ok && rln > 0 {
			options.RepeatLastN = rln
		}
		if rp, ok := configData["repeatPenalty"].(float64); ok {
			options.RepeatPenalty = rp
		}
		if fp, ok := configData["frequencyPenalty"].(float64); ok {
			options.FrequencyPenalty = fp
		}
		if pp, ok := configData["presencePenalty"].(float64); ok {
			options.PresencePenalty = pp
		}
		if temp, ok := configData["temperature"].(float64); ok {
			options.Temperature = temp
		}
		if v, ok := configData["verbose"].(bool); ok {
			options.Verbose = v
		}

		// Initialize llama library before creating translator
		if err := translator.InitLlamaLibrary(libPath); err != nil {
			return nil, fmt.Errorf("failed to initialize llama library: %v", err)
		}

		return translator.NewLlamaTranslator(options)

	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// Helper functions to load llama config from config.yaml
func getLlamaConfigPath(key, defaultValue string) string {
	// Try to read from config.yaml
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("../")
	viper.AddConfigPath("../../")
	viper.AddConfigPath("../../backend/")

	if err := viper.ReadInConfig(); err == nil {
		path := viper.GetString("llama." + key)
		if path != "" {
			return path
		}
	}
	return defaultValue
}

func getLlamaConfigInt(key string, defaultValue int) int {
	// Try to read from config.yaml
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("../")
	viper.AddConfigPath("../../")
	viper.AddConfigPath("../../backend/")

	if err := viper.ReadInConfig(); err == nil {
		val := viper.GetInt("llama." + key)
		if val != 0 {
			return val
		}
	}
	return defaultValue
}

func getLlamaConfigFloat(key string, defaultValue float64) float64 {
	// Try to read from config.yaml
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("../")
	viper.AddConfigPath("../../")
	viper.AddConfigPath("../../backend/")

	if err := viper.ReadInConfig(); err == nil {
		val := viper.GetFloat64("llama." + key)
		if val != 0 {
			return val
		}
	}
	return defaultValue
}

func getLlamaConfigBool(key string, defaultValue bool) bool {
	// Try to read from config.yaml
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("../")
	viper.AddConfigPath("../../")
	viper.AddConfigPath("../../backend/")

	if err := viper.ReadInConfig(); err == nil {
		val := viper.GetBool("llama." + key)
		return val
	}
	return defaultValue
}

// PollForNextJob runs continuously to process jobs in the queue
func (qs *QueueService) PollForNextJob(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := qs.ProcessNextJob()
			if err != nil {
				// Log error but continue processing
				fmt.Printf("Error processing queue job: %v\n", err)
			}
		}
	}
}

// UpdateUserUsage updates the user's monthly translation usage
func (qs *QueueService) UpdateUserUsage(userID uint, count int) error {
	// Get the user
	user, err := qs.Query.User.Where(qs.Query.User.ID.Eq(userID)).First()
	if err != nil {
		return fmt.Errorf("failed to get user: %v", err)
	}

	// Check if user has reached the limit
	if user.MaxTranslations > 0 && user.CurrentUsage+count > user.MaxTranslations {
		return errors.New("translation limit reached for this month")
	}

	// Update usage - using GORM Expr for increment
	_, err = qs.Query.User.Where(qs.Query.User.ID.Eq(userID)).Update(
		qs.Query.User.CurrentUsage,
		gorm.Expr("current_usage + ?", count),
	)
	if err != nil {
		return fmt.Errorf("failed to update user usage: %v", err)
	}

	return nil
}

// Global functions for backward compatibility
func SetQueueService(db *database.Database) {
	queueServiceInstance = &QueueService{
		DB:                     db,
		Query:                  query.Use(db.DB),
		queue:                  make(map[uint]*database.TranslationQueue),
		AppService:             appServiceInstance,
		AppLocalizationService: appLocalizationServiceInstance,
		ProjectService:         projectServiceInstance,
		TranslationService:     translationServiceInstance,
		SubscriptionService:    subscriptionServiceInstance,
		ProviderService:        providerServiceInstance,
		AppProviderConfigService: appProviderConfigServiceInstance,
	}
}

// isLanguageSupportedByHunyuan checks if a language is supported by the Hunyuan model
func isLanguageSupportedByHunyuan(locale string) bool {
	// Common Hunyuan supported languages (simplified check)
	supportedCodes := map[string]bool{
		"zh": true, "zh-Hans": true, "zh-CN": true,
		"en": true, "en-US": true, "en-GB": true,
		"fr": true, "fr-FR": true,
		"pt": true, "pt-PT": true, "pt-BR": true,
		"es": true, "es-ES": true, "es-MX": true,
		"ja": true, "ja-JP": true,
		"tr": true, "tr-TR": true,
		"ru": true, "ru-RU": true,
		"ar": true, "ar-SA": true,
		"ko": true, "ko-KR": true,
		"th": true, "th-TH": true,
		"it": true, "it-IT": true,
		"de": true, "de-DE": true,
		"vi": true, "vi-VN": true,
		"ms": true, "ms-MY": true,
		"id": true, "id-ID": true,
		"tl": true, "fil": true, "tl-PH": true, "fil-PH": true,
		"hi": true, "hi-IN": true,
		"zh-Hant": true, "zh-TW": true, "zh-HK": true,
		"pl": true, "pl-PL": true,
		"cs": true, "cs-CZ": true,
		"nl": true, "nl-NL": true,
		"da": true, "da-DK": true,
		"sv": true, "sv-SE": true,
		"no": true, "nb": true, "nn": true, "nb-NO": true, "nn-NO": true, "no-NO": true,
		"fi": true, "fi-FI": true,
		"el": true, "el-GR": true,
		"he": true, "he-IL": true,
		"uk": true, "uk-UA": true,
		"bn": true, "bn-IN": true,
		"ta": true, "ta-IN": true,
		"km": true, "km-KH": true,
		"my": true, "my-MM": true,
		"fa": true, "fa-IR": true,
		"gu": true, "gu-IN": true,
		"ur": true, "ur-PK": true,
		"te": true, "te-IN": true,
		"mr": true, "mr-IN": true,
		"bo": true, "bo-CN": true,
		"kk": true, "kk-KZ": true,
		"mn": true, "mn-MN": true,
		"ug": true, "ug-CN": true,
		"yue": true, "yue-HK": true,
	}

	// Normalize locale
	normalized := strings.ReplaceAll(locale, "_", "-")
	normalized = strings.TrimSpace(normalized)

	// Check direct match
	if supportedCodes[normalized] {
		return true
	}

	// Check base language code
	parts := strings.Split(normalized, "-")
	if len(parts) > 0 && supportedCodes[parts[0]] {
		return true
	}

	return false
}
