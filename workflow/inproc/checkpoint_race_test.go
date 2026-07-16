// Copyright (c) Microsoft. All rights reserved.

package inproc_test

import (
	"context"
	"sync"
	"testing"

	"github.com/microsoft/agent-framework-go/workflow/checkpoint"
	"github.com/microsoft/agent-framework-go/workflow/inproc"
)

// TestStreamingRun_ConcurrentCheckpointAccess_NoDataRace exercises the public
// checkpoint accessors (Checkpoints / LastCheckpoint) concurrently with a live
// streaming run. The background run loop writes the runner's checkpoint state
// during supersteps (initial run, after a response, and on restore) while a
// consumer polls those accessors — a supported usage for progress monitoring.
//
// Without synchronization the read of runner.checkpoints in Checkpoints()
// races the append in the checkpoint step; run under -race this test flags it.
func TestStreamingRun_ConcurrentCheckpointAccess_NoDataRace(t *testing.T) {
	ctx := context.Background()
	wf, _ := createCheckpointRequestWorkflow(t)
	manager := checkpoint.NewInMemoryManager()

	run, err := inproc.Default.WithCheckpointing(manager).RunStreaming(ctx, wf, "Hello")
	if err != nil {
		t.Fatalf("RunStreaming: %v", err)
	}
	t.Cleanup(func() {
		if err := run.Close(ctx); err != nil {
			t.Errorf("Close run: %v", err)
		}
	})

	// Poll the checkpoint accessors from another goroutine for the whole run.
	stop := make(chan struct{})
	var pollWG sync.WaitGroup
	pollWG.Add(1)
	go func() {
		defer pollWG.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = run.Checkpoints()
				_, _ = run.LastCheckpoint()
			}
		}
	}()

	// Drive the run through several checkpoint-producing supersteps.
	pendingRequest, checkpointInfo := capturePendingRequestAndCheckpointFromStream(t, ctx, run)
	response, err := pendingRequest.CreateResponse("World")
	if err != nil {
		t.Fatalf("CreateResponse: %v", err)
	}
	if err := run.SendResponse(ctx, response); err != nil {
		t.Fatalf("SendResponse: %v", err)
	}
	readStreamToHalt(t, ctx, run)

	// Restore also writes the runner's checkpoint fields.
	if err := run.RestoreCheckpoint(ctx, checkpointInfo); err != nil {
		t.Fatalf("RestoreCheckpoint: %v", err)
	}
	readStreamToHalt(t, ctx, run)

	close(stop)
	pollWG.Wait()
}
