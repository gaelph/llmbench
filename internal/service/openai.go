package service

import (
	"context"
	"fmt"
	"time"

	"llmbench/internal/models"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAIService wraps the OpenAI client for benchmark operations
type OpenAIService struct {
	client   openai.Client
	provider models.Provider
	timeout  time.Duration
}

// NewOpenAIService creates a new OpenAI service instance
func NewOpenAIService(provider models.Provider, timeout time.Duration) *OpenAIService {
	opts := []option.RequestOption{
		option.WithAPIKey(provider.APIKey),
	}

	// Set custom base URL if different from OpenAI's default
	if provider.BaseURL != "" && provider.BaseURL != "https://api.openai.com/v1" {
		opts = append(opts, option.WithBaseURL(provider.BaseURL))
	}

	client := openai.NewClient(opts...)

	return &OpenAIService{
		client:   client,
		provider: provider,
		timeout:  timeout,
	}
}

// SendChatCompletion sends a chat completion request and measures performance
func (s *OpenAIService) SendChatCompletion(ctx context.Context, request models.BenchmarkRequest) models.BenchmarkResult {
	start := time.Now()

	result := models.BenchmarkResult{
		Provider: s.provider.Name,
	}

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Convert our messages to OpenAI format
	messages := make([]openai.ChatCompletionMessageParamUnion, len(request.Messages))
	for i, msg := range request.Messages {
		switch msg.Role {
		case "user":
			messages[i] = openai.UserMessage(msg.Content)
		case "assistant":
			messages[i] = openai.AssistantMessage(msg.Content)
		case "system":
			messages[i] = openai.SystemMessage(msg.Content)
		default:
			messages[i] = openai.UserMessage(msg.Content)
		}
	}

	// Prepare the chat completion request
	chatRequest := openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    s.provider.Model,
	}

	if request.MaxTokens > 0 {
		chatRequest.MaxTokens = openai.Int(int64(request.MaxTokens))
	}

	// Send the request
	response, err := s.client.Chat.Completions.New(timeoutCtx, chatRequest)

	result.ResponseTime = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}

	result.Success = true

	// Extract response content
	if len(response.Choices) > 0 && response.Choices[0].Message.Content != "" {
		result.Response = response.Choices[0].Message.Content
	}

	// Extract token usage if available
	if response.Usage.TotalTokens > 0 {
		result.TokensUsed = int(response.Usage.TotalTokens)
	}

	return result
}

// TestConnection tests the connection to the provider
func (s *OpenAIService) TestConnection(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Send a simple test message
	testRequest := models.BenchmarkRequest{
		Messages: []models.ChatMessage{
			{
				Role:    "user",
				Content: "Hello, this is a connection test. Please respond with 'OK'.",
			},
		},
		Model:     s.provider.Model,
		MaxTokens: 20,
	}

	result := s.SendChatCompletion(timeoutCtx, testRequest)
	if !result.Success {
		return fmt.Errorf("connection test failed: %s", result.Error)
	}

	return nil
}

// SendChatCompletionStream sends a streaming chat completion request and measures performance
func (s *OpenAIService) SendChatCompletionStream(ctx context.Context, request models.BenchmarkRequest) models.BenchmarkResult {
	start := time.Now()

	result := models.BenchmarkResult{
		Provider:    s.provider.Name,
		IsStreaming: true,
	}

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Convert our messages to OpenAI format
	messages := make([]openai.ChatCompletionMessageParamUnion, len(request.Messages))
	for i, msg := range request.Messages {
		switch msg.Role {
		case "user":
			messages[i] = openai.UserMessage(msg.Content)
		case "assistant":
			messages[i] = openai.AssistantMessage(msg.Content)
		case "system":
			messages[i] = openai.SystemMessage(msg.Content)
		default:
			messages[i] = openai.UserMessage(msg.Content)
		}
	}

	// Prepare the streaming chat completion request
	chatRequest := openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    s.provider.Model,
	}

	if request.MaxTokens > 0 {
		chatRequest.MaxTokens = openai.Int(int64(request.MaxTokens))
	}

	// Send the streaming request
	stream := s.client.Chat.Completions.NewStreaming(timeoutCtx, chatRequest)
	defer stream.Close()

	var responseContent string
	var tokenCount int
	var firstTokenTime time.Time
	var lastTokenTime time.Time
	firstToken := true

	// Process the stream
	for stream.Next() {
		chunk := stream.Current()
		
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			if firstToken {
				firstTokenTime = time.Now()
				result.TimeToFirstToken = firstTokenTime.Sub(start)
				firstToken = false
			}
			
			responseContent += chunk.Choices[0].Delta.Content
			tokenCount++
			lastTokenTime = time.Now()
		}
	}

	// Check for streaming errors
	if err := stream.Err(); err != nil {
		result.Success = false
		result.Error = err.Error()
		result.ResponseTime = time.Since(start)
		return result
	}

	// Calculate final metrics
	result.Success = true
	result.ResponseTime = time.Since(start)
	result.Response = responseContent
	result.StreamingTokens = tokenCount
	
	if !firstTokenTime.IsZero() && !lastTokenTime.IsZero() {
		streamingDuration := lastTokenTime.Sub(firstTokenTime)
		result.StreamingDuration = streamingDuration
		
		// Calculate token throughput (tokens per second)
		if streamingDuration.Seconds() > 0 {
			result.TokenThroughput = float64(tokenCount) / streamingDuration.Seconds()
		}
	}

	return result
}

// GetProviderInfo returns information about the provider
func (s *OpenAIService) GetProviderInfo() models.Provider {
	return s.provider
}
