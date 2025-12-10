# xcstrings-translator

## 🚀 项目概述

xcstrings-translator是一个强大的命令行工具，专门用于翻译iOS/macOS应用的Localizable.xcstrings文件。该工具支持多种翻译服务提供商，具备高性能的并发翻译能力。

## ✨ 核心功能

### 🔌 多翻译服务支持
- **Google Translate API**: 支持神经机器翻译模型
- **DeepL API**: 提供高质量翻译，支持免费和专业版
- **Baidu Translate API**: 百度翻译服务
- **OpenAI API**: 支持GPT系列模型的翻译能力

### ⚡ 高性能并发
- 基于Worker Pool模式的并发控制
- 可配置的并发请求数量
- 优雅的错误处理和重试机制
- 上下文超时控制

### 📁 xcstrings文件处理
- 完整解析和生成xcstrings JSON格式
- 智能检测需要翻译的字符串
- 保留原有翻译，只翻译缺失的语言版本
- 保持文件结构和元数据完整性

## 🛠️ 技术实现

### 🏗️ 架构设计
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

### 📊 并发模型
- 使用Goroutine和Channel实现并发
- Worker Pool模式控制并发度
- Context机制实现超时控制
- WaitGroup等待所有任务完成

### 🔧 主要技术栈
- **Go 1.21+**: 主编程语言
- **Cobra**: CLI框架
- **resty**: HTTP客户端
- **JSON**: xcstrings文件格式处理
- **MD5**: Baidu API签名生成

## 安装
To install, run:
```
go install github.com/fdddf/xcstrings-translator@latest
```

Or download the binary from the [releases page](https://github.com/fdddf/xcstrings-translator/releases).

## 📋 使用示例

### Google Translate
```bash
xcstrings-translator google \
  --api-key "AIzaSy..." \
  --input "Localizable.xcstrings" \
  --output "Localizable_zh.xcstrings" \
  --source-language "en" \
  --target-languages "zh-Hans" "ja" \
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
  --target-languages "zh-Hans"
```

### Baidu Translate
```bash
xcstrings-translator baidu \
  --app-id "2024..." \
  --app-secret "f4K..." \
  --input "Localizable.xcstrings" \
  --output "Localizable_baidu.xcstrings"
```

### OpenAI
```bash
xcstrings-translator openai \
  --api-key "sk-proj..." \
  --model "gpt-4" \
  --input "Localizable.xcstrings" \
  --output "Localizable_ai.xcstrings"
```

### 可视化 Web UI
```bash
# 构建 Vue/Tailwind 前端（首次或修改 web/ 后）
cd web && npm install && npm run build

# 启动内置 Fiber 服务
xcstrings-translator serve --addr :8080
```

在浏览器中上传 `Localizable.xcstrings`，选择目标语言，填入各翻译提供商的密钥后运行批量翻译，并可直接导出结果。翻译进度实时推送，已翻译的条目会即时刷新，长任务遇到限流也不会丢失已完成的数据。

### 原生桌面应用（Windows/macOS/Linux）
无需浏览器，在本地窗口中使用同样的 UI：
```bash
xcstrings-translator gui --width 1400 --height 900
```

前置依赖：
- Windows：需要 WebView2 运行时（Win10/11 默认包含，缺失时请从微软官网安装）。
- macOS：使用系统内置 WebKit，无需额外安装。
- Linux：需要 WebKitGTK（如 Debian/Ubuntu 执行 `sudo apt install libwebkit2gtk-4.1-dev`）。

跨平台构建示例：
```bash
GOOS=darwin GOARCH=arm64 go build -o bin/xcstrings-translator-darwin ./...
GOOS=windows GOARCH=amd64 go build -o bin/xcstrings-translator.exe ./...
GOOS=linux GOARCH=amd64 go build -o bin/xcstrings-translator-linux ./...
```
这些构建依赖 CGO，请确保目标平台的工具链和 WebView 依赖已安装（macOS 需 Xcode Command Line Tools，Windows 需 WebView2 SDK/MinGW，Linux 需 WebKitGTK 开发包）。

## 🔒 安全特性
- API密钥通过命令行参数或环境变量传递
- 不存储敏感信息
- HTTPS加密传输
- 输入验证和错误处理

## 📈 性能优化
- 连接池复用
- 请求批处理
- 智能重试机制
- 内存高效处理

## 🎯 适用场景
- iOS/macOS应用本地化
- 批量翻译字符串资源
- CI/CD流水线集成
- 多语言应用开发

## 📚 扩展能力
- 易于添加新的翻译服务提供商
- 支持自定义翻译规则
- 可集成到自动化工作流
- 支持大型项目的分批次翻译

## 🔮 未来功能规划
- [ ] 翻译缓存机制
- [ ] 翻译质量评估
- [ ] 批量文件处理
- [ ] 翻译记忆库
- [ ] 交互式翻译确认

## 🤝 贡献指南
欢迎贡献代码、报告问题或提出建议。项目采用标准的GitHub工作流：
1. Fork项目
2. 创建特性分支
3. 提交更改
4. 创建Pull Request

## 📄 许可证
本项目采用MIT许可证，详情请参见LICENSE文件。
