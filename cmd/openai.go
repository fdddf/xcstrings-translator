package cmd

import (
	"context"
	"fmt"
	"time"

	tm "github.com/fdddf/xcstrings-translator/internal/model"
	"github.com/fdddf/xcstrings-translator/internal/translator"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var openaiCmd = &cobra.Command{
	Use:   "openai",
	Short: "Translate xcstrings using OpenAI compatible API",
	Long: `Translate Localizable.xcstrings file using OpenAI Chat API or compatible APIs.
	
Supports OpenAI API and any API that is compatible with the OpenAI Chat API format. Configuration can be provided
via command line flags, config file, or environment variables.`,
	Run: runOpenAITranslate,
}

func init() {
	rootCmd.AddCommand(openaiCmd)

	// OpenAI specific flags
	openaiCmd.Flags().String("api-key", "", "OpenAI API key (required)")
	openaiCmd.Flags().String("api-base-url", "", "API base URL")
	openaiCmd.Flags().String("model", "", "Model to use for translation")
	openaiCmd.Flags().Float64("temperature", 0, "Temperature for translation")
	openaiCmd.Flags().Int("max-tokens", 0, "Maximum tokens for translation")

	// Bind flags to Viper
	viper.BindPFlag("openai.api_key", openaiCmd.Flags().Lookup("api-key"))
	viper.BindPFlag("openai.api_base_url", openaiCmd.Flags().Lookup("api-base-url"))
	viper.BindPFlag("openai.model", openaiCmd.Flags().Lookup("model"))
	viper.BindPFlag("openai.temperature", openaiCmd.Flags().Lookup("temperature"))
	viper.BindPFlag("openai.max_tokens", openaiCmd.Flags().Lookup("max-tokens"))
}

func runOpenAITranslate(cmd *cobra.Command, args []string) {
	// Get configuration values with fallbacks
	inputFile := viper.GetString("global.input_file")
	if cmd.Flags().Changed("input") {
		inputFile, _ = cmd.Flags().GetString("input")
	}

	outputFile := viper.GetString("global.output_file")
	if cmd.Flags().Changed("output") {
		outputFile, _ = cmd.Flags().GetString("output")
	}

	sourceLang := viper.GetString("global.source_language")
	if cmd.Flags().Changed("source-language") {
		sourceLang, _ = cmd.Flags().GetString("source-language")
	}

	targetLangs := viper.GetStringSlice("global.target_languages")
	if cmd.Flags().Changed("target-languages") {
		targetLangs, _ = cmd.Flags().GetStringSlice("target-languages")
	}

	concurrency := viper.GetInt("global.concurrency")
	if cmd.Flags().Changed("concurrency") {
		concurrency, _ = cmd.Flags().GetInt("concurrency")
	}

	verbose := viper.GetBool("global.verbose")
	if cmd.Flags().Changed("verbose") {
		verbose, _ = cmd.Flags().GetBool("verbose")
	}

	// Get OpenAI specific config
	apiKey := viper.GetString("openai.api_key")
	if cmd.Flags().Changed("api-key") {
		apiKey, _ = cmd.Flags().GetString("api-key")
	}

	apiBaseURL := viper.GetString("openai.api_base_url")
	if cmd.Flags().Changed("api-base-url") {
		apiBaseURL, _ = cmd.Flags().GetString("api-base-url")
	}

	model := viper.GetString("openai.model")
	if cmd.Flags().Changed("model") {
		model, _ = cmd.Flags().GetString("model")
	}

	temperature := viper.GetFloat64("openai.temperature")
	if cmd.Flags().Changed("temperature") {
		temperature, _ = cmd.Flags().GetFloat64("temperature")
	}

	maxTokens := viper.GetInt("openai.max_tokens")
	if cmd.Flags().Changed("max-tokens") {
		maxTokens, _ = cmd.Flags().GetInt("max-tokens")
	}

	if verbose {
		fmt.Printf("Starting OpenAI Translate with:\n")
		fmt.Printf("  Input file: %s\n", inputFile)
		fmt.Printf("  Output file: %s\n", outputFile)
		fmt.Printf("  Source language: %s\n", sourceLang)
		fmt.Printf("  Target languages: %v\n", targetLangs)
		fmt.Printf("  Concurrency: %d\n", concurrency)
		fmt.Printf("  API base URL: %s\n", apiBaseURL)
		fmt.Printf("  Model: %s\n", model)
		fmt.Printf("  Temperature: %.2f\n", temperature)
		fmt.Printf("  Max tokens: %d\n", maxTokens)
	}

	// Load xcstrings file
	if verbose {
		fmt.Println("Loading xcstrings file...")
	}
	xcstrings, err := tm.LoadXCStrings(inputFile)
	if err != nil {
		fmt.Printf("Error loading xcstrings file: %v\n", err)
		return
	}

	// Override source language if specified
	if sourceLang != "" {
		xcstrings.SourceLanguage = sourceLang
	}

	// Create translation requests
	if verbose {
		fmt.Println("Preparing translation requests...")
	}
	requests := translator.CreateTranslationRequests(xcstrings, targetLangs)
	if verbose {
		fmt.Printf("Found %d strings to translate\n", len(requests))
	}

	if len(requests) == 0 {
		fmt.Println("No strings to translate. Exiting.")
		return
	}

	// Create translator
	provider := translator.NewOpenAITranslator(apiKey, apiBaseURL, model)

	// Create translation service
	service := translator.NewTranslationService(
		provider,
		concurrency,
		600*time.Second, // 10 minute timeout (OpenAI can be slow)
	)

	// Run translation
	if verbose {
		fmt.Println("Starting translation...")
	}
	ctx := context.Background()
	responses, err := service.TranslateBatch(ctx, requests)
	if err != nil {
		fmt.Printf("Translation failed: %v\n", err)
	}

	// Process results
	successCount := 0
	errorCount := 0
	for _, resp := range responses {
		if resp.Error != nil {
			if verbose {
				fmt.Printf("Error translating %s to %s: %v\n", resp.Key, resp.TargetLanguage, resp.Error)
			}
			errorCount++
		} else {
			successCount++
		}
	}

	if verbose {
		fmt.Printf("Translation completed: %d successful, %d failed\n", successCount, errorCount)
	}

	// Apply translations
	if verbose {
		fmt.Println("Applying translations...")
	}
	translator.ApplyTranslations(xcstrings, responses)

	// Save output
	if verbose {
		fmt.Printf("Saving output to %s...\n", outputFile)
	}
	err = tm.SaveXCStrings(outputFile, xcstrings)
	if err != nil {
		fmt.Printf("Error saving output file: %v\n", err)
		return
	}

	fmt.Printf("Translation completed successfully!\n")
	fmt.Printf("Results saved to: %s\n", outputFile)
}
