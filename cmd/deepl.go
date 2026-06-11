package cmd

import (
	"time"

	"github.com/fdddf/xcstrings-translator/internal/translator"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deeplCmd = &cobra.Command{
	Use:   "deepl",
	Short: "Translate xcstrings using DeepL API",
	Long: `Translate Localizable.xcstrings file using DeepL API.
	
Requires a valid DeepL API key. Configuration can be provided
via command line flags, config file, or environment variables.`,
	RunE: runDeepLTranslate,
}

func init() {
	rootCmd.AddCommand(deeplCmd)

	// DeepL specific flags
	deeplCmd.Flags().String("api-key", "", "DeepL API key (required)")
	deeplCmd.Flags().Bool("free", false, "Use DeepL free API tier")
	deeplCmd.Flags().String("formality", "", "Formality level (default, more, less)")

	// Bind flags to Viper
	viper.BindPFlag("deepl.api_key", deeplCmd.Flags().Lookup("api-key"))
	viper.BindPFlag("deepl.is_free", deeplCmd.Flags().Lookup("free"))
	viper.BindPFlag("deepl.formality", deeplCmd.Flags().Lookup("formality"))
}

func runDeepLTranslate(cmd *cobra.Command, args []string) error {
	g := resolveGlobalOptions(cmd)
	apiKey := stringFlag(cmd, "api-key", "deepl.api_key")
	isFree := boolFlag(cmd, "free", "deepl.is_free")

	provider := translator.NewDeepLTranslator(apiKey, isFree)
	return runTranslation(g, "DeepL Translate", provider, 300*time.Second)
}
