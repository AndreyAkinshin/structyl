// Package schema provides JSON schema validation for structyl configuration files.
package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"

	schemafs "github.com/AndreyAkinshin/structyl/schema"
)

var (
	configSchema     *jsonschema.Schema
	toolchainsSchema *jsonschema.Schema
	compileOnce      sync.Once
	compileErr       error
)

// compileSchemas compiles all embedded schemas once.
func compileSchemas() error {
	compileOnce.Do(func() {
		compiler := jsonschema.NewCompiler()

		configData, err := schemafs.FS.ReadFile("config.schema.json")
		if err != nil {
			compileErr = fmt.Errorf("read config schema: %w", err)
			return
		}

		toolchainsData, err := schemafs.FS.ReadFile("toolchains.schema.json")
		if err != nil {
			compileErr = fmt.Errorf("read toolchains schema: %w", err)
			return
		}

		configDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(configData))
		if err != nil {
			compileErr = fmt.Errorf("unmarshal config schema: %w", err)
			return
		}

		toolchainsDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(toolchainsData))
		if err != nil {
			compileErr = fmt.Errorf("unmarshal toolchains schema: %w", err)
			return
		}

		if err := compiler.AddResource("config.schema.json", configDoc); err != nil {
			compileErr = fmt.Errorf("add config schema resource: %w", err)
			return
		}

		if err := compiler.AddResource("toolchains.schema.json", toolchainsDoc); err != nil {
			compileErr = fmt.Errorf("add toolchains schema resource: %w", err)
			return
		}

		configSchema, err = compiler.Compile("config.schema.json")
		if err != nil {
			compileErr = fmt.Errorf("compile config schema: %w", err)
			return
		}

		toolchainsSchema, err = compiler.Compile("toolchains.schema.json")
		if err != nil {
			compileErr = fmt.Errorf("compile toolchains schema: %w", err)
			return
		}
	})

	return compileErr
}

// ValidateConfig validates JSON data against the config schema.
func ValidateConfig(data []byte) error {
	if err := compileSchemas(); err != nil {
		return err
	}

	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if err := configSchema.Validate(v); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return nil
}

// ValidateToolchains validates JSON data against the toolchains schema.
func ValidateToolchains(data []byte) error {
	if err := compileSchemas(); err != nil {
		return err
	}

	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if err := toolchainsSchema.Validate(v); err != nil {
		return fmt.Errorf("toolchains validation failed: %w", err)
	}

	return nil
}
