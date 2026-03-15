package gotool

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const (
	cxxKey                      = "CXX"
	realCXXKey                  = "RSTF_REAL_CXX"
	clangVLAExtensionSuppressor = "-Wno-vla-cxx-extension"
)

var (
	darwinCXXWrapperOnce sync.Once
	darwinCXXWrapperPath string
)

// Prepare applies environment tweaks needed for Go tool invocations that pull
// in native dependencies like v8go.
func Prepare(cmd *exec.Cmd) {
	cmd.Env = Env(cmd.Environ(), runtime.GOOS)
}

// Env returns a copy of env with platform-specific Go tool overrides applied.
func Env(env []string, goos string) []string {
	out := append([]string(nil), env...)
	if goos != "darwin" {
		return out
	}

	return darwinEnv(out, darwinCXXWrapper())
}

func darwinEnv(env []string, wrapperPath string) []string {
	if wrapperPath == "" {
		return env
	}

	realCXX := lookupEnv(env, cxxKey)
	if realCXX == "" || realCXX == wrapperPath {
		realCXX = "clang++"
	}

	env = upsertEnv(env, realCXXKey, realCXX)
	return upsertEnv(env, cxxKey, wrapperPath)
}

func upsertEnv(env []string, key string, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			continue
		}

		env[i] = prefix + value
		return env
	}

	return append(env, prefix+value)
}

func lookupEnv(env []string, key string) string {
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	return ""
}

func darwinCXXWrapper() string {
	darwinCXXWrapperOnce.Do(func() {
		file, err := os.CreateTemp("", "rstf-clang++-*")
		if err != nil {
			return
		}
		path := file.Name()
		script := "#!/bin/sh\nexec \"$" + realCXXKey + "\" \"$@\" " + clangVLAExtensionSuppressor + "\n"
		if _, err := file.WriteString(script); err != nil {
			_ = file.Close()
			_ = os.Remove(path)
			return
		}
		if err := file.Close(); err != nil {
			_ = os.Remove(path)
			return
		}
		if err := os.Chmod(path, 0o755); err != nil {
			_ = os.Remove(path)
			return
		}
		darwinCXXWrapperPath = filepath.Clean(path)
	})

	return darwinCXXWrapperPath
}
