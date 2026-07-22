package main

import (
	"context"
	"fmt"
	"os"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/provider/foundryprovider"
	"github.com/microsoft/agent-framework-go/tool"
	"github.com/microsoft/agent-framework-go/tool/agenttool"
)

// This sample demonstrates an agent with RAG (Retrieval-Augmented Generation) capabilities.
// It shows how to use an agent as a tool for knowledge retrieval.

const instructions = `You are a helpful assistant that can search for information.
When the user asks about a topic, use the knowledge_search tool to find relevant information.
Always cite your sources when providing answers.`

func main() {
	endpoint := os.Getenv("FOUNDRY_PROJECT_ENDPOINT")
	model := os.Getenv("FOUNDRY_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	// Create a knowledge search agent
	knowledgeAgent := foundryprovider.NewAgent(endpoint, nil, foundryprovider.ModelDeployment(model),
		foundryprovider.AgentConfig{
			Instructions: `You are a knowledge retrieval agent. When given a query, provide detailed, 
accurate information about the topic. Be specific and include facts, examples, and context.`,
			Config: agent.Config{
				Name:        "KnowledgeSearch",
				Description: "Search for information on a topic. Returns detailed knowledge about the query.",
			},
		},
	)

	// Wrap knowledge agent as a tool
	knowledgeTool := agenttool.New(knowledgeAgent, agenttool.Config{})

	// Create main agent with the knowledge tool
	mainAgent := foundryprovider.NewAgent(endpoint, nil, foundryprovider.ModelDeployment(model),
		foundryprovider.AgentConfig{
			Instructions: instructions,
			Config: agent.Config{
				Tools: []tool.Tool{knowledgeTool},
			},
		},
	)

	ctx := context.Background()

	// Ask questions that require knowledge retrieval
	questions := []string{
		"What is Go's garbage collector and how does it work?",
		"What are Go channels and when should I use them?",
		"Explain the difference between sync.Mutex and sync.RWMutex",
	}

	for _, q := range questions {
		fmt.Printf("\n>> %s\n\n", q)
		resp, err := mainAgent.RunText(ctx, q).Collect()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}
		fmt.Printf("Agent: %s\n", resp.String())
	}
}
