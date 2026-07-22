package main

import (
	"context"
	"fmt"
	"os"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/provider/foundryprovider"
	"github.com/microsoft/agent-framework-go/tool"
	"github.com/microsoft/agent-framework-go/tool/shelltool"
)

// This sample demonstrates an agent with code execution capabilities using the shell tool.
// The agent can execute real shell commands to answer questions.

const instructions = `You are a helpful assistant that can execute shell commands to answer questions.
When the user asks you to run code or execute commands, use the run_shell tool.
Always show the command you're running and its output.`

func main() {
	endpoint := os.Getenv("FOUNDRY_PROJECT_ENDPOINT")
	model := os.Getenv("FOUNDRY_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	// Create shell tool in stateless mode (fresh shell per call)
	shell, err := shelltool.NewLocal(shelltool.LocalConfig{
		Mode:              shelltool.ModeStateless,
		AcknowledgeUnsafe: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create shell: %v\n", err)
		os.Exit(1)
	}
	defer shell.Close()

	// Create environment provider to detect available tools
	envProvider := shelltool.NewEnvironmentProvider(shell, shelltool.EnvironmentProviderConfig{})

	// Create agent with shell tool
	a := foundryprovider.NewAgent(endpoint, nil, foundryprovider.ModelDeployment(model),
		foundryprovider.AgentConfig{
			Instructions: instructions,
			Config: agent.Config{
				Tools:            []tool.Tool{shell},
				ContextProviders: []agent.ContextProvider{envProvider},
			},
		},
	)

	ctx := context.Background()

	// Multi-turn conversation with code execution
	prompts := []string{
		"What's the current working directory?",
		"List all Go files in the current directory",
		"What's the Go version installed on this system?",
	}

	for _, prompt := range prompts {
		fmt.Printf("\n>> %s\n\n", prompt)
		resp, err := a.RunText(ctx, prompt).Collect()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}
		fmt.Printf("Agent: %s\n", resp.String())
	}
}
