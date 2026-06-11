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

// globalOptions holds the shared input/output/language settings, resolved from
// flags first and falling back to the config file / environment via viper.
type globalOptions struct {
	InputFile   string
	OutputFile  string
	SourceLang  string
	TargetLangs []string
	Concurrency int
	Verbose     bool
}

// resolveGlobalOptions reads the shared persistent flags for a command.
func resolveGlobalOptions(cmd *cobra.Command) globalOptions {
	return globalOptions{
		InputFile:   stringFlag(cmd, "input", "global.input_file"),
		OutputFile:  stringFlag(cmd, "output", "global.output_file"),
		SourceLang:  stringFlag(cmd, "source-language", "global.source_language"),
		TargetLangs: stringSliceFlag(cmd, "target-languages", "global.target_languages"),
		Concurrency: intFlag(cmd, "concurrency", "global.concurrency"),
		Verbose:     boolFlag(cmd, "verbose", "global.verbose"),
	}
}

// The *Flag helpers return the flag value when the user set it explicitly,
// otherwise the value resolved by viper (config file / env / default).
func stringFlag(cmd *cobra.Command, name, viperKey string) string {
	if cmd.Flags().Changed(name) {
		v, _ := cmd.Flags().GetString(name)
		return v
	}
	return viper.GetString(viperKey)
}

func stringSliceFlag(cmd *cobra.Command, name, viperKey string) []string {
	if cmd.Flags().Changed(name) {
		v, _ := cmd.Flags().GetStringSlice(name)
		return v
	}
	return viper.GetStringSlice(viperKey)
}

func intFlag(cmd *cobra.Command, name, viperKey string) int {
	if cmd.Flags().Changed(name) {
		v, _ := cmd.Flags().GetInt(name)
		return v
	}
	return viper.GetInt(viperKey)
}

func boolFlag(cmd *cobra.Command, name, viperKey string) bool {
	if cmd.Flags().Changed(name) {
		v, _ := cmd.Flags().GetBool(name)
		return v
	}
	return viper.GetBool(viperKey)
}

func float64Flag(cmd *cobra.Command, name, viperKey string) float64 {
	if cmd.Flags().Changed(name) {
		v, _ := cmd.Flags().GetFloat64(name)
		return v
	}
	return viper.GetFloat64(viperKey)
}

// runTranslation executes the shared load → translate → apply → save pipeline
// used by every provider subcommand.
func runTranslation(g globalOptions, providerName string, provider model.TranslationProvider, timeout time.Duration) error {
	if g.InputFile == "" {
		return fmt.Errorf("input file path is required; set --input or global.input_file in the config")
	}
	if g.OutputFile == "" {
		return fmt.Errorf("output file path is required; set --output or global.output_file in the config")
	}

	if g.Verbose {
		fmt.Printf("Starting %s with:\n", providerName)
		fmt.Printf("  Input file: %s\n", g.InputFile)
		fmt.Printf("  Output file: %s\n", g.OutputFile)
		fmt.Printf("  Source language: %s\n", g.SourceLang)
		fmt.Printf("  Target languages: %v\n", g.TargetLangs)
		fmt.Printf("  Concurrency: %d\n", g.Concurrency)
		fmt.Println("Loading xcstrings file...")
	}

	xcstrings, err := model.LoadXCStrings(g.InputFile)
	if err != nil {
		return fmt.Errorf("loading xcstrings file: %w", err)
	}

	if g.SourceLang != "" {
		xcstrings.SourceLanguage = g.SourceLang
	}

	service := translator.NewTranslationService(provider, g.Concurrency, timeout)
	ctx := context.Background()

	if g.Verbose {
		fmt.Println("Starting translation...")
	}

	var responses []model.TranslationResponse
	for _, target := range g.TargetLangs {
		reqs := translator.CreateTranslationRequestsForLanguage(xcstrings, target)
		if len(reqs) == 0 {
			continue
		}

		if g.Verbose {
			fmt.Printf("Translating to %s (%d strings)...\n", target, len(reqs))
		}

		progress := translator.NewVerboseProgressReporter(target, len(reqs), g.Verbose)
		batch, err := service.TranslateBatch(ctx, reqs, progress)
		responses = append(responses, batch...)
		if err != nil {
			return fmt.Errorf("translation failed for %s: %w", target, err)
		}
	}

	if len(responses) == 0 {
		fmt.Println("No strings to translate. Exiting.")
		return nil
	}

	successCount, errorCount := 0, 0
	for _, resp := range responses {
		if resp.Error != nil {
			if g.Verbose {
				fmt.Printf("Error translating %s to %s: %v\n", resp.Key, resp.TargetLanguage, resp.Error)
			}
			errorCount++
		} else {
			successCount++
		}
	}

	if g.Verbose {
		fmt.Printf("Translation completed: %d successful, %d failed\n", successCount, errorCount)
	}

	if errorCount > 0 {
		fmt.Println("Errors detected during translation. Stopping without applying translations.")
		return nil
	}

	if g.Verbose {
		fmt.Println("Applying translations...")
	}
	translator.ApplyTranslations(xcstrings, responses)

	if g.Verbose {
		fmt.Printf("Saving output to %s...\n", g.OutputFile)
	}
	if err := model.SaveXCStrings(g.OutputFile, xcstrings); err != nil {
		return fmt.Errorf("saving output file: %w", err)
	}

	fmt.Println("Translation completed successfully!")
	fmt.Printf("Results saved to: %s\n", g.OutputFile)
	return nil
}
