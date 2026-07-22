package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/agent/harness/loop"
	"github.com/microsoft/agent-framework-go/provider/foundryprovider"
)

// This sample demonstrates an agent harness with an evaluation loop.
// The agent iterates on a task until a completion marker is found in the response.

const (
	completionMarker = "[TASK_COMPLETE]"

	agentInstructions = `You are a task completion agent. Work through the given task step by step.
When you have fully completed the task, end your response with the marker ` + completionMarker
)

func main() {
	endpoint := os.Getenv("FOUNDRY_PROJECT_ENDPOINT")
	model := os.Getenv("FOUNDRY_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	// Create loop middleware with completion marker evaluator
	loopMiddleware := loop.New(loop.Config{
		Evaluators: []loop.Evaluator{
			loop.NewCompletionMarkerEvaluator(loop.CompletionMarkerConfig{
				Marker: completionMarker,
			}),
		},
		MaxIterations: 3,
	})

	// Create agent with loop middleware
	a := foundryprovider.NewAgent(endpoint, nil, foundryprovider.ModelDeployment(model),
		foundryprovider.AgentConfig{
			Instructions: agentInstructions,
			Config: agent.Config{
				Middlewares: []agent.Middleware{loopMiddleware},
			},
		},
	)

	ctx := context.Background()

	tasks := []string{
		"Write a haiku about programming",
		"Create a simple Go function that adds two numbers",
	}

	for _, task := range tasks {
		fmt.Printf("\n>> Task: %s\n\n", task)
		resp, err := a.RunText(ctx, task).Collect()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}
		// Remove completion marker from output
		output := strings.ReplaceAll(resp.String(), completionMarker, "")
		output = strings.TrimSpace(output)
		fmt.Printf("Agent: %s\n", output)
	}
}
