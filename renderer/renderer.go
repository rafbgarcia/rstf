package renderer

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed ssr.ts
var ssrRuntimeSource []byte

type Renderer struct {
	port int
	cmd  *exec.Cmd
}

func New() *Renderer {
	return &Renderer{}
}

// Start spawns the Node sidecar process and waits for it to report its port.
func (r *Renderer) Start(projectRoot string) error {
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("renderer: resolve project root: %w", err)
	}

	ssrPath, err := ensureProjectRuntime(absRoot)
	if err != nil {
		return err
	}
	loaderPath, err := resolveProjectTSXLoader(absRoot)
	if err != nil {
		return err
	}

	r.cmd = exec.Command("node", "--import", loaderPath, ssrPath, "--project-root", absRoot)
	r.cmd.Dir = absRoot
	r.cmd.Env = append(os.Environ(), "NO_COLOR=1")
	r.cmd.Stderr = os.Stderr

	stdout, err := r.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("renderer: stdout pipe: %w", err)
	}

	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("renderer: start node: %w", err)
	}

	// Read the port from the first line of stdout with a timeout.
	portCh := make(chan int, 1)
	errCh := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			port, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
			if err != nil {
				errCh <- fmt.Errorf("renderer: invalid port %q: %w", scanner.Text(), err)
				return
			}
			portCh <- port
		} else {
			errCh <- fmt.Errorf("renderer: sidecar closed stdout without printing port")
		}
	}()

	select {
	case port := <-portCh:
		r.port = port
		// Write port to file so the CLI watcher can invalidate the sidecar cache.
		os.WriteFile(filepath.Join(absRoot, "rstf", "sidecar.port"), []byte(strconv.Itoa(port)), 0644)
	case err := <-errCh:
		r.cmd.Process.Kill()
		return err
	case <-time.After(10 * time.Second):
		r.cmd.Process.Kill()
		return fmt.Errorf("renderer: timed out waiting for sidecar port")
	}

	return nil
}

func ensureProjectRuntime(projectRoot string) (string, error) {
	runtimeDir := filepath.Join(projectRoot, "rstf", "runtime")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return "", fmt.Errorf("renderer: create runtime dir: %w", err)
	}

	ssrPath := filepath.Join(runtimeDir, "ssr.ts")
	if err := os.WriteFile(ssrPath, ssrRuntimeSource, 0644); err != nil {
		return "", fmt.Errorf("renderer: write ssr runtime: %w", err)
	}

	return ssrPath, nil
}

func resolveProjectTSXLoader(projectRoot string) (string, error) {
	loaderPath := filepath.Join(projectRoot, "node_modules", "tsx", "dist", "loader.mjs")
	if _, err := os.Stat(loaderPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf(
				"renderer: missing app runtime dependency \"tsx\" at %s; run npm install in %s",
				loaderPath,
				projectRoot,
			)
		}
		return "", fmt.Errorf("renderer: stat tsx loader: %w", err)
	}
	return loaderPath, nil
}

// Stop sends SIGINT to the sidecar and waits for it to exit.
func (r *Renderer) Stop() error {
	if r.cmd == nil || r.cmd.Process == nil {
		return nil
	}

	if err := r.cmd.Process.Signal(os.Interrupt); err != nil {
		// Process may have already exited.
		return nil
	}

	var once sync.Once
	waitCh := make(chan struct{})
	go func() {
		once.Do(func() { _ = r.cmd.Wait() })
		close(waitCh)
	}()

	select {
	case <-waitCh:
		return nil
	case <-time.After(2 * time.Second):
		// Fall back to force-kill to avoid hanging test/process shutdown.
		_ = r.cmd.Process.Kill()
	}

	select {
	case <-waitCh:
	case <-time.After(2 * time.Second):
	}

	return nil
}

// RenderRequest describes what to render: a route component inside a layout,
// with request-scoped SSR props keyed by component path.
type RenderRequest struct {
	Component string                    `json:"component"`
	Layout    string                    `json:"layout"`
	SSRProps  map[string]map[string]any `json:"ssrProps,omitempty"`
}

type renderResponse struct {
	HTML  string `json:"html"`
	Error string `json:"error"`
}

// Render sends a render request to the sidecar and returns the HTML string.
func (r *Renderer) Render(req RenderRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("renderer: marshal request: %w", err)
	}

	url := fmt.Sprintf("http://localhost:%d/render", r.port)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("renderer: POST /render: %w", err)
	}
	defer resp.Body.Close()

	var result renderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("renderer: decode response: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("renderer: %s", result.Error)
	}

	return result.HTML, nil
}
