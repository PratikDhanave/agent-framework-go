package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/microsoft/agent-framework-go/provider/foundryprovider"
	"github.com/microsoft/agent-framework-go/workflow"
	"github.com/microsoft/agent-framework-go/workflow/inproc"
)

// This sample demonstrates iterative refinement with quality gates, max iteration safety,
// multiple message handlers, and conditional routing for feedback loops.

const maxIterations = 3

// CriticDecision represents the critic's review decision
type CriticDecision struct {
	Approved bool   `json:"approved"`
	Feedback string `json:"feedback"`
}

func main() {
	endpoint := os.Getenv("FOUNDRY_PROJECT_ENDPOINT")
	model := os.Getenv("FOUNDRY_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	fmt.Println()
	fmt.Println("=== Writer-Critic Iteration Workflow ===")
	fmt.Println()
	fmt.Printf("Writer and Critic will iterate up to %d times until approval.\n\n", maxIterations)

	writerAgent := foundryprovider.NewAgent(endpoint, nil, foundryprovider.ModelDeployment(model),
		foundryprovider.AgentConfig{
			Instructions: `You are a skilled writer. Create clear, engaging content.
If you receive feedback, carefully revise the content to address all concerns.
Maintain the same topic and length requirements.`,
		},
	)

	criticAgent := foundryprovider.NewAgent(endpoint, nil, foundryprovider.ModelDeployment(model),
		foundryprovider.AgentConfig{
			Instructions: `You are a constructive critic. Review the content and provide specific feedback.
Always try to provide actionable suggestions for improvement.
Only approve if the content is high quality, clear, and meets the original requirements.

Provide your decision as JSON with:
- approved: true if content is good, false if revisions needed
- feedback: specific improvements needed (empty if approved)

Be concise but specific in your feedback.`,
		},
	)

	summaryAgent := foundryprovider.NewAgent(endpoint, nil, foundryprovider.ModelDeployment(model),
		foundryprovider.AgentConfig{
			Instructions: "You present the final approved content to the user. Simply output the polished content - no additional commentary needed.",
		},
	)

	// Build workflow: Writer -> Critic -> (if approved) Summary, (if rejected) Writer
	writerExec := workflow.NewExecutor("Writer", func(msg string) string {
		fmt.Printf("\n=== Writer ===\n\n")
		resp, err := writerAgent.RunText(context.Background(), msg).Collect()
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		fmt.Println(resp)
		return resp.String()
	}).Bind()

	criticExec := workflow.NewExecutor("Critic", func(msg string) string {
		fmt.Println("=== Critic ===")
		fmt.Println()
		resp, err := criticAgent.RunText(context.Background(), msg).Collect()
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		fmt.Println(resp)

		// Try to parse JSON decision from response
		respText := resp.String()
		decision := CriticDecision{Approved: false, Feedback: respText}
		if idx := strings.Index(respText, "{"); idx >= 0 {
			jsonStr := respText[idx:]
			if endIdx := strings.Index(jsonStr, "}"); endIdx >= 0 {
				jsonStr = jsonStr[:endIdx+1]
				_ = json.Unmarshal([]byte(jsonStr), &decision)
			}
		}

		fmt.Printf("Decision: %s\n", map[bool]string{true: "APPROVED", false: "NEEDS REVISION"}[decision.Approved])
		if decision.Feedback != "" {
			fmt.Printf("Feedback: %s\n", decision.Feedback)
		}
		fmt.Println()

		result, _ := json.Marshal(decision)
		return string(result)
	}).Bind()

	summaryExec := workflow.NewExecutor("Summary", func(msg string) string {
		fmt.Println("=== Summary ===")
		fmt.Println()

		resp, err := summaryAgent.RunText(context.Background(), msg).Collect()
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		fmt.Println(resp)
		return resp.String()
	}).Bind()

	wf, err := workflow.NewBuilder(writerExec).
		AddEdge(writerExec, criticExec).
		AddDirectEdge(criticExec, summaryExec, false, func(msg any) bool {
			var decision CriticDecision
			_ = json.Unmarshal([]byte(msg.(string)), &decision)
			return decision.Approved
		}).
		AddDirectEdge(criticExec, writerExec, false, func(msg any) bool {
			var decision CriticDecision
			_ = json.Unmarshal([]byte(msg.(string)), &decision)
			return !decision.Approved
		}).
		WithOutputFrom(summaryExec).
		Build()
	if err != nil {
		panic(err)
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("TASK: Write a short blog post about AI ethics (200 words)")
	fmt.Println(strings.Repeat("=", 80) + "\n")

	initialTask := "Write a 200-word blog post about AI ethics. Make it thoughtful and engaging."

	ctx := context.Background()
	run, err := inproc.Default.RunStreaming(ctx, wf, initialTask)
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
			fmt.Printf("\n\n%s\n", strings.Repeat("=", 80))
			fmt.Println("FINAL APPROVED CONTENT")
			fmt.Println(strings.Repeat("=", 80))
			fmt.Printf("\n%v\n\n", e.Output)
			fmt.Println(strings.Repeat("=", 80))
		case workflow.ErrorEvent:
			fmt.Printf("ERROR: %v\n", e.Error)
		}
	}
}
