// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin

package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"cloudeng.io/macos/keychain"
)

var (
	write         bool
	account       string
	service       string
	keychainType  string
	updateInPlace bool
)

func usage() {
	name := filepath.Base(os.Args[0])
	fmt.Printf("%s <flags> [filename|-]\n", name)
	flag.PrintDefaults()
}

func usageAndExit() {
	usage()
	os.Exit(1)
}

func initFlags() {
	flag.BoolVar(&write, "write", false, "set to true to write a note instead of reading")
	flag.StringVar(&keychainType, "keychain-type", "file", "keychain type: file, data-protection, or icloud")
	flag.StringVar(&account, "account", "", "keychain account that the note belongs to")
	flag.StringVar(&service, "service", "", "keychain service that the note belongs to")
	flag.BoolVar(&updateInPlace, "update-in-place", false, "set to true to update existing note in place")

}

func getUser() (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	return user.Username, nil
}

func main() {
	initFlags()
	flag.Parse()
	if len(service) == 0 {
		fmt.Fprintf(os.Stderr, "-service must be specified\n")
		usageAndExit()
	}
	kt, err := keychain.ParseKeychainType(keychainType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		usageAndExit()
	}
	if len(account) == 0 {
		var err error
		account, err = getUser()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error getting current user: %v\n", err)
			return
		}
	}
	args := flag.Args()
	if write {
		if len(args) != 1 {
			fmt.Fprintf(os.Stderr, "need a single filename argument\n")
			usageAndExit()
		}
		writeNote(kt, args)
		return
	}
	readNote(kt)
}

func writeNote(kt keychain.Type, args []string) {
	kc := keychain.New(kt, account, keychain.WithUpdateInPlace(updateInPlace))
	data, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading file: %v\n", err)
		return
	}
	err = kc.WriteSecureNote(service, data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing note: account %s, service %s, error: %v\n", account, service, err)
		return
	}
	fmt.Printf("note written successfully: account %s, service %s\n", account, service)
}

func readNote(kt keychain.Type) {
	kc := keychain.NewReadonly(kt, account)
	data, err := kc.ReadSecureNote(service)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading note: %v\n", err)
		return
	}
	fmt.Printf("%s\n", data)
}
