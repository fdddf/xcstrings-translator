package cmd

import (
	"fmt"
	"os"

	"github.com/fdddf/xcstrings-translator/internal/config"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long:  `Commands for managing configuration files.`,
}

// generateCmd represents the generate subcommand
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a default config.yaml file",
	Long:  `Generate a default config.yaml file with example configuration values.`,
	Run: func(cmd *cobra.Command, args []string) {
		generateConfig()
	},
	// This prevents the normal config file loading for this command
	// by bypassing the PersistentPreRun that would normally run
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Do nothing - this prevents the root's PersistentPreRun from executing
	},
}

func init() {
	configCmd.AddCommand(generateCmd)

	// Add the config command to the root command
	rootCmd.AddCommand(configCmd)
}

func generateConfig() {
	cfg := config.DefaultConfig()

	// Create the default configuration with example values
	yamlContent := `# Global configuration settings
global:
  input_file: "` + cfg.Global.InputFile + `"
  output_file: "` + cfg.Global.OutputFile + `"
  source_language: "` + cfg.Global.SourceLanguage + `"
  target_languages:
    - "` + cfg.Global.TargetLanguages[0] + `"
    - "ja"
    - "ko"
  concurrency: ` + fmt.Sprintf("%d", cfg.Global.Concurrency) + `
  verbose: ` + fmt.Sprintf("%t", cfg.Global.Verbose) + `

# Google Translate API configuration
google:
  api_key: "your-google-api-key-here"
  model: "` + cfg.Google.Model + `"
  glossary: ""

# DeepL API configuration
deepl:
  api_key: "your-deepl-api-key-here"
  is_free: ` + fmt.Sprintf("%t", cfg.DeepL.IsFree) + `
  formality: "` + cfg.DeepL.Formality + `"

# Baidu Translate API configuration
baidu:
  app_id: "your-baidu-app-id-here"
  app_secret: "your-baidu-app-secret-here"

# OpenAI compatible API configuration
openai:
  api_key: "your-openai-api-key-here"
  api_base_url: "` + cfg.OpenAI.APIBaseURL + `"
  model: "` + cfg.OpenAI.Model + `"
  temperature: ` + fmt.Sprintf("%.1f", cfg.OpenAI.Temperature) + `
  max_tokens: ` + fmt.Sprintf("%d", cfg.OpenAI.MaxTokens) + `
`

	// Write the config file
	err := os.WriteFile("config.yaml", []byte(yamlContent), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generated config.yaml with default values.")
	fmt.Println("Please edit the file with your API keys and settings.")
}
