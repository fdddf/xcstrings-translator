package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "xcstrings-translator",
	Short: "A CLI tool to translate Localizable.xcstrings files using multiple translation providers",
	Long: `xcstrings-translator is a powerful command-line tool that translates Localizable.xcstrings files
using various translation providers including Google Translate, DeepL, Baidu Translate, and OpenAI compatible APIs.

It supports concurrent translation requests for improved performance and allows configuration
of provider-specific parameters through command-line flags.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringP("input", "i", "Localizable.xcstrings", "Input xcstrings file path")
	rootCmd.PersistentFlags().StringP("output", "o", "Localizable_translated.xcstrings", "Output xcstrings file path")
	rootCmd.PersistentFlags().StringP("source-language", "s", "en", "Source language code (e.g., en, zh-Hans)")
	rootCmd.PersistentFlags().StringSliceP("target-languages", "t", []string{"zh-Hans"}, "Target language codes (e.g., zh-Hans, ja, ko)")
	rootCmd.PersistentFlags().IntP("concurrency", "c", 5, "Number of concurrent translation requests")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
}
