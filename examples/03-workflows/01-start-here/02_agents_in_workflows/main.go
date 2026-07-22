package main

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/provider/foundryprovider"
	"github.com/microsoft/agent-framework-go/workflow"
	"github.com/microsoft/agent-framework-go/workflow/inproc"
)

// This sample demonstrates using agents in workflows.
// Three translation agents are chained sequentially:
// French -> Spanish -> English.

func main() {
	endpoint := os.Getenv("FOUNDRY_PROJECT_ENDPOINT")
	model := os.Getenv("FOUNDRY_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	frenchAgent := getTranslationAgent("French", endpoint, model)
	spanishAgent := getTranslationAgent("Spanish", endpoint, model)
	englishAgent := getTranslationAgent("English", endpoint, model)

	wf, err := workflow.NewBuilder(frenchAgent).
		AddEdge(frenchAgent, spanishAgent).
		AddEdge(spanishAgent, englishAgent).
		Build()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	run, err := inproc.Default.RunStreaming(ctx, wf, "Hello World!")
	if err != nil {
		panic(err)
	}
	defer func() { _ = run.Close(ctx) }()

	for evt, err := range run.WatchStream(ctx) {
		if err != nil {
			panic(err)
		}
		switch e := evt.(type) {
		case workflow.OutputEvent:
			fmt.Printf("Workflow Output: %v\n", e.Output)
		case workflow.ErrorEvent:
			fmt.Printf("ERROR: %v\n", e.Error)
		case workflow.ExecutorFailedEvent:
			fmt.Printf("Executor '%s' failed: %v\n", e.ExecutorID, e.Error)
		}
	}
}

func getTranslationAgent(targetLanguage, endpoint, model string) workflow.ExecutorBinding {
	a := foundryprovider.NewAgent(endpoint, nil, foundryprovider.ModelDeployment(model),
		foundryprovider.AgentConfig{
			Instructions: fmt.Sprintf("You are a translation assistant that translates the provided text to %s.", targetLanguage),
			Config: agent.Config{
				Name: targetLanguage,
			},
		},
	)
	return workflow.BindNewExecutorFunc(targetLanguage, func(_ string, executorID string) (*workflow.Executor, error) {
		return &workflow.Executor{
			ID: executorID,
			ConfigureProtocol: func(rb *workflow.ProtocolBuilder) (*workflow.ProtocolBuilder, error) {
				rb.RouteBuilder.
					AddHandlerRaw(reflect.TypeFor[string](), reflect.TypeFor[string](), func(ctx *workflow.Context, msg any) (any, error) {
						return a.RunText(ctx, msg.(string)).Collect()
					})
				return rb, nil
			},
		}, nil
	})
}
