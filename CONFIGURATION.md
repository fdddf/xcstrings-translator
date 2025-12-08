# Configuration Guide

The xcstrings-translator now supports configuration via YAML files, environment variables, and command-line flags. The configuration system uses Viper, which provides flexible configuration management with multiple sources and precedence.

## Configuration Precedence

Configuration values are loaded with the following precedence (highest to lowest):
1. Command-line flags
2. Environment variables
3. Configuration file
4. Default values

## Configuration File Format

The default configuration file is `config.yaml` in the current directory, but you can specify a different file using the `--config` flag.

Example `config.yaml`:

```yaml
# Global configuration settings
global:
  input_file: "Localizable.xcstrings"
  output_file: "Localizable_translated.xcstrings"
  source_language: "en"
  target_languages:
    - "zh-Hans"
    - "ja"
    - "ko"
  concurrency: 5
  verbose: false

# Google Translate API configuration
google:
  api_key: "your-google-api-key-here"
  model: "nmt"
  glossary: ""

# DeepL API configuration
deepl:
  api_key: "your-deepl-api-key-here"
  is_free: false
  formality: "default"

# Baidu Translate API configuration
baidu:
  app_id: "your-baidu-app-id-here"
  app_secret: "your-baidu-app-secret-here"

# OpenAI compatible API configuration
openai:
  api_key: "your-openai-api-key-here"
  api_base_url: "https://api.openai.com"
  model: "gpt-3.5-turbo"
  temperature: 0.3
  max_tokens: 1024
```

## Environment Variables

All configuration options can also be set using environment variables. The environment variables follow the pattern: `XSTRANSLATOR_<SECTION>_<KEY>` (in uppercase with underscores).

Examples:
- `XSTRANSLATOR_GLOBAL_INPUT_FILE`
- `XSTRANSLATOR_GOOGLE_API_KEY`
- `XSTRANSLATOR_DEEPL_API_KEY`
- `XSTRANSLATOR_BAIDU_APP_ID`
- `XSTRANSLATOR_OPENAI_API_KEY`

## Usage Examples

### Using Configuration File Only
```bash
# Use default config.yaml
xcstrings-translator google

# Use specific config file
xcstrings-translator --config myconfig.yaml google
```

### Combining Configuration File with Command-Line Flags
```bash
# Override input file from command line
xcstrings-translator --input custom.xcstrings google

# Override target languages
xcstrings-translator -t "es" -t "fr" google
```

### Using Environment Variables
```bash
export XSTRANSLATOR_GOOGLE_API_KEY="your-api-key"
export XSTRANSLATOR_GLOBAL_TARGET_LANGUAGES="zh-Hans,ja,ko"
xcstrings-translator google
```

## Available Configuration Options

### Global Options
- `input_file`: Path to input xcstrings file (default: "Localizable.xcstrings")
- `output_file`: Path to output xcstrings file (default: "Localizable_translated.xcstrings")
- `source_language`: Source language code (default: "en")
- `target_languages`: List of target language codes
- `concurrency`: Number of concurrent translation requests (default: 5)
- `verbose`: Enable verbose output (default: false)

### Google Translate Options
- `api_key`: Google Cloud API key (required)
- `model`: Translation model ("nmt" or "base", default: "nmt")
- `glossary`: Glossary to use for translation (default: "")

### DeepL Options
- `api_key`: DeepL API key (required)
- `is_free`: Use DeepL free API tier (default: false)
- `formality`: Formality level ("default", "more", "less", default: "default")

### Baidu Translate Options
- `app_id`: Baidu Translate AppID (required)
- `app_secret`: Baidu Translate AppSecret (required)

### OpenAI Options
- `api_key`: OpenAI API key (required)
- `api_base_url`: API base URL (default: "https://api.openai.com")
- `model`: Model to use for translation (default: "gpt-3.5-turbo")
- `temperature`: Temperature for translation (default: 0.3)
- `max_tokens`: Maximum tokens for translation (default: 1024)
