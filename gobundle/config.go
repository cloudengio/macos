// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"cloudeng.io/macos/buildtools"
	"gopkg.in/yaml.v3"
)

const (
	verboseEnvVar      = "GOBUNDLE_VERBOSE"
	sharedBundleEnvVar = "GOBUNDLE_SHARED_CONFIG"
	sharedConfigFile   = "gobundle-shared"
	appBundleEnvVar    = "GOBUNDLE_APP_CONFIG"
	appConfigFile      = "gobundle-app"
)

type config struct {
	buildtools.SigningConfig `yaml:",inline"`
	Path                     string               `yaml:"bundle"`
	Info                     buildtools.InfoPlist `yaml:"info.plist"`
	ProvisioningProfile      string               `yaml:"profile"`
}

func readconfig(file string) (map[string]any, error) {
	cfg := map[string]any{}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func configFileNames(name string) []string {
	return []string{
		name + ".yaml",
		name + ".yml",
		"." + name + ".yaml",
		"." + name + ".yml",
		filepath.Join(os.Getenv("HOME"), name+".yaml"),
		filepath.Join(os.Getenv("HOME"), name+".yml"),
		filepath.Join(os.Getenv("HOME"), "."+name+".yaml"),
		filepath.Join(os.Getenv("HOME"), "."+name+".yml"),
	}
}

func loadconfig(envVar, filename string) (string, map[string]any, error) {
	files := append([]string{os.Getenv(envVar)}, configFileNames(filename)...)
	for _, file := range files {
		if file != "" {
			cfg, err := readconfig(file)
			if err == nil {
				return file, cfg, nil
			}
			if !os.IsNotExist(err) {
				return "", nil, fmt.Errorf("error reading config file %s: %v", file, err)
			}
		}
	}
	return "", map[string]any{}, os.ErrNotExist
}

func readAndMergeConfigs() ([]byte, error) {
	sharedFile, shared, err := loadconfig(sharedBundleEnvVar, sharedConfigFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error loading shared config: %v", err)
	}
	appFile, app, err := loadconfig(appBundleEnvVar, appConfigFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error loading app config: %v", err)
	}
	deepMergeMaps(app, shared)
	merged, err := yaml.Marshal(app)
	if err != nil {
		return nil, fmt.Errorf("error marshaling merged config: %v", err)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "Merged config from %s and %s:\n%s\n", sharedFile, appFile, merged)
	}
	return merged, nil
}

// deepMergeMaps recursively merges the contents of src map into dst map.
// Values from src will override values in dst for conflicting keys.
func deepMergeMaps(dst, src map[string]any) {
	for key, srcVal := range src {
		if dstVal, ok := dst[key]; ok {
			// If both values are maps, recurse
			if dstMap, isDstMap := dstVal.(map[string]any); isDstMap {
				if srcMap, isSrcMap := srcVal.(map[string]any); isSrcMap {
					deepMergeMaps(dstMap, srcMap)
					continue
				}
			}
		}
		// If no recursion or types don't match for recursion,
		// or key is new, override/set the value
		if _, islist := srcVal.([]any); islist {
			dst[key] = append(dst[key].([]any), srcVal.([]any)...)
			continue
		}
		dst[key] = srcVal
	}
}

func configFromMerged(merged []byte, binary string) (config, error) {
	raw := map[string]any{}
	if err := yaml.Unmarshal(merged, &raw); err != nil {
		return config{}, fmt.Errorf("error unmarshaling merged config: %v", err)
	}
	binary = filepath.Base(binary)
	rawInfo := map[string]any{}
	if plist := raw["info.plist"]; plist != nil {
		rawInfo = raw["info.plist"].(map[string]any)
	}
	provideDefault(rawInfo, "CFBundleName", binary)
	provideDefault(rawInfo, "CFBundlePackageType", "APPL")
	provideDefault(rawInfo, "CFBundleExecutable", binary)
	provideDefault(rawInfo, "CFBundleDisplayName", binary)
	provideDefault(rawInfo, "CFBundleVersion", "0.0.0")
	provideDefault(rawInfo, "LSMinimumSystemVersion", "10.15")
	updated, err := yaml.Marshal(raw)
	if err != nil {
		return config{}, fmt.Errorf("error marshaling updated config: %v", err)
	}
	var cfg config
	if err := yaml.Unmarshal(updated, &cfg); err != nil {
		return config{}, fmt.Errorf("error unmarshaling merged config: %v", err)
	}
	return cfg, nil
}

func provideDefault(raw map[string]any, key, defaultValue string) {
	if _, ok := raw[key]; !ok {
		raw[key] = defaultValue
	}
}
