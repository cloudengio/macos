// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package tartvm

import (
	"strings"
	"testing"
	"time"

	"cloudeng.io/vms"
)

// runningInstance returns an instance in StateRunning that fetches its IP on
// demand rather than at start.
func runningInstance(t *testing.T) *Instance {
	t.Helper()
	inst := New(t.Context(), "test-source", "test-vm",
		WithObtainIPAtStart(false),
		WithRunTimeout(2*time.Second))
	inst.setState(vms.StateRunning)
	return inst
}

// TestGetIPOnDemand verifies that the first call for a running VM fetches the
// address with "tart ip" and returns it to the caller.
func TestGetIPOnDemand(t *testing.T) {
	invocations := fakeTart(t, "  192.168.64.5  ", "", 0)
	inst := runningInstance(t)

	ip, err := inst.getIP(t.Context())
	if err != nil {
		t.Fatalf("getIP: %v", err)
	}
	if got, want := ip, "192.168.64.5"; got != want {
		t.Errorf("ip: got %q, want %q", got, want)
	}
	got := invocations()
	if len(got) != 1 {
		t.Fatalf("tart invocations: got %v, want exactly one", got)
	}
	if want := "ip test-vm --wait 2"; got[0] != want {
		t.Errorf("tart args: got %q, want %q", got[0], want)
	}
}

// TestGetIPCached verifies that the address is fetched once and served from the
// instance thereafter.
func TestGetIPCached(t *testing.T) {
	invocations := fakeTart(t, "192.168.64.5", "", 0)
	inst := runningInstance(t)

	first, err := inst.getIP(t.Context())
	if err != nil {
		t.Fatalf("getIP: %v", err)
	}
	second, err := inst.getIP(t.Context())
	if err != nil {
		t.Fatalf("getIP (cached): %v", err)
	}
	if first != second {
		t.Errorf("ip: got %q then %q, want the same address both times", first, second)
	}
	if got := invocations(); len(got) != 1 {
		t.Errorf("tart invocations: got %v, want exactly one, the second call should be served from the cache", got)
	}
}

// TestGetIPEmpty verifies that tart reporting no address is an error rather
// than a silently empty IP, and that nothing is cached.
func TestGetIPEmpty(t *testing.T) {
	fakeTart(t, "", "", 0)
	inst := runningInstance(t)

	ip, err := inst.getIP(t.Context())
	if err == nil {
		t.Fatal("getIP: got nil error, want an error for an empty address")
	}
	if !strings.Contains(err.Error(), "empty IP address") {
		t.Errorf("error %q does not mention an empty IP address", err)
	}
	if ip != "" {
		t.Errorf("ip: got %q, want empty", ip)
	}
	if got := inst.currentIP; got != "" {
		t.Errorf("currentIP: got %q, want it left unset", got)
	}
}

// TestGetIPError verifies that a failing tart invocation is reported and
// nothing is cached.
func TestGetIPError(t *testing.T) {
	fakeTart(t, "", "tart: no such VM", 1)
	inst := runningInstance(t)

	ip, err := inst.getIP(t.Context())
	if err == nil {
		t.Fatal("getIP: got nil error, want an error")
	}
	if !strings.Contains(err.Error(), "failed to get IP address") {
		t.Errorf("error %q does not mention the failure to get an IP address", err)
	}
	if ip != "" {
		t.Errorf("ip: got %q, want empty", ip)
	}
	if got := inst.currentIP; got != "" {
		t.Errorf("currentIP: got %q, want it left unset", got)
	}
}

// TestGetIPNotFetched covers the cases where no lookup should be attempted at
// all: the IP was already obtained at start, and the VM is not running.
func TestGetIPNotFetched(t *testing.T) {
	for _, tc := range []struct {
		name      string
		ipAtStart bool
		state     vms.State
		currentIP string
		want      string
	}{
		{"obtained at start", true, vms.StateRunning, "10.0.0.1", "10.0.0.1"},
		{"stopped", false, vms.StateStopped, "", ""},
		{"suspended", false, vms.StateSuspended, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invocations := fakeTart(t, "192.168.64.5", "", 0)
			inst := New(t.Context(), "test-source", "test-vm",
				WithObtainIPAtStart(tc.ipAtStart),
				WithRunTimeout(2*time.Second))
			inst.setState(tc.state)
			inst.currentIP = tc.currentIP

			ip, err := inst.getIP(t.Context())
			if err != nil {
				t.Fatalf("getIP: %v", err)
			}
			if got, want := ip, tc.want; got != want {
				t.Errorf("ip: got %q, want %q", got, want)
			}
			if got := invocations(); got != nil {
				t.Errorf("tart invocations: got %v, want none", got)
			}
		})
	}
}
