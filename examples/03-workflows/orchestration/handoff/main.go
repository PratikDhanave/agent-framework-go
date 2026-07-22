package main

import (
	"context"
	"fmt"
	"os"

	"github.com/microsoft/agent-framework-go/provider/foundryprovider"
	"github.com/microsoft/agent-framework-go/workflow"
	"github.com/microsoft/agent-framework-go/workflow/agentworkflow"
	"github.com/microsoft/agent-framework-go/workflow/inproc"
)

// This sample demonstrates the Sequential Orchestration pattern.
// Three translation agents are chained sequentially to translate text.

func main() {
	endpoint := os.Getenv("FOUNDRY_PROJECT_ENDPOINT")
	model := os.Getenv("FOUNDRY_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	frenchAgent := foundryprovider.NewAgent(endpoint, nil, foundryprovider.ModelDeployment(model),
		foundryprovider.AgentConfig{
			Instructions: "You are a translation assistant that translates the provided text to French.",
		},
	)

	spanishAgent := foundryprovider.NewAgent(endpoint, nil, foundryprovider.ModelDeployment(model),
		foundryprovider.AgentConfig{
			Instructions: "You are a translation assistant that translates the provided text to Spanish.",
		},
	)

	englishAgent := foundryprovider.NewAgent(endpoint, nil, foundryprovider.ModelDeployment(model),
		foundryprovider.AgentConfig{
			Instructions: "You are a translation assistant that translates the provided text to English.",
		},
	)

	// Build sequential workflow
	wf, err := agentworkflow.NewSequentialWorkflowBuilder(frenchAgent, spanishAgent, englishAgent).
		Build()
	if err != nil {
		panic(err)
	}

	fmt.Println("Sequential Translation Workflow:")
	fmt.Println("French -> Spanish -> English")
	fmt.Println()

	ctx := context.Background()
	run, err := inproc.Default.RunStreaming(ctx, wf, "Hello World! This is a test of the sequential workflow pattern.")
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
			fmt.Printf("\nFinal Output: %v\n", e.Output)
		case workflow.ErrorEvent:
			fmt.Printf("ERROR: %v\n", e.Error)
		}
	}
}
