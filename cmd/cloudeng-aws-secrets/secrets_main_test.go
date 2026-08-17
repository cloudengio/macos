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

	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/aws/awstestutil"
	"cloudeng.io/cmdutil/keys"
	"cloudeng.io/file"
	"cloudeng.io/file/filetestutil"
	"cloudeng.io/security/keys/keychain/keychaintestutil"
)

var awsInstance *awstestutil.AWS

func TestMain(m *testing.M) {
	awstestutil.AWSTestMain(m, &awsInstance, awstestutil.WithSecretsManager())
}

func runCLI(ctx context.Context, args ...string) error {
	cmd := cli()
	return cmd.DispatchWithArgs(ctx, "secrets", args...)
}

func TestErrors(t *testing.T) {
	ctx := context.Background()
	p := keychaintestutil.New()
	inMemFS := keychaintestutil.NewFS(p, true)
	ctx = file.ContextWithReadWriteFS(ctx, inMemFS)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "secret.txt")
	_ = os.WriteFile(tmpFile, []byte("val"), 0600)

	// 1. Missing ARN
	if err := runCLI(ctx, "read", tmpFile); err == nil || !strings.Contains(err.Error(), "missing secret ARN or name") {
		t.Errorf("expected missing ARN error, got: %v", err)
	}

	// 2. Missing keychain item
	if err := runCLI(ctx, "read", "--arn=test-secret", tmpFile); err == nil || !strings.Contains(err.Error(), "no keychain item provided") {
		t.Errorf("expected missing keychain item error, got: %v", err)
	}

	// 3. Missing key info ID
	if err := runCLI(ctx, "read", "--arn=test-secret", "--keychain-item=kc-item", tmpFile); err == nil || !strings.Contains(err.Error(), "no key info ID provided") {
		t.Errorf("expected missing key info ID error, got: %v", err)
	}

	// 4. Missing keychain file in store
	if err := runCLI(ctx, "read", "--arn=test-secret", "--keychain-item=missing-kc", "--aws-key-info-id=my-key", tmpFile); err == nil || !strings.Contains(err.Error(), "failed to read") {
		t.Errorf("expected failed to read error, got: %v", err)
	}

	// 5. Keychain file exists but does not contain the requested key ID
	ims := keys.NewInMemoryKeyStore()
	if err := ims.WriteYAML(ctx, inMemFS, "empty-kc", 0600); err != nil {
		t.Fatalf("writing empty keystore: %v", err)
	}
	if err := runCLI(ctx, "read", "--arn=test-secret", "--keychain-item=empty-kc", "--aws-key-info-id=nonexistent-key", tmpFile); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected key info not found error, got: %v", err)
	}
}

func TestHandleError(t *testing.T) {
	if err := handleError(nil); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	sampleErr := errors.New("sample error")
	if err := handleError(sampleErr); !errors.Is(err, sampleErr) {
		t.Errorf("expected sampleErr, got %v", err)
	}
}

func TestIntegration(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	t.Setenv("AWS_ENDPOINT_URL", awsInstance.URL())
	t.Setenv("AWS_REGION", "us-east-1")
	ctx := context.Background()

	p := keychaintestutil.New()
	inMemFS := keychaintestutil.NewFS(p, true)
	ctx = file.ContextWithReadWriteFS(ctx, inMemFS)

	// Set up AWS credentials in in-memory keychain store
	ki := awsconfig.NewKeyInfo("test-key", "test-user", []byte("test-secret-key"), awsconfig.KeyInfoExtra{
		AccessKeyID: "test-access-key",
		Region:      "us-east-1",
	})
	ims := keys.NewInMemoryKeyStore()
	ims.Add(ki)
	if err := ims.WriteYAML(ctx, inMemFS, "aws-keychain", 0600); err != nil {
		t.Fatalf("writing keystore to in-memory FS: %v", err)
	}

	// Inject AWS config connected to local mock SecretsManager
	cfg := awstestutil.DefaultAWSConfig()
	resClient := awsInstance.SecretsManager(cfg)
	_ = resClient
	ctx = awsconfig.ContextWith(ctx, cfg)

	tmpDir := t.TempDir()
	secretFile := filepath.Join(tmpDir, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("super-secret-aws-value"), 0600); err != nil {
		t.Fatalf("writing secret file: %v", err)
	}

	arnName := "arn:aws:secretsmanager:us-east-1:000000000000:secret:test-secret"

	// 1. Write secret to AWS SecretsManager
	if err := runCLI(ctx, "write",
		"--arn="+arnName,
		"--keychain-item=aws-keychain",
		"--aws-key-info-id=test-key",
		"--aws-key-info-user=test-user",
		secretFile); err != nil {
		t.Fatalf("runCLI write: %v", err)
	}

	// 2. Read back secret from AWS SecretsManager
	outFile := filepath.Join(tmpDir, "read-back.txt")
	if err := runCLI(ctx, "read",
		"--arn="+arnName,
		"--keychain-item=aws-keychain",
		"--aws-key-info-id=test-key",
		"--aws-key-info-user=test-user",
		outFile); err != nil {
		t.Fatalf("runCLI read: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading read-back file: %v", err)
	}
	if string(data) != "super-secret-aws-value" {
		t.Errorf("read secret = %q, want super-secret-aws-value", string(data))
	}

	// 3. Test writing from stdin
	err = filetestutil.FeedStdin("stdin-aws-secret-data", func() error {
		return runCLI(ctx, "write",
			"--arn=arn:aws:secretsmanager:us-east-1:000000000000:secret:stdin-secret",
			"--keychain-item=aws-keychain",
			"--aws-key-info-id=test-key",
			"--aws-key-info-user=test-user",
			"-")
	})
	if err != nil {
		t.Fatalf("write from stdin: %v", err)
	}

	// 4. Test reading to stdout
	out, err := filetestutil.CaptureStdout(func() error {
		return runCLI(ctx, "read",
			"--arn=arn:aws:secretsmanager:us-east-1:000000000000:secret:stdin-secret",
			"--keychain-item=aws-keychain",
			"--aws-key-info-id=test-key",
			"--aws-key-info-user=test-user",
			"-")
	})
	if err != nil {
		t.Fatalf("read to stdout: %v", err)
	}
	if string(out) != "stdin-aws-secret-data" {
		t.Errorf("stdout = %q, want stdin-aws-secret-data", out)
	}
}
