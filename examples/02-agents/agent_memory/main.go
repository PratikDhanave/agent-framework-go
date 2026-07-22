package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/message"
	"github.com/microsoft/agent-framework-go/provider/foundryprovider"
)

// This sample demonstrates an agent with persistent memory.
// The agent remembers user information across turns using a custom ContextProvider.

func main() {
	endpoint := os.Getenv("FOUNDRY_PROJECT_ENDPOINT")
	model := os.Getenv("FOUNDRY_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	a := foundryprovider.NewAgent(endpoint, nil, foundryprovider.ModelDeployment(model),
		foundryprovider.AgentConfig{
			Instructions: "You are a friendly assistant. Remember the user's details.",
			Config: agent.Config{
				Name:             "MemoryAgent",
				ContextProviders: []agent.ContextProvider{newUserMemoryProvider()},
			},
		},
	)

	ctx := context.Background()
	session, err := a.CreateSession(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create session: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Memory Agent Demo ===")
	fmt.Println()

	// First turn - blank memory
	fmt.Println(">> Turn 1: Hello, what is the square root of 9?")
	resp, err := a.RunText(ctx, "Hello, what is the square root of 9?", agent.WithSession(session)).Collect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	fmt.Printf("Agent: %s\n\n", resp.String())

	// Second turn - provide name
	fmt.Println(">> Turn 2: My name is Alice")
	resp, err = a.RunText(ctx, "My name is Alice", agent.WithSession(session)).Collect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	fmt.Printf("Agent: %s\n\n", resp.String())

	// Third turn - provide age
	fmt.Println(">> Turn 3: I am 25 years old")
	resp, err = a.RunText(ctx, "I am 25 years old", agent.WithSession(session)).Collect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	fmt.Printf("Agent: %s\n\n", resp.String())

	// Fourth turn - ask about memory
	fmt.Println(">> Turn 4: What is my name and age?")
	resp, err = a.RunText(ctx, "What is my name and age?", agent.WithSession(session)).Collect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	fmt.Printf("Agent: %s\n\n", resp.String())

	// Show stored memory
	state := getProviderState(session)
	fmt.Println(">> Stored Memory:")
	fmt.Printf("  Name: %s\n", state.UserName)
	fmt.Printf("  Age: %d\n", state.UserAge)
}

func newUserMemoryProvider() agent.ContextProvider {
	return agent.NewContextProvider(agent.ContextProviderConfig{
		SourceID: userMemorySourceID,
		Provide:  provideUserMemory,
		Store:    storeUserMemory,
	})
}

const userMemorySourceID = "user_memory"

type providerState struct {
	UserName string `json:"user_name,omitempty"`
	UserAge  int    `json:"user_age,omitempty"`
}

func getProviderState(session *agent.Session) providerState {
	if session == nil {
		return providerState{}
	}
	var state providerState
	_, _ = session.Get(userMemorySourceID, &state)
	return state
}

func provideUserMemory(_ context.Context, invoking agent.InvokingContext) ([]*message.Message, []agent.Option, error) {
	session, _ := agent.GetOption(invoking.Options, agent.WithSession)
	state := getProviderState(session)
	var instructions strings.Builder
	if strings.TrimSpace(state.UserName) != "" {
		fmt.Fprintf(&instructions, "The user's name is %s.\n", state.UserName)
	} else {
		instructions.WriteString("Ask the user for their name.\n")
	}
	if state.UserAge > 0 {
		fmt.Fprintf(&instructions, "The user's age is %d.\n", state.UserAge)
	} else {
		instructions.WriteString("Ask the user for their age.\n")
	}
	return nil, []agent.Option{agent.WithInstructions(instructions.String())}, nil
}

func storeUserMemory(_ context.Context, invoked agent.InvokedContext) error {
	session, _ := agent.GetOption(invoked.Options, agent.WithSession)
	state := getProviderState(session)
	for _, msg := range invoked.RequestMessages {
		if msg == nil || msg.Role != message.RoleUser {
			continue
		}
		text := strings.TrimSpace(msg.Contents.Text())
		if text == "" {
			continue
		}
		lower := strings.ToLower(text)
		if state.UserName == "" {
			if name, ok := extractName(text, lower); ok {
				state.UserName = name
			}
		}
		if state.UserAge == 0 {
			if age, ok := extractAge(lower); ok {
				state.UserAge = age
			}
		}
	}
	session.Set(userMemorySourceID, state)
	return nil
}

func extractName(text, lower string) (string, bool) {
	idx := strings.Index(lower, "my name is")
	if idx < 0 {
		return "", false
	}
	name := strings.TrimSpace(text[idx+len("my name is"):])
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "", false
	}
	return strings.Trim(parts[0], ".,!?"), true
}

func extractAge(lower string) (int, bool) {
	fields := strings.Fields(lower)
	for i, field := range fields {
		value, err := strconv.Atoi(strings.Trim(field, ".,!?"))
		if err != nil || value <= 0 {
			continue
		}
		if i >= 2 && fields[i-2] == "i" && fields[i-1] == "am" && followedByYear(fields, i) {
			return value, true
		}
		if i >= 1 && fields[i-1] == "i'm" && followedByYear(fields, i) {
			return value, true
		}
		if i >= 3 && fields[i-3] == "my" && fields[i-2] == "age" && fields[i-1] == "is" {
			return value, true
		}
	}
	return 0, false
}

func followedByYear(fields []string, numberIndex int) bool {
	if numberIndex+1 >= len(fields) {
		return false
	}
	next := strings.Trim(fields[numberIndex+1], ".,!? ")
	return next == "year" || next == "years"
}
