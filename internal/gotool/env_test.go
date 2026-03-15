package gotool

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDarwinEnvUsesWrapperAndCompilerOverride(t *testing.T) {
	env := darwinEnv([]string{"PATH=/usr/bin"}, "/tmp/rstf-cxx-wrapper")
	assert.Equal(t, []string{
		"PATH=/usr/bin",
		"RSTF_REAL_CXX=clang++",
		"CXX=/tmp/rstf-cxx-wrapper",
	}, env)
}

func TestDarwinEnvPreservesExistingCompiler(t *testing.T) {
	env := darwinEnv([]string{"CXX=/opt/homebrew/bin/clang++"}, "/tmp/rstf-cxx-wrapper")
	assert.Equal(t, []string{
		"CXX=/tmp/rstf-cxx-wrapper",
		"RSTF_REAL_CXX=/opt/homebrew/bin/clang++",
	}, env)
}

func TestDarwinEnvLeavesEmptyWrapperUntouched(t *testing.T) {
	env := darwinEnv([]string{"PATH=/usr/bin"}, "")
	assert.Equal(t, []string{"PATH=/usr/bin"}, env)
}

func TestEnvLeavesNonDarwinUntouched(t *testing.T) {
	env := Env([]string{"PATH=/usr/bin"}, "linux")
	assert.Equal(t, []string{"PATH=/usr/bin"}, env)
}
