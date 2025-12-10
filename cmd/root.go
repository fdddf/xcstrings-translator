package cmd

import (
	"fmt"
	"os"

	"github.com/fdddf/xcstrings-translator/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	Version = "0.1.3"
	Build   = "2025-12-08"
	Commit  = "0.0.1"
)

var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:   "xcstrings-translator",
	Short: "A CLI tool to translate Localizable.xcstrings files using multiple translation providers",
	Long: `xcstrings-translator is a powerful command-line tool that translates Localizable.xcstrings files
using various translation providers including Google Translate, DeepL, Baidu Translate, and OpenAI compatible APIs.

It supports concurrent translation requests for improved performance and allows configuration
of provider-specific parameters through command-line flags or configuration files.`,
	Version: getVersion(),
}

func getVersion() string {
	return fmt.Sprintf("%s (%s) %s", Version, Build, Commit)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}

// ExecuteGUI runs the CLI defaulting to the gui subcommand (used by GUI-only builds).
func ExecuteGUI() {
	rootCmd.SetArgs([]string{"gui"})
	Execute()
}

func init() {
	// Initialize default configuration
	cfg = config.DefaultConfig()

	// Initialize Viper
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "Config file (default is config.yaml)")
	rootCmd.PersistentFlags().StringP("input", "i", "", "Input xcstrings file path")
	rootCmd.PersistentFlags().StringP("output", "o", "", "Output xcstrings file path")
	rootCmd.PersistentFlags().StringP("source-language", "s", "", "Source language code (e.g., en, zh-Hans)")
	rootCmd.PersistentFlags().StringSliceP("target-languages", "t", []string{}, "Target language codes (e.g., zh-Hans, ja, ko)")
	rootCmd.PersistentFlags().Int("concurrency", 0, "Number of concurrent translation requests")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")

	// Bind flags to Viper
	viper.BindPFlag("global.input_file", rootCmd.PersistentFlags().Lookup("input"))
	viper.BindPFlag("global.output_file", rootCmd.PersistentFlags().Lookup("output"))
	viper.BindPFlag("global.source_language", rootCmd.PersistentFlags().Lookup("source-language"))
	viper.BindPFlag("global.target_languages", rootCmd.PersistentFlags().Lookup("target-languages"))
	viper.BindPFlag("global.concurrency", rootCmd.PersistentFlags().Lookup("concurrency"))
	viper.BindPFlag("global.verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	// Set the config file name (without extension) - Viper will search for various formats
	configFile, _ := rootCmd.PersistentFlags().GetString("config")

	if configFile != "" {
		// Use the config file from the flag
		viper.SetConfigFile(configFile)
	} else {
		// Search for config in the current directory with name "config"
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
	}

	// Read in environment variables that match
	viper.AutomaticEnv()

	// Try to read the config file
	if err := viper.ReadInConfig(); err == nil {
		fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	}

	// Unmarshal the configuration into the config struct
	if err := viper.Unmarshal(&cfg); err != nil {
		fmt.Printf("Error parsing config: %v\n", err)
	}
}
