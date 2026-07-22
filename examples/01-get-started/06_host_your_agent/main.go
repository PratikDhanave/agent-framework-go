package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/provider/foundryprovider"
	"github.com/microsoft/agent-framework-go/tool"
	"github.com/microsoft/agent-framework-go/tool/functool"
)

// This sample demonstrates hosting an agent via HTTP server.
// The agent uses function tools and exposes an HTTP endpoint for interaction.

func main() {
	endpoint := os.Getenv("FOUNDRY_PROJECT_ENDPOINT")
	model := os.Getenv("FOUNDRY_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	// Define a weather tool
	type WeatherArgs struct {
		Location string `json:"location"`
	}

	weatherTool := functool.MustNew(functool.Config{
		Name:        "get_weather",
		Description: "Get the current weather for a location",
	}, func(_ context.Context, args WeatherArgs) (string, error) {
		return fmt.Sprintf("The weather in %s is sunny and 72°F", args.Location), nil
	})

	// Create agent with tools
	a := foundryprovider.NewAgent(endpoint, nil, foundryprovider.ModelDeployment(model),
		foundryprovider.AgentConfig{
			Instructions: "You are a helpful assistant with access to weather information.",
			Config: agent.Config{
				Tools: []tool.Tool{weatherTool},
			},
		},
	)

	// HTTP handler
	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		query := r.URL.Query().Get("q")
		if query == "" {
			http.Error(w, "Missing 'q' parameter", http.StatusBadRequest)
			return
		}

		ctx := context.Background()
		result, err := a.RunText(ctx, query).Collect()
		if err != nil {
			http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, result)
	})

	fmt.Println("Agent server listening on :8080")
	fmt.Println("Try: curl -X POST 'http://localhost:8080/chat?q=What+is+the+weather+in+Seattle'")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
