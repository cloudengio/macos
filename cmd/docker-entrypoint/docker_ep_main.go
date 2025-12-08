// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"

	"cloudeng.io/cmdutil/subcmd"
)

const cmdSpec = `name: docker-entrypoint
summary: utility to run docker commands with secrets piped into the container
		and read by the entrypoint command. A container should have
		'docker-entrypoint entrypoint' as its entrypoint and the container
		can be run with
		   'docker-entrypoint run <docker run flags>...'
		When run on macos the keychain-item flag can be used to specify
		a keychain item containing keys in cloudeng.io/cmdutil/keys format
		that will be piped into the container. The entrypoint command will
		read the keys from the pipe and write them to the keyring. If the
		keychain item contains a key with id 'my-key' and value 'my-value'
		then the entrypoint command will write a key to the linux
		session keyring named 'my-key' with value 'my-value'.
commands:
  - name: run
    summary: "run a command with secrets piped into the container. Note
		run will automatically add 'run -i -t --security-opt seccomp=<profile>'
		to the docker run command line. Where profile is set to a temp file
		containing a seccomp profile that allows access to the linux kernel
		key ring. This profile is created by the 'create-seccomp-profile'
		command."
    arguments:
      - <args>... - arguments that are passed to docker run
  - name: entrypoint
    summary: entrypoint command to run inside a container
    arguments:
      - <args>... - arguments passed to the entrypoint command
  - name: create-seccomp-profile
    summary: create a seccomp profile that allows access to the linux kernel key ring
    arguments:
`

func cli() *subcmd.CommandSetYAML {
	cmd := subcmd.MustFromYAML(cmdSpec)
	var dockerCmds dockerCmds
	cmd.Set("run").MustRunner(dockerCmds.Run, &RunFlags{})
	cmd.Set("entrypoint").MustRunner(dockerCmds.Entry, &EntryFlags{})
	cmd.Set("create-seccomp-profile").MustRunner(dockerCmds.createSeccompProfile, &seccompFlags{})
	return cmd
}

func main() {
	ctx := context.Background()
	subcmd.Dispatch(ctx, cli())
}

type dockerCmds struct{}

func (dc dockerCmds) Run(ctx context.Context, f any, args []string) error {
	return dc.run(ctx, f, args)
}

func (dc dockerCmds) Entry(ctx context.Context, f any, args []string) error {
	return dc.entry(ctx, f, args)
}
