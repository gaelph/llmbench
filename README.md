# LLMBench

A powerful CLI tool for benchmarking and comparing LLM service providers that use OpenAI-compliant APIs. LLMBench supports multiple providers, concurrent requests, and provides detailed performance metrics with both CLI and interactive TUI modes.

## Features

- 🚀 **Multiple Provider Support**: Test OpenAI, Anthropic, Azure OpenAI, and any OpenAI-compatible API
- ⚡ **Concurrent Testing**: Configure concurrent requests for realistic load testing
- 📊 **Detailed Metrics**: Response times, success rates, token usage, and error analysis
- 🎨 **Interactive TUI**: Beautiful terminal interface powered by BubbleTea
- 📝 **Flexible Configuration**: YAML configuration with environment variable support
- 🔧 **Easy Setup**: Simple configuration initialization and validation

## Installation

### From Source

```bash
git clone https://github.com/your-username/llmbench.git
cd llmbench
go build -o llmbench .
```

### Using Go Install

```bash
go install github.com/your-username/llmbench@latest
```

## Quick Start

1. **Initialize Configuration**
   ```bash
   llmbench config init
   ```

2. **Edit Configuration**
   Edit the generated `llmbench.yaml` file with your API keys:
   ```yaml
   benchmark:
     providers:
       - name: openai
         base_url: https://api.openai.com/v1
         api_key: your-openai-api-key
         model: gpt-3.5-turbo
       - name: anthropic
         base_url: https://api.anthropic.com/v1
         api_key: your-anthropic-api-key
         model: claude-3-haiku-20240307
     concurrency: 2
     requests: 50
     timeout: 30s
   ```

3. **Test Connections**
   ```bash
   llmbench test
   ```

4. **Run Benchmark**
   ```bash
   # CLI mode
   llmbench benchmark -m "Hello, how are you?"
   
   # Interactive TUI mode
   llmbench benchmark -i
   ```

## Usage

### Commands

#### `config` - Configuration Management

```bash
# Initialize a new configuration file
llmbench config init [path]

# Show current configuration
llmbench config show

# Validate configuration
llmbench config validate
```

#### `test` - Connection Testing

```bash
# Test connections to all configured providers
llmbench test
```

#### `benchmark` - Run Benchmarks

```bash
# Basic benchmark
llmbench benchmark -m "Your test message"

# Custom parameters
llmbench benchmark \
  --message "Hello, world!" \
  --requests 100 \
  --concurrent 5 \
  --max-tokens 150

# Interactive mode
llmbench benchmark --interactive

# JSON output
llmbench benchmark -m "Test" --json
```

### Configuration

#### Configuration File Locations

LLMBench looks for configuration files in the following order:
1. File specified with `--config` flag
2. `./llmbench.yaml` (current directory)
3. `~/.config/llmbench/llmbench.yaml`
4. `/etc/llmbench/llmbench.yaml`

#### Configuration Structure

```yaml
benchmark:
  providers:
    - name: provider-name          # Unique identifier
      base_url: https://api.url    # API endpoint
      api_key: your-api-key        # API key
      model: model-name            # Model to use
  concurrency: 2                  # Concurrent requests
  requests: 50                     # Total requests per provider
  timeout: 30s                     # Request timeout
```

#### Environment Variables

You can override configuration values using environment variables with the `LLMBENCH_` prefix:

```bash
export LLMBENCH_BENCHMARK_CONCURRENCY=5
export LLMBENCH_BENCHMARK_REQUESTS=100
```

### Supported Providers

LLMBench works with any OpenAI-compatible API. Here are some examples:

#### OpenAI
```yaml
- name: openai
  base_url: https://api.openai.com/v1
  api_key: sk-...
  model: gpt-3.5-turbo
```

#### Anthropic
```yaml
- name: anthropic
  base_url: https://api.anthropic.com/v1
  api_key: sk-ant-...
  model: claude-3-haiku-20240307
```

#### Azure OpenAI
```yaml
- name: azure-openai
  base_url: https://your-resource.openai.azure.com/
  api_key: your-azure-key
  model: gpt-35-turbo
```

#### Local/Self-hosted
```yaml
- name: local-llm
  base_url: http://localhost:8080/v1
  api_key: not-needed
  model: llama-2-7b
```

## Output Formats

### CLI Output

```
Starting benchmark...
Message: Hello, how are you?
Requests per provider: 50
Concurrency: 2

Testing connections...
✅ openai: Connected
✅ anthropic: Connected

Running benchmark...
openai: 50/50 completed ✅
anthropic: 50/50 completed ✅

