package cmd

import (
	"context"
	"fmt"
	"time"

	"xcstrings-translator/internal/model"
	"xcstrings-translator/internal/translator"

	"github.com/spf13/cobra"
)

var baiduCmd = &cobra.Command{
	Use:   "baidu",
	Short: "Translate xcstrings using Baidu Translate API",
	Long: `Translate Localizable.xcstrings file using Baidu Translate API.
	
Requires a valid Baidu Translate API AppID and AppSecret.`,
	Run: runBaiduTranslate,
}

func init() {
	rootCmd.AddCommand(baiduCmd)

	// Baidu specific flags
	baiduCmd.Flags().StringP("app-id", "i", "", "Baidu Translate AppID (required)")
	baiduCmd.Flags().StringP("app-secret", "s", "", "Baidu Translate AppSecret (required)")
	baiduCmd.MarkFlagRequired("app-id")
	baiduCmd.MarkFlagRequired("app-secret")
}

func runBaiduTranslate(cmd *cobra.Command, args []string) {
	// Get flags
	inputFile, _ := cmd.Flags().GetString("input")
	outputFile, _ := cmd.Flags().GetString("output")
	sourceLang, _ := cmd.Flags().GetString("source-language")
	targetLangs, _ := cmd.Flags().GetStringSlice("target-languages")
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	verbose, _ := cmd.Flags().GetBool("verbose")
	appID, _ := cmd.Flags().GetString("app-id")
	appSecret, _ := cmd.Flags().GetString("app-secret")

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
		return
	}

	fmt.Printf("Translation completed successfully!\n")
	fmt.Printf("Results saved to: %s\n", outputFile)
}
