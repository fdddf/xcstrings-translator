package cmd

import (
	"time"

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
	RunE: runOpenAITranslate,
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

func runOpenAITranslate(cmd *cobra.Command, args []string) error {
	g := resolveGlobalOptions(cmd)
	apiKey := stringFlag(cmd, "api-key", "openai.api_key")
	apiBaseURL := stringFlag(cmd, "api-base-url", "openai.api_base_url")
	model := stringFlag(cmd, "model", "openai.model")
	temperature := float64Flag(cmd, "temperature", "openai.temperature")
	maxTokens := intFlag(cmd, "max-tokens", "openai.max_tokens")

	provider := translator.NewOpenAITranslator(apiKey, apiBaseURL, model, temperature, maxTokens)
	// OpenAI can be slow, so allow a longer (10 minute) timeout.
	return runTranslation(g, "OpenAI Translate", provider, 600*time.Second)
}
