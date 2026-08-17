// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/keys"
	"cloudeng.io/cmdutil/keys/keyscmd"
	"cloudeng.io/file"
	"cloudeng.io/file/filetestutil"
	"cloudeng.io/security/keys/keychain/keychaintestutil"
	"cloudeng.io/security/keys/keychain/plugins"
)

func runCLI(ctx context.Context, args ...string) error {
	cmd := cli()
	return cmd.DispatchWithArgs(ctx, "keychain", args...)
}

func TestReadWriteWithInjectedFS(t *testing.T) {
	ctx := context.Background()
	p := keychaintestutil.New()
	inMemFS := keychaintestutil.NewFS(p, true)
	ctx = file.ContextWithReadWriteFS(ctx, inMemFS)

	tmpDir := t.TempDir()
	dataFile := filepath.Join(tmpDir, "secret.txt")
	if err := os.WriteFile(dataFile, []byte("super-secret-payload"), 0600); err != nil {
		t.Fatalf("writing secret file: %v", err)
	}

	// 1. Write file to keychain item
	if err := runCLI(ctx, "write", "--keychain-item=test-item", dataFile); err != nil {
		t.Fatalf("runCLI write: %v", err)
	}

	// Verify key in in-memory plugin
	stored, ok := p.Get("test-item")
	if !ok || string(stored) != "super-secret-payload" {
		t.Fatalf("p.Get = %q, %v, want super-secret-payload, true", stored, ok)
	}

	// 2. Read from keychain item to a new file
	outFile := filepath.Join(tmpDir, "read-back.txt")
	if err := runCLI(ctx, "read", "--keychain-item=test-item", outFile); err != nil {
		t.Fatalf("runCLI read: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading read-back file: %v", err)
	}
	if string(data) != "super-secret-payload" {
		t.Errorf("read data = %q, want super-secret-payload", string(data))
	}
}

func TestReadWritePipes(t *testing.T) {
	ctx := context.Background()
	p := keychaintestutil.New()
	inMemFS := keychaintestutil.NewFS(p, true)
	ctx = file.ContextWithReadWriteFS(ctx, inMemFS)

	// 1. Write from stdin
	err := filetestutil.FeedStdin("pipe-secret-data", func() error {
		return runCLI(ctx, "write", "--keychain-item=pipe-item", "-")
	})
	if err != nil {
		t.Fatalf("write from stdin: %v", err)
	}

	stored, ok := p.Get("pipe-item")
	if !ok || string(stored) != "pipe-secret-data" {
		t.Fatalf("p.Get(pipe-item) = %q, %v, want pipe-secret-data, true", stored, ok)
	}

	// 2. Read to stdout
	out, err := filetestutil.CaptureStdout(func() error {
		return runCLI(ctx, "read", "--keychain-item=pipe-item", "-")
	})
	if err != nil {
		t.Fatalf("read to stdout: %v", err)
	}
	if string(out) != "pipe-secret-data" {
		t.Errorf("read stdout = %q, want pipe-secret-data", out)
	}
}

func TestKeyInfoLifecycle(t *testing.T) {
	ctx := context.Background()
	p := keychaintestutil.New()
	inMemFS := keychaintestutil.NewFS(p, true)
	ctx = file.ContextWithReadWriteFS(ctx, inMemFS)

	tmpDir := t.TempDir()
	kiFile1 := filepath.Join(tmpDir, "key1.json")
	kiFile2 := filepath.Join(tmpDir, "key2.json")

	// 1. Create KeyInfo files using key-info create
	if err := runCLI(ctx, "key-info", "create", "--user=alice", "--id=token-1", "--size=16", "--format=hex", kiFile1); err != nil {
		t.Fatalf("key-info create 1: %v", err)
	}
	if err := runCLI(ctx, "key-info", "create", "--user=bob", "--id=token-2", "--size=16", "--format=hex", kiFile2); err != nil {
		t.Fatalf("key-info create 2: %v", err)
	}

	// Verify KeyInfo created
	ki1, err := keyscmd.ReadKeyInfoFromLocalJSON(ctx, kiFile1)
	if err != nil {
		t.Fatalf("reading created keyinfo 1: %v", err)
	}
	if ki1.User != "alice" || ki1.ID != "token-1" || len(ki1.Token().Value()) != 32 {
		t.Fatalf("unexpected ki1: user=%s id=%s tokenLen=%d", ki1.User, ki1.ID, len(ki1.Token().Value()))
	}

	// 2. Set KeyInfos into keychain item
	if err := runCLI(ctx, "key-info", "set", "--keychain-item=my-keys", kiFile1); err != nil {
		t.Fatalf("key-info set 1: %v", err)
	}
	if err := runCLI(ctx, "key-info", "set", "--keychain-item=my-keys", kiFile2); err != nil {
		t.Fatalf("key-info set 2: %v", err)
	}

	// 3. List keys in keychain item
	listOut, err := filetestutil.CaptureStdout(func() error {
		return runCLI(ctx, "key-info", "list", "--keychain-item=my-keys")
	})
	if err != nil {
		t.Fatalf("key-info list: %v", err)
	}
	if !strings.Contains(string(listOut), "token-1[alice]") || !strings.Contains(string(listOut), "token-2[bob]") {
		t.Errorf("list output missing keys: %s", listOut)
	}

	// 4. Get a specific KeyInfo
	getOutFile := filepath.Join(tmpDir, "got-alice.json")
	if err := runCLI(ctx, "key-info", "get", "--keychain-item=my-keys", "--key-user=alice", "--key-id=token-1", getOutFile); err != nil {
		t.Fatalf("key-info get: %v", err)
	}
	gotKi, err := keyscmd.ReadKeyInfoFromLocalJSON(ctx, getOutFile)
	if err != nil {
		t.Fatalf("reading got keyinfo: %v", err)
	}
	if gotKi.User != "alice" || gotKi.ID != "token-1" || string(gotKi.Token().Value()) != string(ki1.Token().Value()) {
		t.Errorf("got unexpected KeyInfo: %+v", gotKi)
	}

	// 5. Delete alice / token-1
	if err := runCLI(ctx, "key-info", "delete", "--keychain-item=my-keys", "--key-user=alice", "--key-id=token-1"); err != nil {
		t.Fatalf("key-info delete: %v", err)
	}

	// Verify alice is deleted and bob remains
	delGetOut := filepath.Join(tmpDir, "del-check.json")
	if err := runCLI(ctx, "key-info", "get", "--keychain-item=my-keys", "--key-user=alice", "--key-id=token-1", delGetOut); err == nil {
		t.Errorf("expected error getting deleted key, got nil")
	}

	getBobOut := filepath.Join(tmpDir, "got-bob.json")
	if err := runCLI(ctx, "key-info", "get", "--keychain-item=my-keys", "--key-user=bob", "--key-id=token-2", getBobOut); err != nil {
		t.Fatalf("getting remaining key bob: %v", err)
	}
}

func TestErrors(t *testing.T) {
	ctx := context.Background()
	p := keychaintestutil.New()
	inMemFS := keychaintestutil.NewFS(p, true)
	ctx = file.ContextWithReadWriteFS(ctx, inMemFS)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "file.txt")
	_ = os.WriteFile(tmpFile, []byte("data"), 0600)

	// 1. Missing keychain item on read
	if err := runCLI(ctx, "read", tmpFile); err == nil || !strings.Contains(err.Error(), "missing keychain item") {
		t.Errorf("expected missing keychain item error, got: %v", err)
	}

	// 2. Missing keychain item on write
	if err := runCLI(ctx, "write", tmpFile); err == nil || !strings.Contains(err.Error(), "missing keychain item") {
		t.Errorf("expected missing keychain item error, got: %v", err)
	}

	// 3. Read non-existent item
	if err := runCLI(ctx, "read", "--keychain-item=does-not-exist", tmpFile); err == nil {
		t.Errorf("expected error reading non-existent item, got nil")
	}

	// 4. Injected error on read
	p.SetError("error-item", &plugins.Error{Message: "injected failure", Detail: "detail"})
	if err := runCLI(ctx, "read", "--keychain-item=error-item", tmpFile); err == nil {
		t.Errorf("expected error on injected failure, got nil")
	}

	// 5. Injected error on delete
	p.SetError("del-error", &plugins.Error{Message: "cannot delete", Detail: "detail"})
	if err := runCLI(ctx, "key-info", "delete", "--keychain-item=del-error", "--key-user=u", "--key-id=k"); err == nil {
		t.Errorf("expected error on injected delete error, got nil")
	}
}

func TestKeyInfoCreateWithManualJSON(t *testing.T) {
	ctx := context.Background()
	p := keychaintestutil.New()
	inMemFS := keychaintestutil.NewFS(p, true)
	ctx = file.ContextWithReadWriteFS(ctx, inMemFS)

	tmpDir := t.TempDir()
	customFile := filepath.Join(tmpDir, "custom.json")

	ki := keys.NewInfo("api-key-1", "service-user", []byte("my-static-token"))
	if err := keyscmd.SafeWriteKeyInfoJSON(ctx, ki, customFile, 0600); err != nil {
		t.Fatalf("SafeWriteKeyInfoJSON: %v", err)
	}

	// Set custom key into keychain
	if err := runCLI(ctx, "key-info", "set", "--keychain-item=service-store", customFile); err != nil {
		t.Fatalf("key-info set: %v", err)
	}

	// Get custom key back
	readBackFile := filepath.Join(tmpDir, "readback.json")
	if err := runCLI(ctx, "key-info", "get", "--keychain-item=service-store", "--key-user=service-user", "--key-id=api-key-1", readBackFile); err != nil {
		t.Fatalf("key-info get: %v", err)
	}

	gotKi, err := keyscmd.ReadKeyInfoFromLocalJSON(ctx, readBackFile)
	if err != nil {
		t.Fatalf("ReadKeyInfoFromLocalJSON: %v", err)
	}
	if gotKi.ID != "api-key-1" || gotKi.User != "service-user" || string(gotKi.Token().Value()) != "my-static-token" {
		t.Errorf("gotKi mismatch: %+v, token: %s", gotKi, string(gotKi.Token().Value()))
	}
}

func TestHandleErrorHelper(t *testing.T) {
	if err := handleError(nil); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	genericErr := errors.New("plain error")
	if err := handleError(genericErr); !errors.Is(err, genericErr) {
		t.Errorf("expected genericErr, got %v", err)
	}

	pluginErrWithStderr := &plugins.Error{
		Message: "plugin failed",
		Detail:  "some detail",
		Stderr:  "plugin stderr text",
	}
	out, err := filetestutil.CaptureStdout(func() error {
		return handleError(pluginErrWithStderr)
	})
	if !errors.Is(err, pluginErrWithStderr) {
		t.Errorf("expected pluginErrWithStderr, got %v", err)
	}
	if !strings.Contains(string(out), "plugin failed") || !strings.Contains(string(out), "plugin stderr text") {
		t.Errorf("stdout missing expected error text: %s", out)
	}
}
