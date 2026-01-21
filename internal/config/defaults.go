package config

const (
	DefaultVersionSource     = ".structyl/PROJECT_VERSION"
	DefaultTestsDirectory    = "tests"
	DefaultTestsPattern      = "**/*.json"
	DefaultFloatTolerance    = 1e-9
	DefaultToleranceMode     = "relative"
	DefaultNaNEqualsNaN      = true
	DefaultDockerComposeFile = "docker-compose.yml"
	DefaultDockerEnvVar      = "STRUCTYL_DOCKER"
	DefaultMiseAutoGenerate  = true
)

// applyDefaults fills in default values for unset configuration fields.
func applyDefaults(cfg *Config) {
	applyVersionDefaults(cfg)
	applyTestsDefaults(cfg)
	applyTargetDefaults(cfg)
	applyDockerDefaults(cfg)
}

func applyVersionDefaults(cfg *Config) {
	if cfg.Version == nil {
		cfg.Version = &VersionConfig{}
	}
	if cfg.Version.Source == "" {
		cfg.Version.Source = DefaultVersionSource
	}
}

func applyTestsDefaults(cfg *Config) {
	if cfg.Tests == nil {
		cfg.Tests = &TestsConfig{}
	}
	if cfg.Tests.Directory == "" {
		cfg.Tests.Directory = DefaultTestsDirectory
	}
	if cfg.Tests.Pattern == "" {
		cfg.Tests.Pattern = DefaultTestsPattern
	}
	comparisonWasNil := cfg.Tests.Comparison == nil
	if comparisonWasNil {
		cfg.Tests.Comparison = &ComparisonConfig{}
	}
	if cfg.Tests.Comparison.FloatTolerance == nil {
		defaultTolerance := DefaultFloatTolerance
		cfg.Tests.Comparison.FloatTolerance = &defaultTolerance
	}
	if cfg.Tests.Comparison.ToleranceMode == "" {
		cfg.Tests.Comparison.ToleranceMode = DefaultToleranceMode
	}
	// NaNEqualsNaN defaults to true per schema. We only set this when the entire
	// comparison section was nil (user didn't provide any comparison config).
	// If user provided comparison config but omitted nan_equals_nan, they get false
	// (Go zero value) which differs from schema default—this is a known limitation
	// since we can't distinguish "not set" from "explicitly false" with a plain bool.
	if comparisonWasNil {
		cfg.Tests.Comparison.NaNEqualsNaN = DefaultNaNEqualsNaN
	}
}

func applyTargetDefaults(cfg *Config) {
	// Note: target is a copy (value semantics), so we must reassign to the map
	// after modification. This is intentional Go map iteration behavior.
	for name, target := range cfg.Targets {
		// Default directory is the target name
		if target.Directory == "" {
			target.Directory = name
		}
		// Default cwd is the directory
		if target.Cwd == "" {
			target.Cwd = target.Directory
		}
		cfg.Targets[name] = target
	}
}

func applyDockerDefaults(cfg *Config) {
	if cfg.Docker == nil {
		return // Docker is optional
	}
	if cfg.Docker.ComposeFile == "" {
		cfg.Docker.ComposeFile = DefaultDockerComposeFile
	}
	if cfg.Docker.EnvVar == "" {
		cfg.Docker.EnvVar = DefaultDockerEnvVar
	}
}
