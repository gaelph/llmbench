package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"llmbench/internal/models"
)

// BenchmarkService orchestrates benchmark tests across multiple providers
type BenchmarkService struct {
	providers []models.Provider
	config    models.BenchmarkConfig
	timeout   time.Duration
}

// NewBenchmarkService creates a new benchmark service
func NewBenchmarkService(config models.BenchmarkConfig) (*BenchmarkService, error) {
	timeout, err := time.ParseDuration(config.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout duration: %w", err)
	}

	return &BenchmarkService{
		providers: config.Providers,
		config:    config,
		timeout:   timeout,
	}, nil
}

// TestConnections tests connectivity to all configured providers
func (bs *BenchmarkService) TestConnections(ctx context.Context) map[string]error {
	results := make(map[string]error)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, provider := range bs.providers {
		wg.Add(1)
		go func(p models.Provider) {
			defer wg.Done()
			
			service := NewOpenAIService(p, bs.timeout)
			err := service.TestConnection(ctx)
			
			mu.Lock()
			results[p.Name] = err
			mu.Unlock()
		}(provider)
	}

	wg.Wait()
	return results
}

// RunBenchmark executes benchmark tests for all providers
func (bs *BenchmarkService) RunBenchmark(ctx context.Context, request models.BenchmarkRequest, progressCallback func(string, int, int)) (map[string][]models.BenchmarkResult, error) {
	results := make(map[string][]models.BenchmarkResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, provider := range bs.providers {
		wg.Add(1)
		go func(p models.Provider) {
			defer wg.Done()
			
			providerResults := bs.runProviderBenchmark(ctx, p, request, progressCallback)
			
			mu.Lock()
			results[p.Name] = providerResults
			mu.Unlock()
		}(provider)
	}

	wg.Wait()
	return results, nil
}

// runProviderBenchmark runs benchmark for a single provider
func (bs *BenchmarkService) runProviderBenchmark(ctx context.Context, provider models.Provider, request models.BenchmarkRequest, progressCallback func(string, int, int)) []models.BenchmarkResult {
	service := NewOpenAIService(provider, bs.timeout)
	results := make([]models.BenchmarkResult, 0, bs.config.Requests)
	
	// Create semaphore for concurrency control
	semaphore := make(chan struct{}, bs.config.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	for i := 0; i < bs.config.Requests; i++ {
		wg.Add(1)
		go func(requestNum int) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			// Update request model to use provider's model
			providerRequest := request
			providerRequest.Model = provider.Model
			
			var result models.BenchmarkResult
			if providerRequest.Stream {
				result = service.SendChatCompletionStream(ctx, providerRequest)
			} else {
				result = service.SendChatCompletion(ctx, providerRequest)
			}
			
			mu.Lock()
			results = append(results, result)
			if progressCallback != nil {
				progressCallback(provider.Name, len(results), bs.config.Requests)
			}
			mu.Unlock()
		}(i)
	}
	
	wg.Wait()
	return results
}

// GenerateSummary creates a summary of benchmark results
func (bs *BenchmarkService) GenerateSummary(results map[string][]models.BenchmarkResult) map[string]models.BenchmarkSummary {
	summaries := make(map[string]models.BenchmarkSummary)
	
	for providerName, providerResults := range results {
		summary := models.BenchmarkSummary{
			Provider:      providerName,
			TotalRequests: len(providerResults),
		}
		
		var totalResponseTime time.Duration
		var totalTokens int
		var minTime, maxTime time.Duration
		var successCount int
		
		// Streaming metrics
		var isStreaming bool
		var totalTTFT time.Duration
		var minTTFT, maxTTFT time.Duration
		var totalThroughput float64
		var minThroughput, maxThroughput float64
		var streamingCount int
		
		for i, result := range providerResults {
			if result.Success {
				successCount++
				
				// Count tokens from both streaming and non-streaming
				if result.IsStreaming {
					totalTokens += result.StreamingTokens
					isStreaming = true
					
					// Track streaming metrics
					if result.TimeToFirstToken > 0 {
						totalTTFT += result.TimeToFirstToken
						streamingCount++
						
						if streamingCount == 1 || result.TimeToFirstToken < minTTFT {
							minTTFT = result.TimeToFirstToken
						}
						if streamingCount == 1 || result.TimeToFirstToken > maxTTFT {
							maxTTFT = result.TimeToFirstToken
						}
					}
					
					// Track throughput metrics
					if result.TokenThroughput > 0 {
						totalThroughput += result.TokenThroughput
						
						if streamingCount == 1 || result.TokenThroughput < minThroughput {
							minThroughput = result.TokenThroughput
						}
						if streamingCount == 1 || result.TokenThroughput > maxThroughput {
							maxThroughput = result.TokenThroughput
						}
					}
				} else {
					totalTokens += result.TokensUsed
				}
			}
			
			totalResponseTime += result.ResponseTime
			
			if i == 0 || result.ResponseTime < minTime {
				minTime = result.ResponseTime
			}
			if i == 0 || result.ResponseTime > maxTime {
				maxTime = result.ResponseTime
			}
		}
		
		summary.SuccessfulReqs = successCount
		summary.FailedRequests = summary.TotalRequests - successCount
		summary.TotalTokens = totalTokens
		
		if summary.TotalRequests > 0 {
			summary.AvgResponseTime = totalResponseTime / time.Duration(summary.TotalRequests)
			summary.ErrorRate = float64(summary.FailedRequests) / float64(summary.TotalRequests) * 100
		}
		
		summary.MinResponseTime = minTime
		summary.MaxResponseTime = maxTime
		
		// Set streaming metrics if applicable
		if isStreaming {
			summary.IsStreaming = true
			
			if streamingCount > 0 {
				summary.AvgTimeToFirstToken = totalTTFT / time.Duration(streamingCount)
				summary.MinTimeToFirstToken = minTTFT
				summary.MaxTimeToFirstToken = maxTTFT
				
				summary.AvgTokenThroughput = totalThroughput / float64(streamingCount)
				summary.MinTokenThroughput = minThroughput
				summary.MaxTokenThroughput = maxThroughput
			}
		}
		
		summaries[providerName] = summary
	}
	
	return summaries
}

// GetProviders returns the configured providers
func (bs *BenchmarkService) GetProviders() []models.Provider {
	return bs.providers
}
