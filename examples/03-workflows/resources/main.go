package main

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync"

	"github.com/microsoft/agent-framework-go/workflow"
	"github.com/microsoft/agent-framework-go/workflow/inproc"
)

// This sample demonstrates executor lifecycle management with resource initialization and cleanup.
// It shows how to properly manage resources using Initialize, Close, and Reset functions.

func main() {
	// Create executors with lifecycle management
	dbConn := newDatabaseConnection("DBConnection")
	apiClient := newAPIClient("APIClient")

	wf, err := workflow.NewBuilder(dbConn).
		AddEdge(dbConn, apiClient).
		WithOutputFrom(apiClient).
		Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build workflow: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	fmt.Println("=== Executor Lifecycle Demo ===")
	fmt.Println()

	// Run 1
	fmt.Println(">> Run 1:")
	run1, err := inproc.Default.RunStreaming(ctx, wf, "Fetch user data for ID 123")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start run: %v\n", err)
		os.Exit(1)
	}
	for evt, err := range run1.WatchStream(ctx) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}
		switch e := evt.(type) {
		case workflow.OutputEvent:
			fmt.Printf("Output: %v\n", e.Output)
		}
	}

	fmt.Println()

	// Run 2 (shared executor will be reset)
	fmt.Println(">> Run 2:")
	run2, err := inproc.Default.RunStreaming(ctx, wf, "Fetch user data for ID 456")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start run: %v\n", err)
		os.Exit(1)
	}
	for evt, err := range run2.WatchStream(ctx) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}
		switch e := evt.(type) {
		case workflow.OutputEvent:
			fmt.Printf("Output: %v\n", e.Output)
		}
	}
}

type resourceTracker struct {
	mu       sync.Mutex
	opened   bool
	queryCnt int
}

func newDatabaseConnection(id string) workflow.ExecutorBinding {
	tracker := &resourceTracker{}

	return workflow.BindNewExecutorFunc(id, func(_ string, executorID string) (*workflow.Executor, error) {
		return &workflow.Executor{
			ID: executorID,
			ConfigureProtocol: func(rb *workflow.ProtocolBuilder) (*workflow.ProtocolBuilder, error) {
				rb.RouteBuilder.
					AddHandlerRaw(reflect.TypeFor[string](), reflect.TypeFor[string](), func(ctx *workflow.Context, msg any) (any, error) {
						tracker.mu.Lock()
						defer tracker.mu.Unlock()
						tracker.queryCnt++
						result := fmt.Sprintf("[DB] Query #%d executed for: %s", tracker.queryCnt, msg.(string))
						return result, nil
					})
				return rb, nil
			},
			InitializeFunc: func(_ *workflow.Context) error {
				tracker.mu.Lock()
				defer tracker.mu.Unlock()
				tracker.opened = true
				fmt.Printf("  [%s] Connection opened\n", id)
				return nil
			},
			CloseFunc: func(_ context.Context) error {
				tracker.mu.Lock()
				defer tracker.mu.Unlock()
				tracker.opened = false
				fmt.Printf("  [%s] Connection closed (total queries: %d)\n", id, tracker.queryCnt)
				return nil
			},
			ResetFunc: func() error {
				tracker.mu.Lock()
				defer tracker.mu.Unlock()
				tracker.queryCnt = 0
				fmt.Printf("  [%s] State reset\n", id)
				return nil
			},
		}, nil
	})
}

func newAPIClient(id string) workflow.ExecutorBinding {
	tracker := &resourceTracker{}

	return workflow.BindNewExecutorFunc(id, func(_ string, executorID string) (*workflow.Executor, error) {
		return &workflow.Executor{
			ID: executorID,
			ConfigureProtocol: func(rb *workflow.ProtocolBuilder) (*workflow.ProtocolBuilder, error) {
				rb.RouteBuilder.
					AddHandlerRaw(reflect.TypeFor[string](), reflect.TypeFor[string](), func(ctx *workflow.Context, msg any) (any, error) {
						tracker.mu.Lock()
						defer tracker.mu.Unlock()
						tracker.queryCnt++
						result := fmt.Sprintf("[API] Request #%d processed: %s", tracker.queryCnt, msg.(string))
						return result, nil
					})
				return rb, nil
			},
			InitializeFunc: func(_ *workflow.Context) error {
				tracker.mu.Lock()
				defer tracker.mu.Unlock()
				tracker.opened = true
				fmt.Printf("  [%s] Client initialized\n", id)
				return nil
			},
			CloseFunc: func(_ context.Context) error {
				tracker.mu.Lock()
				defer tracker.mu.Unlock()
				tracker.opened = false
				fmt.Printf("  [%s] Client closed (total requests: %d)\n", id, tracker.queryCnt)
				return nil
			},
			ResetFunc: func() error {
				tracker.mu.Lock()
				defer tracker.mu.Unlock()
				tracker.queryCnt = 0
				fmt.Printf("  [%s] State reset\n", id)
				return nil
			},
		}, nil
	})
}
