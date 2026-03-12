package release

import (
	"os"
	"strings"
)

const (
	Version          = "0.1.0-alpha.3"
	ModuleVersion    = "v0.1.0-alpha.3"
	FrameworkModule  = "github.com/rafbgarcia/rstf"
	CLIPackage       = "@rstf/cli"
	CreatePackage    = "create-rstf"
	GoVersion        = "1.24.6"
	NodeVersion      = "24"
	DefaultCLIRef    = Version
	DefaultModuleRef = ModuleVersion
)

type ScaffoldConfig struct {
	FrameworkModule  string
	FrameworkRef     string
	FrameworkReplace string
	CLIPackage       string
	CLIRef           string
}

func CurrentScaffoldConfig() ScaffoldConfig {
	cfg := ScaffoldConfig{
		FrameworkModule:  FrameworkModule,
		FrameworkRef:     DefaultModuleRef,
		CLIPackage:       CLIPackage,
		CLIRef:           DefaultCLIRef,
		FrameworkReplace: strings.TrimSpace(os.Getenv("RSTF_SCAFFOLD_FRAMEWORK_REPLACE")),
	}

	if ref := strings.TrimSpace(os.Getenv("RSTF_SCAFFOLD_FRAMEWORK_VERSION")); ref != "" {
		cfg.FrameworkRef = ref
	}
	if ref := strings.TrimSpace(os.Getenv("RSTF_SCAFFOLD_CLI_PACKAGE")); ref != "" {
		cfg.CLIRef = ref
	}

	return cfg
}
