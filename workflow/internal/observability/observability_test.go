// Copyright (c) Microsoft. All rights reserved.

package observability_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	observability "github.com/microsoft/agent-framework-go/workflow/internal/observability"
	workflowobservability "github.com/microsoft/agent-framework-go/workflow/observability"
)

func attributeValue(t *testing.T, attrs []workflowobservability.Attribute, key string) string {
	t.Helper()
	for _, attr := range attrs {
		if attr.Key == key {
			value, ok := attr.Value.(string)
			if !ok {
				t.Fatalf("attribute %q value is %T, want string", key, attr.Value)
			}
			return value
		}
	}
	t.Fatalf("attribute %q not found", key)
	return ""
}

func TestErrorAttributesShortTypeName(t *testing.T) {
	attrs := observability.ErrorAttributes(errors.New("boom"))
	if got := attributeValue(t, attrs, observability.TagErrorType); got != "errorString" {
		t.Errorf("error.type = %q, want %q", got, "errorString")
	}

	wrapped := fmt.Errorf("ctx: %w", errors.New("boom"))
	attrs = observability.ErrorAttributes(wrapped)
	if got := attributeValue(t, attrs, observability.TagErrorType); got != "wrapError" {
		t.Errorf("wrapped error.type = %q, want %q", got, "wrapError")
	}
}

func TestBuildErrorAttributesShortTypeName(t *testing.T) {
	attrs := observability.BuildErrorAttributes(errors.New("boom"))
	if got := attributeValue(t, attrs, observability.TagBuildErrorType); got != "errorString" {
		t.Errorf("build.error.type = %q, want %q", got, "errorString")
	}

	wrapped := fmt.Errorf("ctx: %w", errors.New("boom"))
	attrs = observability.BuildErrorAttributes(wrapped)
	if got := attributeValue(t, attrs, observability.TagBuildErrorType); got != "wrapError" {
		t.Errorf("wrapped build.error.type = %q, want %q", got, "wrapError")
	}
}

type fakeSpan struct {
	attrs []workflowobservability.Attribute
}

func (s *fakeSpan) End()                                                {}
func (s *fakeSpan) AddEvent(string, ...workflowobservability.Attribute) {}
func (s *fakeSpan) SetAttributes(attrs ...workflowobservability.Attribute) {
	s.attrs = append(s.attrs, attrs...)
}
func (s *fakeSpan) RecordError(error) {}
func (s *fakeSpan) SetError(string)   {}

type fakeTracer struct {
	span *fakeSpan
}

func (tr *fakeTracer) Start(ctx context.Context, _ string, _ workflowobservability.SpanOptions) (context.Context, workflowobservability.Span) {
	return ctx, tr.span
}
func (tr *fakeTracer) ExtractTraceContext(context.Context) map[string]string { return nil }

func TestCaptureErrorShortTypeName(t *testing.T) {
	span := &fakeSpan{}
	telemetry := observability.New(observability.Options{Tracer: &fakeTracer{span: span}})

	_, activity := telemetry.StartWorkflowRun(context.Background(), observability.WorkflowMetadata{ID: "wf"})
	activity.CaptureError(errors.New("boom"))

	if got := attributeValue(t, span.attrs, observability.TagErrorType); got != "errorString" {
		t.Errorf("captured error.type = %q, want %q", got, "errorString")
	}
}
