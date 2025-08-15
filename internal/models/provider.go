package models

import "time"

// Provider represents an LLM service provider configuration
type Provider struct {
	Name    string   `mapstructure:"name" yaml:"name"`
	BaseURL string   `mapstructure:"base_url" yaml:"base_url"`
	APIKey  string   `mapstructure:"api_key" yaml:"api_key"`
	Models  []string `mapstructure:"models" yaml:"models"`
}

// BenchmarkConfig represents the benchmark configuration
type BenchmarkConfig struct {
	Providers   []Provider `mapstructure:"providers" yaml:"providers"`
	Concurrency int        `mapstructure:"concurrency" yaml:"concurrency"`
	Requests    int        `mapstructure:"requests" yaml:"requests"`
	Timeout     string     `mapstructure:"timeout" yaml:"timeout"`
}

// BenchmarkRequest represents a single benchmark request
type BenchmarkRequest struct {
	Messages  []ChatMessage `json:"messages"`
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens,omitempty"`
	Stream    bool          `json:"stream,omitempty"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// BenchmarkResult represents the result of a benchmark test
type BenchmarkResult struct {
	Provider     string        `json:"provider"`
	ModelName    string        `json:"model_name"`
	Success      bool          `json:"success"`
	ResponseTime time.Duration `json:"response_time"`
	TokensUsed   int           `json:"tokens_used,omitempty"`
	Error        string        `json:"error,omitempty"`
	Response     string        `json:"response,omitempty"`
	
	// Streaming metrics
	IsStreaming       bool          `json:"is_streaming"`
	TimeToFirstToken  time.Duration `json:"time_to_first_token,omitempty"`
	TokenThroughput   float64       `json:"token_throughput,omitempty"` // tokens per second
	StreamingTokens   int           `json:"streaming_tokens,omitempty"`
	StreamingDuration time.Duration `json:"streaming_duration,omitempty"`
}

// BenchmarkSummary represents the summary of all benchmark results
type BenchmarkSummary struct {
	Provider        string        `json:"provider"`
	ModelName       string        `json:"model_name"`
	TotalRequests   int           `json:"total_requests"`
	SuccessfulReqs  int           `json:"successful_requests"`
	FailedRequests  int           `json:"failed_requests"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	MinResponseTime time.Duration `json:"min_response_time"`
	MaxResponseTime time.Duration `json:"max_response_time"`
	TotalTokens     int           `json:"total_tokens"`
	ErrorRate       float64       `json:"error_rate"`
	
	// Streaming metrics
	IsStreaming          bool          `json:"is_streaming,omitempty"`
	AvgTimeToFirstToken  time.Duration `json:"avg_time_to_first_token,omitempty"`
	MinTimeToFirstToken  time.Duration `json:"min_time_to_first_token,omitempty"`
	MaxTimeToFirstToken  time.Duration `json:"max_time_to_first_token,omitempty"`
	AvgTokenThroughput   float64       `json:"avg_token_throughput,omitempty"`
	MinTokenThroughput   float64       `json:"min_token_throughput,omitempty"`
	MaxTokenThroughput   float64       `json:"max_token_throughput,omitempty"`
}
