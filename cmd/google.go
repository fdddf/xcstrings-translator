package cmd

import (
	"time"

	"github.com/fdddf/xcstrings-translator/internal/translator"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var googleCmd = &cobra.Command{
	Use:   "google",
	Short: "Translate xcstrings using Google Translate API",
	Long: `Translate Localizable.xcstrings file using Google Translate API.
	
Requires a valid Google Cloud API key with Translate API enabled. Configuration can be provided
via command line flags, config file, or environment variables.`,
	RunE: runGoogleTranslate,
}

func init() {
	rootCmd.AddCommand(googleCmd)

	// Google specific flags
	googleCmd.Flags().String("api-key", "", "Google Cloud API key (required)")
	googleCmd.Flags().String("model", "", "Translation model (nmt or base)")
	googleCmd.Flags().String("glossary", "", "Glossary to use for translation")

	// Bind flags to Viper
	viper.BindPFlag("google.api_key", googleCmd.Flags().Lookup("api-key"))
	viper.BindPFlag("google.model", googleCmd.Flags().Lookup("model"))
	viper.BindPFlag("google.glossary", googleCmd.Flags().Lookup("glossary"))
}

func runGoogleTranslate(cmd *cobra.Command, args []string) error {
	g := resolveGlobalOptions(cmd)
	apiKey := stringFlag(cmd, "api-key", "google.api_key")

	provider := translator.NewGoogleTranslator(apiKey)
	return runTranslation(g, "Google Translate", provider, 300*time.Second)
}
