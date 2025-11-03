package cmd

import (
	"context"
	"fmt"
	"time"

	"xcstrings-translator/internal/model"
	"xcstrings-translator/internal/translator"

	"github.com/spf13/cobra"
)

var deeplCmd = &cobra.Command{
	Use:   "deepl",
	Short: "Translate xcstrings using DeepL API",
	Long: `Translate Localizable.xcstrings file using DeepL API.
	
Requires a valid DeepL API key. Use --free for the free API tier.`,
	Run: runDeepLTranslate,
}

func init() {
	rootCmd.AddCommand(deeplCmd)

	// DeepL specific flags
	deeplCmd.Flags().StringP("api-key", "k", "", "DeepL API key (required)")
	deeplCmd.Flags().BoolP("free", "f", false, "Use DeepL free API tier")
	deeplCmd.Flags().StringP("formality", "m", "default", "Formality level (default, more, less)")
	deeplCmd.MarkFlagRequired("api-key")
}

func runDeepLTranslate(cmd *cobra.Command, args []string) {
	// Get flags
	inputFile, _ := cmd.Flags().GetString("input")
	outputFile, _ := cmd.Flags().GetString("output")
	sourceLang, _ := cmd.Flags().GetString("source-language")
	targetLangs, _ := cmd.Flags().GetStringSlice("target-languages")
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	verbose, _ := cmd.Flags().GetBool("verbose")
	apiKey, _ := cmd.Flags().GetString("api-key")
	isFree, _ := cmd.Flags().GetBool("free")
	formality, _ := cmd.Flags().GetString("formality")

	if verbose {
		fmt.Printf("Starting DeepL Translate with:\n")
		fmt.Printf("  Input file: %s\n", inputFile)
		fmt.Printf("  Output file: %s\n", outputFile)
		fmt.Printf("  Source language: %s\n", sourceLang)
		fmt.Printf("  Target languages: %v\n", targetLangs)
		fmt.Printf("  Concurrency: %d\n", concurrency)
		fmt.Printf("  API tier: %s\n", map[bool]string{true: "free", false: "pro"}[isFree])
		fmt.Printf("  Formality: %s\n", formality)
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
	provider := translator.NewDeepLTranslator(apiKey, isFree)

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
