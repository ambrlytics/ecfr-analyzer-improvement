package concurrent

import (
	"fmt"
	"github.com/gofiber/fiber/v2/log"
	"sync"
)

// WorkerFunc defines the function signature for work to be executed
// It receives the item to process and channels for communication
type WorkerFunc[T any, R any] func(item T, messages chan<- string, results chan<- R, errors chan<- error)

// RunnerConfig configures the concurrent runner
type RunnerConfig struct {
	MaxConcurrency int    // 0 means unlimited concurrency
	LogPrefix      string // Prefix for log messages
}

// Runner encapsulates concurrent processing with channels and wait groups
type Runner[T any, R any] struct {
	config RunnerConfig
}

// NewRunner creates a new concurrent runner with the given configuration
func NewRunner[T any, R any](config RunnerConfig) *Runner[T, R] {
	if config.LogPrefix == "" {
		config.LogPrefix = "Runner"
	}
	return &Runner[T, R]{
		config: config,
	}
}

// RunResult contains the results of a concurrent run
type RunResult[R any] struct {
	Results []R
	Errors  []error
}

// Run executes the worker function for each item concurrently
// Returns aggregated results and errors
func (r *Runner[T, R]) Run(items []T, worker WorkerFunc[T, R]) RunResult[R] {
	if len(items) == 0 {
		return RunResult[R]{
			Results: []R{},
			Errors:  []error{},
		}
	}

	var messagesWG sync.WaitGroup

	// Messages channel for logging
	messages := make(chan string)
	messagesWG.Add(1)
	go func() {
		defer messagesWG.Done()
		for message := range messages {
			r.logInfo(message)
		}
	}()

	// Results channel for successful completions
	results := make(chan R)
	var resultsList []R
	messagesWG.Add(1)
	go func() {
		defer messagesWG.Done()
		for result := range results {
			resultsList = append(resultsList, result)
		}
	}()

	// Errors channel for failures
	errors := make(chan error)
	var errorsList []error
	messagesWG.Add(1)
	go func() {
		defer messagesWG.Done()
		for err := range errors {
			errorsList = append(errorsList, err)
		}
	}()

	// Worker wait group
	var workersWg sync.WaitGroup

	// Throttle channel for limiting concurrency (if configured)
	var throttle chan int
	if r.config.MaxConcurrency > 0 {
		throttle = make(chan int, r.config.MaxConcurrency)
	}

	// Process each item
	for _, item := range items {
		workersWg.Add(1)

		// Acquire throttle slot if configured
		if throttle != nil {
			throttle <- 1
		}

		go func(item T) {
			defer workersWg.Done()

			// Release throttle slot if configured
			if throttle != nil {
				defer func() { <-throttle }()
			}

			// Execute worker function
			worker(item, messages, results, errors)
		}(item)
	}

	// Wait for all workers to complete
	workersWg.Wait()

	// Close channels
	close(messages)
	close(results)
	close(errors)

	// Wait for all message handlers to complete
	messagesWG.Wait()

	return RunResult[R]{
		Results: resultsList,
		Errors:  errorsList,
	}
}

// RunWithContext is similar to Run but provides a way to access results as they come
// Useful when you need more control over result handling
func (r *Runner[T, R]) RunWithCallbacks(
	items []T,
	worker WorkerFunc[T, R],
	onMessage func(string),
	onResult func(R),
	onError func(error),
) {
	if len(items) == 0 {
		return
	}

	var messagesWG sync.WaitGroup

	// Messages channel for logging
	messages := make(chan string)
	messagesWG.Add(1)
	go func() {
		defer messagesWG.Done()
		for message := range messages {
			if onMessage != nil {
				onMessage(message)
			}
			r.logInfo(message)
		}
	}()

	// Results channel for successful completions
	results := make(chan R)
	messagesWG.Add(1)
	go func() {
		defer messagesWG.Done()
		for result := range results {
			if onResult != nil {
				onResult(result)
			}
		}
	}()

	// Errors channel for failures
	errors := make(chan error)
	messagesWG.Add(1)
	go func() {
		defer messagesWG.Done()
		for err := range errors {
			if onError != nil {
				onError(err)
			}
		}
	}()

	// Worker wait group
	var workersWg sync.WaitGroup

	// Throttle channel for limiting concurrency (if configured)
	var throttle chan int
	if r.config.MaxConcurrency > 0 {
		throttle = make(chan int, r.config.MaxConcurrency)
	}

	// Process each item
	for _, item := range items {
		workersWg.Add(1)

		// Acquire throttle slot if configured
		if throttle != nil {
			throttle <- 1
		}

		go func(item T) {
			defer workersWg.Done()

			// Release throttle slot if configured
			if throttle != nil {
				defer func() { <-throttle }()
			}

			// Execute worker function
			worker(item, messages, results, errors)
		}(item)
	}

	// Wait for all workers to complete
	workersWg.Wait()

	// Close channels
	close(messages)
	close(results)
	close(errors)

	// Wait for all message handlers to complete
	messagesWG.Wait()
}

func (r *Runner[T, R]) logInfo(message string) {
	log.Info(fmt.Sprintf("%s: %s", r.config.LogPrefix, message))
}
