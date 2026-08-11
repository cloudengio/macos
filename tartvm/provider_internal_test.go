// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package tartvm

import (
	"context"
	"testing"

	"cloudeng.io/vms"
	"cloudeng.io/vms/vmspool"
)

func TestNewProviderWiring(t *testing.T) {
	var calls int
	ctor := func(context.Context) (vms.Instance, error) {
		calls++
		return nil, nil
	}
	p := NewProvider(ctor, WithNamePrefix("pfx-"), WithPoolName("mypool"))

	// New delegates to the supplied constructor.
	if _, err := p.New(context.Background()); err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := p.New(context.Background()); err != nil {
		t.Fatalf("New: %v", err)
	}
	if calls != 2 {
		t.Errorf("constructor called %d times, want 2", calls)
	}
	if p.prefix != "pfx-" {
		t.Errorf("prefix = %q, want pfx-", p.prefix)
	}
	if p.pool != "mypool" {
		t.Errorf("pool = %q, want mypool", p.pool)
	}
	// The Provider satisfies vmspool.Provider.
	var _ vmspool.Provider = p
}
