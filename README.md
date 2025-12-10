# xcstrings-translator

## 🚀 Project Overview

xcstrings-translator is a powerful command-line tool specifically designed for translating Localizable.xcstrings files for iOS/macOS applications. This tool supports multiple translation service providers and boasts high-performance concurrent translation capabilities.

**Read this in other languages: [中文](README_zh.md)**


## ✨ Core Functionality

### 🔌 Multi-Translation Service Support

- **Google Translate API**: Supports neural machine translation models
- **DeepL API**: Provides high-quality translation, supporting both free and professional versions
- **Baidu Translate API**: Baidu Translate service
- **OpenAI API**: Supports translation capabilities of GPT series models

### ⚡ High-Performance Concurrency

- Concurrency control based on Worker Pool mode
- Configurable number of concurrent requests
- Elegant error handling and retry mechanism
- Context timeout control

### 📁 xcstrings File Processing

- Complete parsing and generation of xcstrings JSON format
- Intelligent detection of strings requiring translation
- Preserve original translations, translating only missing language versions
- Maintain file structure and metadata integrity

### ⚙️ Flexible Configuration

- Support for YAML configuration files
- Environment variable support
- Command-line flag overrides

## 🛠️ Technical Implementation

### 🏗️ Architecture design
```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   CLI Layer     │     │  Service Layer  │     │ Provider Layer  │
│  (Cobra Commands)│────▶│ (Concurrency &  │────▶│ (Translation    │
│                 │     │   Translation)  │     │  Implementations)│
└─────────────────┘     └─────────────────┘     └─────────────────┘
        ▲                       ▲                       ▲
        │                       │                       │
        ▼                       ▼                       ▼
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   User Input    │     │  Model Layer    │     │  HTTP Client    │
│  (Flags/Args)   │     │ (Data Structures)│     │  (resty)        │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

### 📊 Concurrency Model

- Concurrency is achieved using Goroutines and Channels
- Concurrency is controlled using the Worker Pool pattern
- Timeout control is implemented using the Context mechanism
- WaitGroup waits for all tasks to complete

### 🔧 Main Technology Stack

- **Go 1.21+**: Main programming language
- **Cobra**: CLI framework
- **resty**: HTTP client
- **JSON**: xcstrings file format processing
- **MD5**: Baidu API signature generation

## Installation
To install, run:
```
go install github.com/fdddf/xcstrings-translator@latest
```

Or download the binary from the [releases page](https://github.com/fdddf/xcstrings-translator/releases).

## 📋 Usage Examples

### Using Configuration File
```bash
# Use default config.yaml
xcstrings-translator google

# Use specific config file
xcstrings-translator --config myconfig.yaml google

# Override specific settings from command line
xcstrings-translator --input custom.xcstrings -t "es" -t "fr" google
```

### Google Translate
```bash
xcstrings-translator google \ 
--api-key "AIzaSy..." \ 
--input "Localizable.xcstrings" \ 
--output "Localizable_zh.xcstrings" \ 
--source-language "en" \ 
--target-languages ​​"zh-Hans" "ja" \ 
--concurrency 10 \ 
--verbose
```

### DeepL
```bash
xcstrings-translator deepl \ 
--api-key "2a7f4..." \ 
--free \ 
--input "Localizable.xcstrings" \ 
--output "Localizable_translated.xcstrings" \ 
--target-languages ​​"zh-Hans"
```

### Baidu Translate
```bash
xcstrings-translator baidu \ 
--app-id "2024..." \ 
--app-secret "f4K..." --input "Localizable.xcstrings" --output "Localizable_baidu.xcstrings"

```

### OpenAI
```bash
xcstrings-translator openai
--api-key "sk-proj..."
--model "gpt-4"
--input "Localizable.xcstrings" --output "Localizable_ai.xcstrings"

```

### Visual Web UI
```bash
# Build the Vue/Tailwind UI (once, or after editing web/)
cd web && npm install && npm run build

# Start the embedded Fiber server
xcstrings-translator serve --addr :8080
```

Upload a `Localizable.xcstrings` file, choose target languages, run batch translation with your provider keys, and export the updated file directly from the browser. Progress is streamed; translated keys appear in the grid in real time so you don’t lose work if rate limits interrupt a long run.

## 🔒 Security Features

- API keys are passed via command-line arguments or environment variables
- No sensitive information is stored
- HTTPS encrypted transmission
- Input validation and error handling

## 📈 Performance Optimizations

- Connection pool reuse
- Batch request processing
- Intelligent retry mechanism
- Efficient memory management

## 🎯 Applicable Scenarios

- iOS/macOS application localization
- Batch translation of string resources
- CI/CD pipeline integration
- Multilingual application development

## 📚 Scalability

- Easy addition of new translation service providers
- Support for custom translation rules
- Integration into automated workflows
- Support for batch translation of large projects

## 🔮 Future Feature Plans

- [ ] Translation caching mechanism
- [ ] Translation quality assessment
- [ ] Batch file processing
- [ ] Translation memory
- [ ] Interactive translation confirmation

## 🤝 Contribution Guidelines Contributions, problem reporting, and suggestions are welcome. The project uses a standard GitHub workflow:

1. Fork the project
2. Create a feature branch
3. Submit changes
4. Create a pull request

## 📄 License This project is licensed under the MIT license. Please see the LICENSE file for details.
