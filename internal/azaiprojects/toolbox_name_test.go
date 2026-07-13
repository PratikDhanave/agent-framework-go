// Copyright (c) Microsoft. All rights reserved.

package azaiprojects

import (
	"context"
	"testing"
)

func TestValidateToolboxName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid single-segment identifiers.
		{"simple", "my-toolbox", false},
		{"underscores and digits", "tool_box_1", false},
		{"interior dot", "v1.2.3", false},
		{"allowed specials", "a:b@c~d", false},

		// Empty.
		{"empty", "", true},

		// Raw separators and delimiters.
		{"forward slash", "a/b", true},
		{"leading slash", "/evil", true},
		{"backslash", "a\\b", true},
		{"question mark", "a?b", true},
		{"hash", "a#b", true},

		// Traversal segments.
		{"dot", ".", true},
		{"dotdot", "..", true},

		// Percent-encoded separators / traversal decode back to unsafe forms.
		{"encoded slash lower", "a%2fb", true},
		{"encoded slash upper", "a%2Fb", true},
		{"encoded backslash", "a%5cb", true},
		{"encoded question", "a%3fb", true},
		{"encoded hash", "a%23b", true},
		{"encoded dotdot", "%2e%2e", true},

		// Malformed percent-escape cannot round-trip to a safe segment.
		{"malformed percent", "a%2gb", true},
		{"trailing percent", "toolbox%", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateToolboxName(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("validateToolboxName(%q) = nil, want error", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateToolboxName(%q) = %v, want nil", tt.input, err)
			}
		})
	}
}

// TestToolboxRequestBuildersRejectUnsafeNames verifies that every toolbox
// request builder that interpolates a name into the request path refuses a
// target-altering name before building the request, and still accepts a valid
// one. The *CreateRequest methods only assemble the request (no network I/O),
// so a bare client with just an endpoint is enough to exercise them.
func TestToolboxRequestBuildersRejectUnsafeNames(t *testing.T) {
	client := &ToolboxesClient{endpoint: "https://example.com"}
	ctx := context.Background()

	builders := map[string]func(name string) error{
		"CreateToolboxVersion": func(name string) error {
			_, err := client.createToolboxVersionCreateRequest(ctx, name, nil, nil)
			return err
		},
		"DeleteToolbox": func(name string) error {
			_, err := client.deleteToolboxCreateRequest(ctx, name, nil)
			return err
		},
		"DeleteToolboxVersion": func(name string) error {
			_, err := client.deleteToolboxVersionCreateRequest(ctx, name, "v1", nil)
			return err
		},
		"GetToolbox": func(name string) error {
			_, err := client.getToolboxCreateRequest(ctx, name, nil)
			return err
		},
		"GetToolboxVersion": func(name string) error {
			_, err := client.getToolboxVersionCreateRequest(ctx, name, "v1", nil)
			return err
		},
		"ListToolboxVersions": func(name string) error {
			_, err := client.listToolboxVersionsCreateRequest(ctx, name, &ToolboxesClientListToolboxVersionsOptions{})
			return err
		},
		"UpdateToolbox": func(name string) error {
			_, err := client.updateToolboxCreateRequest(ctx, name, "v1", nil)
			return err
		},
	}

	for op, build := range builders {
		t.Run(op+"/rejects_traversal", func(t *testing.T) {
			if err := build(".."); err == nil {
				t.Errorf("%s built a request for name %q, want error", op, "..")
			}
		})
		t.Run(op+"/rejects_encoded_separator", func(t *testing.T) {
			if err := build("a%2Fb"); err == nil {
				t.Errorf("%s built a request for name %q, want error", op, "a%2Fb")
			}
		})
		t.Run(op+"/accepts_valid_name", func(t *testing.T) {
			if err := build("my-toolbox"); err != nil {
				t.Errorf("%s(%q) returned %v, want nil", op, "my-toolbox", err)
			}
		})
	}
}
