// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package tartvm

import (
	"errors"
	"testing"

	"cloudeng.io/vms"
)

// TestDeleteFromInitial verifies that Delete is permitted from Initial, rather
// than rejected by the state machine, so that a clone which failed or was
// interrupted part way through can be cleaned up.
func TestDeleteFromInitial(t *testing.T) {
	invocations := fakeTart(t, "", "", 0)
	inst := New(t.Context(), "test-source", "test-vm")

	if got, want := inst.State(t.Context()), vms.StateInitial; got != want {
		t.Fatalf("state before delete: got %v, want %v", got, want)
	}
	if err := inst.Delete(t.Context()); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got, want := inst.State(t.Context()), vms.StateDeleted; got != want {
		t.Errorf("state after delete: got %v, want %v", got, want)
	}
	got := invocations()
	if len(got) != 1 {
		t.Fatalf("tart invocations: got %v, want exactly one", got)
	}
	if want := "delete test-vm"; got[0] != want {
		t.Errorf("tart args: got %q, want %q", got[0], want)
	}
}

// TestDeleteMissingVM verifies that deleting a VM tart does not know about
// reports vms.ErrVMNotFound, which is how a caller cleaning up after an
// interrupted clone distinguishes "nothing to do" from a real failure.
func TestDeleteMissingVM(t *testing.T) {
	fakeTart(t, "", `the specified VM "test-vm" does not exist`, 2)
	inst := New(t.Context(), "test-source", "test-vm")

	err := inst.Delete(t.Context())
	if err == nil {
		t.Fatal("Delete: got nil error, want an error")
	}
	if !errors.Is(err, vms.ErrVMNotFound) {
		t.Errorf("Delete: got %v, want it to wrap %v", err, vms.ErrVMNotFound)
	}
	// A failed delete must leave the instance in its previous state so that the
	// caller can retry rather than believing the VM is gone.
	if got, want := inst.State(t.Context()), vms.StateInitial; got != want {
		t.Errorf("state after a failed delete: got %v, want %v", got, want)
	}
}
