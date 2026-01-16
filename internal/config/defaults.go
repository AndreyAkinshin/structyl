package config

// Default configuration values.
const (
	DefaultVersionSource     = "VERSION"
	DefaultTestsDirectory    = "tests"
	DefaultTestsPattern      = "**/*.json"
	DefaultFloatTolerance    = 1e-9
	DefaultToleranceMode     = "relative"
	DefaultDockerComposeFile = "docker-compose.yml"
	DefaultDockerEnvVar      = "STRUCTYL_DOCKER"
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
	if cfg.Tests.Comparison == nil {
		cfg.Tests.Comparison = &ComparisonConfig{}
	}
	if cfg.Tests.Comparison.FloatTolerance == 0 {
		cfg.Tests.Comparison.FloatTolerance = DefaultFloatTolerance
	}
	if cfg.Tests.Comparison.ToleranceMode == "" {
		cfg.Tests.Comparison.ToleranceMode = DefaultToleranceMode
	}
}

func applyTargetDefaults(cfg *Config) {
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
