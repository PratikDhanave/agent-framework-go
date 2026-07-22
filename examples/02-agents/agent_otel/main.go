package main

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/provider/foundryprovider"
	"github.com/microsoft/agent-framework-go/provider/otelprovider"
)

// This sample demonstrates OpenTelemetry integration with the Agent Framework.
// It sets up tracing and creates an agent with OTel middleware for observability.

func main() {
	endpoint := os.Getenv("FOUNDRY_PROJECT_ENDPOINT")
	model := os.Getenv("FOUNDRY_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	// Initialize OpenTelemetry
	tp := initTracer()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	// Create OTel middleware
	otelMiddleware := otelprovider.NewMiddleware(otelprovider.MiddlewareConfig{})

	// Create agent with OTel middleware
	a := foundryprovider.NewAgent(endpoint, nil, foundryprovider.ModelDeployment(model),
		foundryprovider.AgentConfig{
			Instructions: "You are a helpful assistant.",
			Config: agent.Config{
				Middlewares: []agent.Middleware{otelMiddleware},
			},
		},
	)

	ctx := context.Background()

	// Run the agent
	fmt.Println("Running agent with OpenTelemetry tracing...")
	result, err := a.RunText(ctx, "Hello! What can you help me with?").Collect()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println("Response:", result)

	// Another invocation
	result2, err := a.RunText(ctx, "Tell me a joke").Collect()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println("Response:", result2)

	fmt.Println("\nCheck your OTel collector or console output for trace details.")
}

func initTracer() *sdktrace.TracerProvider {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		panic(err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)

	return tp
}
