package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/fdddf/xcstrings-translator/internal/model"
	"github.com/fdddf/xcstrings-translator/internal/translator"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var baiduCmd = &cobra.Command{
	Use:   "baidu",
	Short: "Translate xcstrings using Baidu Translate API",
	Long: `Translate Localizable.xcstrings file using Baidu Translate API.
	
Requires a valid Baidu Translate API AppID and AppSecret. Configuration can be provided
via command line flags, config file, or environment variables.`,
	RunE: runBaiduTranslate,
}

func init() {
	rootCmd.AddCommand(baiduCmd)

	// Baidu specific flags
	baiduCmd.Flags().String("app-id", "", "Baidu Translate AppID (required)")
	baiduCmd.Flags().String("app-secret", "", "Baidu Translate AppSecret (required)")

	// Bind flags to Viper
	viper.BindPFlag("baidu.app_id", baiduCmd.Flags().Lookup("app-id"))
	viper.BindPFlag("baidu.app_secret", baiduCmd.Flags().Lookup("app-secret"))
}

func runBaiduTranslate(cmd *cobra.Command, args []string) error {
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

	// Get Baidu specific config
	appID := viper.GetString("baidu.app_id")
	if cmd.Flags().Changed("app-id") {
		appID, _ = cmd.Flags().GetString("app-id")
	}

	appSecret := viper.GetString("baidu.app_secret")
	if cmd.Flags().Changed("app-secret") {
		appSecret, _ = cmd.Flags().GetString("app-secret")
	}

	if verbose {
		fmt.Printf("Starting Baidu Translate with:\n")
		fmt.Printf("  Input file: %s\n", inputFile)
		fmt.Printf("  Output file: %s\n", outputFile)
		fmt.Printf("  Source language: %s\n", sourceLang)
		fmt.Printf("  Target languages: %v\n", targetLangs)
		fmt.Printf("  Concurrency: %d\n", concurrency)
		fmt.Printf("  AppID: %s\n", appID)
	}

	// Load xcstrings file
	if verbose {
		fmt.Println("Loading xcstrings file...")
	}
	xcstrings, err := model.LoadXCStrings(inputFile)
	if err != nil {
		fmt.Printf("Error loading xcstrings file: %v\n", err)
		return err
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
		return nil
	}

	// Create translator
	provider := translator.NewBaiduTranslator(appID, appSecret)

	// Create translation service
	service := translator.NewTranslationService(
		provider,
		concurrency,
		300*time.Second, // 5 minute timeout
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
	err = model.SaveXCStrings(outputFile, xcstrings)
	if err != nil {
		fmt.Printf("Error saving output file: %v\n", err)
		return err
	}

	fmt.Printf("Translation completed successfully!\n")
	fmt.Printf("Results saved to: %s\n", outputFile)
	return nil
}
