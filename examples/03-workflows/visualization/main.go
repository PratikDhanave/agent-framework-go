package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/microsoft/agent-framework-go/workflow"
	"github.com/microsoft/agent-framework-go/workflow/inproc"
)

// This sample demonstrates workflow visualization by showing the workflow structure.
// The workflow processes text through uppercase, reverse, and append steps.

func main() {
	step1 := workflow.NewExecutor("Step1_Uppercase", func(input string) string {
		return strings.ToUpper(input)
	}).Bind()

	step2 := workflow.NewExecutor("Step2_Reverse", func(input string) string {
		runes := []rune(input)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes)
	}).Bind()

	step3 := workflow.NewExecutor("Step3_Append", func(input string) string {
		return input + "_DONE"
	}).Bind()

	wf, err := workflow.NewBuilder(step1).
		AddEdge(step1, step2).
		AddEdge(step2, step3).
		WithOutputFrom(step3).
		Build()
	if err != nil {
		panic(err)
	}

	// Show workflow structure
	fmt.Println("Workflow Structure:")
	fmt.Println("===================")
	fmt.Printf("Name: %s\n", wf.Name())
	fmt.Printf("Start Executor: %s\n", wf.StartExecutorID())
	fmt.Printf("Output Executors: %v\n", wf.OutputExecutorIDs())

	fmt.Println("\nEdges:")
	for sourceID, edges := range wf.Edges() {
		for _, edge := range edges {
			for _, sinkID := range edge.Connection.SinkIDs {
				fmt.Printf("  %s -> %s\n", sourceID, sinkID)
			}
		}
	}

	fmt.Println("\nExecutors:")
	for id := range wf.ReflectExecutors() {
		fmt.Printf("  %s\n", id)
	}

	// Execute the workflow to verify it works
	ctx := context.Background()
	run, err := inproc.Default.Run(ctx, wf, "Hello")
	if err != nil {
		panic(err)
	}

	for evt := range run.NewEvents() {
		if output, ok := evt.(workflow.OutputEvent); ok {
			fmt.Printf("\nWorkflow Output: %v\n", output.Output)
		}
	}
}
