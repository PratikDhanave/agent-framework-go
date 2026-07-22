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

// This sample demonstrates Concurrent Orchestration with a researcher and a coder.
// Multiple agents work in parallel to solve a task.

const taskPrompt = `I am preparing a report on the energy efficiency of different machine learning model architectures. 
Compare the estimated training and inference energy consumption of ResNet-50, BERT-base, and GPT-2 
on standard datasets (e.g., ImageNet for ResNet, GLUE for BERT, WebText for GPT-2). 
Then, estimate the CO2 emissions associated with each, assuming training on an Azure Standard_NC6s_v3 
VM for 24 hours. Provide tables for clarity, and recommend the most energy-efficient model 
per task type (image classification, text classification, and text generation).`

func main() {
	endpoint := os.Getenv("FOUNDRY_PROJECT_ENDPOINT")
	model := os.Getenv("FOUNDRY_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	researcherAgent := foundryprovider.NewAgent(endpoint, nil, foundryprovider.ModelDeployment(model),
		foundryprovider.AgentConfig{
			Instructions: "You are a researcher. Find relevant information about machine learning model energy efficiency.",
		},
	)

	coderAgent := foundryprovider.NewAgent(endpoint, nil, foundryprovider.ModelDeployment(model),
		foundryprovider.AgentConfig{
			Instructions: "You solve quantitative questions by writing and running code. Show the analysis clearly.",
		},
	)

	reporterAgent := foundryprovider.NewAgent(endpoint, nil, foundryprovider.ModelDeployment(model),
		foundryprovider.AgentConfig{
			Instructions: "You summarize research findings into a clear, concise report with tables and recommendations.",
		},
	)

	// Build concurrent workflow
	wf, err := agentworkflow.NewConcurrentWorkflowBuilder(researcherAgent, coderAgent, reporterAgent).
		Build()
	if err != nil {
		panic(err)
	}

	fmt.Println("Concurrent Orchestration Workflow:")
	fmt.Println("Researcher + Coder + Reporter working in parallel")
	fmt.Printf("\nTask: %s\n", taskPrompt)
	fmt.Println("\nStarting workflow execution...")

	ctx := context.Background()
	run, err := inproc.Default.RunStreaming(ctx, wf, taskPrompt)
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
			fmt.Printf("\n\nFinal Output: %v\n", e.Output)
		case workflow.ErrorEvent:
			fmt.Printf("ERROR: %v\n", e.Error)
		}
	}
}
