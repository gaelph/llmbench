package service

import (
	"context"
	"fmt"
	"time"

	"llmbench/internal/models"
	"llmbench/internal/utils"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAIService wraps the OpenAI client for benchmark operations
type OpenAIService struct {
	client       openai.Client
	provider     models.Provider
	timeout      time.Duration
	tokenCounter *utils.TokenCounter
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

	// Initialize token counter
	tokenCounter, err := utils.NewTokenCounter()
	if err != nil {
		// Log error but don't fail - we can still function without token counting
		fmt.Printf("Warning: Failed to initialize token counter: %v\n", err)
	}

	return &OpenAIService{
		client:       client,
		provider:     provider,
		timeout:      timeout,
		tokenCounter: tokenCounter,
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
		Model:    request.Model,
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

	// Calculate token usage using our token counter
	if s.tokenCounter != nil {
		// Count input tokens
		inputTokens := s.tokenCounter.CountChatCompletionTokens(request.Messages, request.Model)
		
		// Count output tokens
		outputTokens := 0
		if result.Response != "" {
			outputTokens = s.tokenCounter.CountTokens(result.Response)
		}
		
		result.TokensUsed = inputTokens + outputTokens
	} else if response.Usage.TotalTokens > 0 {
		// Fallback to OpenAI's token count if our counter is not available
		result.TokensUsed = int(response.Usage.TotalTokens)
	}

	return result
}

// TestConnection tests the connection to the provider
func (s *OpenAIService) TestConnection(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Use the first model for connection testing
	if len(s.provider.Models) == 0 {
		return fmt.Errorf("no models configured for provider %s", s.provider.Name)
	}

	// Send a simple test message
	testRequest := models.BenchmarkRequest{
		Messages: []models.ChatMessage{
			{
				Role:    "user",
				Content: "Hello, this is a connection test. Please respond with 'OK'.",
			},
		},
		Model:     s.provider.Models[0],
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
		Model:    request.Model,
	}

	if request.MaxTokens > 0 {
		chatRequest.MaxTokens = openai.Int(int64(request.MaxTokens))
	}

	// Send the streaming request
	stream := s.client.Chat.Completions.NewStreaming(timeoutCtx, chatRequest)
	defer stream.Close()

	var responseContent string
	var chunkCount int
	var firstTokenTime time.Time
	var streamEndTime time.Time
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
			chunkCount++
		}
	}
	
	// Mark the end of streaming
	streamEndTime = time.Now()

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
	
	// Calculate proper token counts using our token counter
	var totalTokens int
	var outputTokens int
	
	if s.tokenCounter != nil {
		// Count input tokens
		inputTokens := s.tokenCounter.CountChatCompletionTokens(request.Messages, request.Model)
		
		// Count output tokens from the complete response
		if responseContent != "" {
			outputTokens = s.tokenCounter.CountTokens(responseContent)
		}
		
		totalTokens = inputTokens + outputTokens
		result.TokensUsed = totalTokens
	}
	
	// Set streaming-specific metrics
	result.StreamingTokens = outputTokens // Use actual token count, not chunk count
	
	// Calculate streaming duration and throughput properly
	if !firstTokenTime.IsZero() && !streamEndTime.IsZero() {
		// Calculate the total streaming duration from first token to end of stream
		streamingDuration := streamEndTime.Sub(firstTokenTime)
		result.StreamingDuration = streamingDuration
		
		// Calculate token throughput (tokens per second) using actual output tokens
		// Only calculate if we have a reasonable duration (at least 1ms) and output tokens
		if streamingDuration.Milliseconds() > 0 && outputTokens > 0 {
			result.TokenThroughput = float64(outputTokens) / streamingDuration.Seconds()
		}
	}

	return result
}

// GetProviderInfo returns information about the provider
func (s *OpenAIService) GetProviderInfo() models.Provider {
	return s.provider
}
