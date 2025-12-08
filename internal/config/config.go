package config

// Config represents the application configuration
type Config struct {
	Global GlobalConfig `mapstructure:"global"`
	Google GoogleConfig `mapstructure:"google"`
	DeepL  DeepLConfig  `mapstructure:"deepl"`
	Baidu  BaiduConfig  `mapstructure:"baidu"`
	OpenAI OpenAIConfig `mapstructure:"openai"`
}

// GlobalConfig contains global configuration settings
type GlobalConfig struct {
	InputFile       string   `mapstructure:"input_file"`
	OutputFile      string   `mapstructure:"output_file"`
	SourceLanguage  string   `mapstructure:"source_language"`
	TargetLanguages []string `mapstructure:"target_languages"`
	Concurrency     int      `mapstructure:"concurrency"`
	Verbose         bool     `mapstructure:"verbose"`
}

// GoogleConfig contains Google Translate configuration
type GoogleConfig struct {
	APIKey   string `mapstructure:"api_key"`
	Model    string `mapstructure:"model"`
	Glossary string `mapstructure:"glossary"`
}

// DeepLConfig contains DeepL configuration
type DeepLConfig struct {
	APIKey    string `mapstructure:"api_key"`
	IsFree    bool   `mapstructure:"is_free"`
	Formality string `mapstructure:"formality"`
}

// BaiduConfig contains Baidu Translate configuration
type BaiduConfig struct {
	AppID     string `mapstructure:"app_id"`
	AppSecret string `mapstructure:"app_secret"`
}

// OpenAIConfig contains OpenAI configuration
type OpenAIConfig struct {
	APIKey      string  `mapstructure:"api_key"`
	APIBaseURL  string  `mapstructure:"api_base_url"`
	Model       string  `mapstructure:"model"`
	Temperature float64 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		Global: GlobalConfig{
			InputFile:       "Localizable.xcstrings",
			OutputFile:      "Localizable_translated.xcstrings",
			SourceLanguage:  "en",
			TargetLanguages: []string{"zh-Hans"},
			Concurrency:     5,
			Verbose:         false,
		},
		Google: GoogleConfig{
			Model: "nmt",
		},
		DeepL: DeepLConfig{
			IsFree:    false,
			Formality: "default",
		},
		Baidu: BaiduConfig{},
		OpenAI: OpenAIConfig{
			APIBaseURL:  "https://api.openai.com",
			Model:       "gpt-3.5-turbo",
			Temperature: 0.3,
			MaxTokens:   1024,
		},
	}
}
