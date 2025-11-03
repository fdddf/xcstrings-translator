package cmd

import (
	"context"
	"fmt"
	"time"

	tm "xcstrings-translator/internal/model"
	"xcstrings-translator/internal/translator"

	"github.com/spf13/cobra"
)

var googleCmd = &cobra.Command{
	Use:   "google",
	Short: "Translate xcstrings using Google Translate API",
	Long: `Translate Localizable.xcstrings file using Google Translate API.
	
Requires a valid Google Cloud API key with Translate API enabled.`,
	Run: runGoogleTranslate,
}

func init() {
	rootCmd.AddCommand(googleCmd)

	// Google specific flags
	googleCmd.Flags().StringP("api-key", "k", "", "Google Cloud API key (required)")
	googleCmd.Flags().StringP("model", "m", "nmt", "Translation model (nmt or base)")
	googleCmd.Flags().StringP("glossary", "g", "", "Glossary to use for translation")
	googleCmd.MarkFlagRequired("api-key")
}

func runGoogleTranslate(cmd *cobra.Command, args []string) {
	// Get flags
	inputFile, _ := cmd.Flags().GetString("input")
	outputFile, _ := cmd.Flags().GetString("output")
	sourceLang, _ := cmd.Flags().GetString("source-language")
	targetLangs, _ := cmd.Flags().GetStringSlice("target-languages")
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	verbose, _ := cmd.Flags().GetBool("verbose")
	apiKey, _ := cmd.Flags().GetString("api-key")
	model, _ := cmd.Flags().GetString("model")
	// glossary, _ := cmd.Flags().GetString("glossary")

	if verbose {
		fmt.Printf("Starting Google Translate with:\n")
		fmt.Printf("  Input file: %s\n", inputFile)
		fmt.Printf("  Output file: %s\n", outputFile)
		fmt.Printf("  Source language: %s\n", sourceLang)
		fmt.Printf("  Target languages: %v\n", targetLangs)
		fmt.Printf("  Concurrency: %d\n", concurrency)
		fmt.Printf("  Model: %s\n", model)
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
	provider := translator.NewGoogleTranslator(apiKey)

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
	err = tm.SaveXCStrings(outputFile, xcstrings)
	if err != nil {
		fmt.Printf("Error saving output file: %v\n", err)
		return
	}

	fmt.Printf("Translation completed successfully!\n")
	fmt.Printf("Results saved to: %s\n", outputFile)
}
