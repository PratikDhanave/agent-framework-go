package main

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/microsoft/agent-framework-go/provider/foundryprovider"
	"github.com/microsoft/agent-framework-go/workflow"
	"github.com/microsoft/agent-framework-go/workflow/inproc"
)

// This sample demonstrates common agentic workflow patterns:
// - Sequential: agents run one after another
// - Concurrent: agents run in parallel

func main() {
	endpoint := os.Getenv("FOUNDRY_PROJECT_ENDPOINT")
	model := os.Getenv("FOUNDRY_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	fmt.Print("Choose workflow type ('sequential', 'concurrent'): ")

	var choice string
	fmt.Scanln(&choice)

	frenchAgent := getTranslationAgent("French", endpoint, model)
	spanishAgent := getTranslationAgent("Spanish", endpoint, model)
	englishAgent := getTranslationAgent("English", endpoint, model)

	switch choice {
	case "sequential":
		runSequential(frenchAgent, spanishAgent, englishAgent)
	case "concurrent":
		runConcurrent(frenchAgent, spanishAgent, englishAgent)
	default:
		fmt.Println("Invalid choice. Running sequential by default.")
		runSequential(frenchAgent, spanishAgent, englishAgent)
	}
}

func runSequential(agents ...workflow.ExecutorBinding) {
	wf, err := workflow.NewBuilder(agents[0]).
		AddEdge(agents[0], agents[1]).
		AddEdge(agents[1], agents[2]).
		WithOutputFrom(agents[2]).
		Build()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	run, err := inproc.Default.RunStreaming(ctx, wf, "Hello, world!")
	if err != nil {
		panic(err)
	}
	defer func() { _ = run.Close(ctx) }()

	printStream(ctx, run)
}

func runConcurrent(agents ...workflow.ExecutorBinding) {
	wf, err := workflow.NewBuilder(agents[0]).
		AddFanOutEdge(agents[0], agents[1:]).
		AddFanInBarrierEdge(agents[1:], agents[0]).
		WithOutputFrom(agents[0]).
		Build()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	run, err := inproc.Default.RunStreaming(ctx, wf, "Hello, world!")
	if err != nil {
		panic(err)
	}
	defer func() { _ = run.Close(ctx) }()

	printStream(ctx, run)
}

func printStream(ctx context.Context, run *inproc.StreamingRun) {
	for evt, err := range run.WatchStream(ctx) {
		if err != nil {
			panic(err)
		}
		switch e := evt.(type) {
		case workflow.OutputEvent:
			fmt.Printf("\nOutput: %v\n", e.Output)
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
			Instructions: fmt.Sprintf("You are a translation assistant who only responds in %s. Respond to any input by outputting the name of the input language and then translating the input to %s.", targetLanguage, targetLanguage),
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
