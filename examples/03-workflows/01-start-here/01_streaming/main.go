package main

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/microsoft/agent-framework-go/workflow"
	"github.com/microsoft/agent-framework-go/workflow/inproc"
)

// This sample introduces streaming output in workflows.
//
// While a basic workflow waits for the entire workflow to complete before showing results,
// this example streams events back to you in real-time as each executor finishes processing.
// This is useful for monitoring long-running workflows or providing live feedback to users.
//
// The workflow logic is identical: uppercase text, then reverse it. The difference is in
// how we observe the execution - we see intermediate results as they happen.

func main() {
	uppercase := workflow.NewExecutor("UppercaseExecutor", func(input string) string {
		return strings.ToUpper(input)
	}).Bind()

	reverse := workflow.NewExecutor("ReverseTextExecutor", func(input string) string {
		runes := []rune(input)
		slices.Reverse(runes)
		return string(runes)
	}).Bind()

	wf, err := workflow.NewBuilder(uppercase).
		AddEdge(uppercase, reverse).
		WithOutputFrom(reverse).
		Build()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	run, err := inproc.Default.RunStreaming(ctx, wf, "Hello, World!")
	if err != nil {
		panic(err)
	}
	defer func() { _ = run.Close(ctx) }()

	for evt, err := range run.WatchStream(ctx) {
		if err != nil {
			panic(err)
		}
		switch e := evt.(type) {
		case workflow.ExecutorCompletedEvent:
			fmt.Printf("%s: %v\n", e.ExecutorID, e.Result)
		case workflow.ErrorEvent:
			fmt.Printf("ERROR: %v\n", e.Error)
		case workflow.ExecutorFailedEvent:
			fmt.Printf("Executor '%s' failed: %v\n", e.ExecutorID, e.Error)
		}
	}
}
