// Copyright (c) Microsoft. All rights reserved.

package mcptool

import (
	"testing"

	"github.com/microsoft/agent-framework-go/message"
)

// A tool may return a typed-nil message.Content (e.g. a (*message.ErrorContent)(nil)
// from a `var ec *message.ErrorContent; return ec, nil`). Such a value satisfies
// the message.Content interface with a non-nil dynamic type, so the conversion
// must not dereference it and panic.
func TestAgentContentToMCPContent_TypedNilDoesNotPanic(t *testing.T) {
	cases := []message.Content{
		(*message.TextContent)(nil),
		(*message.ErrorContent)(nil),
		(*message.DataContent)(nil),
		(*message.URIContent)(nil),
	}
	for _, c := range cases {
		got := agentContentToMCPContent(c) // must not panic
		if got == nil {
			t.Errorf("agentContentToMCPContent(%T) returned nil", c)
		}
	}
}

func TestAgentResultToMCPCallToolResult_TypedNilContent(t *testing.T) {
	// Single typed-nil content.
	if res := agentResultToMCPCallToolResult((*message.ErrorContent)(nil)); res == nil {
		t.Fatal("single typed-nil content produced a nil result")
	}
	// Slice containing a typed-nil element.
	res := agentResultToMCPCallToolResult([]message.Content{(*message.ErrorContent)(nil)})
	if res == nil || len(res.Content) != 1 {
		t.Fatalf("slice with typed-nil content produced unexpected result: %+v", res)
	}
}
