// Copyright (c) Microsoft. All rights reserved.

package mcptool

import (
	"testing"

	"github.com/microsoft/agent-framework-go/message"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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
		// A typed-nil pointer marshals to "null" via the JSON fallback
		// (encoding/json emits "null" for a nil pointer before invoking its
		// MarshalJSON), so the conversion is a TextContent{Text:"null"}.
		tc, ok := got.(*mcp.TextContent)
		if !ok {
			t.Errorf("agentContentToMCPContent(%T) = %T, want *mcp.TextContent", c, got)
			continue
		}
		if tc.Text != "null" {
			t.Errorf("agentContentToMCPContent(%T) text = %q, want %q", c, tc.Text, "null")
		}
	}
}

func TestAgentResultToMCPCallToolResult_TypedNilContent(t *testing.T) {
	// Single typed-nil content.
	res := agentResultToMCPCallToolResult((*message.ErrorContent)(nil))
	if res == nil || len(res.Content) != 1 {
		t.Fatalf("single typed-nil content produced unexpected result: %+v", res)
	}
	if tc, ok := res.Content[0].(*mcp.TextContent); !ok || tc.Text != "null" {
		t.Errorf("single typed-nil content = %+v, want TextContent{null}", res.Content[0])
	}

	// Slice containing a typed-nil element.
	res = agentResultToMCPCallToolResult([]message.Content{(*message.ErrorContent)(nil)})
	if res == nil || len(res.Content) != 1 {
		t.Fatalf("slice with typed-nil content produced unexpected result: %+v", res)
	}
	if tc, ok := res.Content[0].(*mcp.TextContent); !ok || tc.Text != "null" {
		t.Errorf("slice typed-nil content = %+v, want TextContent{null}", res.Content[0])
	}
}
