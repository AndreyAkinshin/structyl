// Package config provides configuration loading and validation for config.json.
package config

// Config represents the complete config.json configuration.
type Config struct {
	Project       ProjectConfig              `json:"project"`
	Version       *VersionConfig             `json:"version,omitempty"`
	Targets       map[string]TargetConfig    `json:"targets,omitempty"`
	Toolchains    map[string]ToolchainConfig `json:"toolchains,omitempty"`
	Mise          *MiseConfig                `json:"mise,omitempty"`
	Tests         *TestsConfig               `json:"tests,omitempty"`
	Documentation *DocsConfig                `json:"documentation,omitempty"`
	Docker        *DockerConfig              `json:"docker,omitempty"`
	Release       *ReleaseConfig             `json:"release,omitempty"`
	CI            *CIConfig                  `json:"ci,omitempty"`
	Artifacts     *ArtifactsConfig           `json:"artifacts,omitempty"`
}

// ProjectConfig contains project metadata.
type ProjectConfig struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Homepage    string `json:"homepage,omitempty"`
	Repository  string `json:"repository,omitempty"`
	License     string `json:"license,omitempty"`
}

// VersionConfig configures version management.
type VersionConfig struct {
	Source string              `json:"source,omitempty"`
	Files  []VersionFileConfig `json:"files,omitempty"`
}

// VersionFileConfig defines a version file update rule.
type VersionFileConfig struct {
	Path       string `json:"path"`
	Pattern    string `json:"pattern"`
	Replace    string `json:"replace"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

// TargetConfig defines a build target (language or auxiliary).
type TargetConfig struct {
	Type             string                 `json:"type"`
	Title            string                 `json:"title"`
	Toolchain        string                 `json:"toolchain,omitempty"`
	ToolchainVersion string                 `json:"toolchain_version,omitempty"` // Override mise tool version
	Directory        string                 `json:"directory,omitempty"`
	Cwd              string                 `json:"cwd,omitempty"`
	Commands         map[string]interface{} `json:"commands,omitempty"`
	Vars             map[string]string      `json:"vars,omitempty"`
	Env              map[string]string      `json:"env,omitempty"`
	DependsOn        []string               `json:"depends_on,omitempty"`
	DemoPath         string                 `json:"demo_path,omitempty"`
}

// ToolchainConfig defines a custom toolchain.
type ToolchainConfig struct {
	Extends  string                 `json:"extends,omitempty"`
	Version  string                 `json:"version,omitempty"` // Mise tool version for this toolchain
	Commands map[string]interface{} `json:"commands,omitempty"`
}

// MiseConfig configures mise integration.
// Note: mise is always required; there is no way to disable it.
type MiseConfig struct {
	AutoGenerate *bool             `json:"auto_generate,omitempty"` // Regenerate mise.toml before each run (default: true)
	ExtraTools   map[string]string `json:"extra_tools,omitempty"`   // Additional mise tools to install
}

// TestsConfig configures the reference test system.
type TestsConfig struct {
	Directory  string            `json:"directory,omitempty"`
	Pattern    string            `json:"pattern,omitempty"`
	Comparison *ComparisonConfig `json:"comparison,omitempty"`
}

// ComparisonConfig defines test result comparison settings.
type ComparisonConfig struct {
	FloatTolerance float64 `json:"float_tolerance,omitempty"`
	ToleranceMode  string  `json:"tolerance_mode,omitempty"` // "relative", "absolute", or "ulp"
	ArrayOrder     string  `json:"array_order,omitempty"`    // "strict" or "unordered"
	NaNEqualsNaN   bool    `json:"nan_equals_nan,omitempty"`
}

// DocsConfig configures documentation generation.
type DocsConfig struct {
	ReadmeTemplate string   `json:"readme_template,omitempty"`
	Placeholders   []string `json:"placeholders,omitempty"`
}

// DockerConfig configures Docker integration.
type DockerConfig struct {
	ComposeFile string                        `json:"compose_file,omitempty"`
	EnvVar      string                        `json:"env_var,omitempty"`
	Services    map[string]ServiceConfig      `json:"services,omitempty"`
	Targets     map[string]DockerTargetConfig `json:"targets,omitempty"`
}

// ServiceConfig defines Docker service settings.
type ServiceConfig struct {
	BaseImage string `json:"base_image,omitempty"`
}

// DockerTargetConfig defines Docker settings for a specific target.
type DockerTargetConfig struct {
	Platform    string            `json:"platform,omitempty"`
	CacheVolume string            `json:"cache_volume,omitempty"`
	Entrypoint  string            `json:"entrypoint,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

// ReleaseConfig configures the release workflow.
type ReleaseConfig struct {
	TagFormat   string   `json:"tag_format,omitempty"`
	ExtraTags   []string `json:"extra_tags,omitempty"`
	PreCommands []string `json:"pre_commands,omitempty"`
	Remote      string   `json:"remote,omitempty"`
	Branch      string   `json:"branch,omitempty"`
}

// CIConfig configures the CI pipeline.
type CIConfig struct {
	Steps []CIStep `json:"steps,omitempty"`
}

// CIStep defines a single step in the CI pipeline.
type CIStep struct {
	Name            string   `json:"name"`
	Target          string   `json:"target"`
	Command         string   `json:"command"`
	Flags           []string `json:"flags,omitempty"`
	DependsOn       []string `json:"depends_on,omitempty"`
	ContinueOnError bool     `json:"continue_on_error,omitempty"`
}

// ArtifactsConfig configures artifact collection.
type ArtifactsConfig struct {
	OutputDir string                    `json:"output_dir,omitempty"`
	Targets   map[string][]ArtifactSpec `json:"targets,omitempty"`
}

// ArtifactSpec defines an artifact to collect.
type ArtifactSpec struct {
	Source      string `json:"source"`
	Destination string `json:"destination,omitempty"`
	Rename      string `json:"rename,omitempty"`
}

// ToleranceMode represents how float comparison tolerance is applied.
type ToleranceMode string

const (
	// ToleranceModeRelative uses relative tolerance (percentage of expected value).
	ToleranceModeRelative ToleranceMode = "relative"
	// ToleranceModeAbsolute uses absolute tolerance.
	ToleranceModeAbsolute ToleranceMode = "absolute"
	// ToleranceModeULP uses ULP (Units in Last Place) tolerance for IEEE 754 precision.
	ToleranceModeULP ToleranceMode = "ulp"
)

// ArrayOrder represents how array elements are compared.
type ArrayOrder string

const (
	// ArrayOrderStrict requires elements to match in order.
	ArrayOrderStrict ArrayOrder = "strict"
	// ArrayOrderUnordered allows elements to match in any order (set comparison).
	ArrayOrderUnordered ArrayOrder = "unordered"
)
