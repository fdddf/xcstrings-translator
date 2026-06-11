package cmd

import (
	"time"

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
	g := resolveGlobalOptions(cmd)
	appID := stringFlag(cmd, "app-id", "baidu.app_id")
	appSecret := stringFlag(cmd, "app-secret", "baidu.app_secret")

	provider := translator.NewBaiduTranslator(appID, appSecret)
	return runTranslation(g, "Baidu Translate", provider, 300*time.Second)
}
