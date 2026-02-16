package renderer

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Renderer struct {
	port int
	cmd  *exec.Cmd
}

func New() *Renderer {
	return &Renderer{}
}

// Start spawns the Bun sidecar process and waits for it to report its port.
func (r *Renderer) Start(projectRoot string) error {
	ssrPath := filepath.Join(frameworkRoot(), "runtime", "ssr.ts")

	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("renderer: resolve project root: %w", err)
	}

	r.cmd = exec.Command("bun", "run", ssrPath, "--project-root", absRoot)
	r.cmd.Stderr = os.Stderr

	stdout, err := r.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("renderer: stdout pipe: %w", err)
	}

	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("renderer: start bun: %w", err)
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
		os.WriteFile(filepath.Join(absRoot, ".rstf", "sidecar.port"), []byte(strconv.Itoa(port)), 0644)
	case err := <-errCh:
		r.cmd.Process.Kill()
		return err
	case <-time.After(10 * time.Second):
		r.cmd.Process.Kill()
		return fmt.Errorf("renderer: timed out waiting for sidecar port")
	}

	return nil
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
	// Ignore the exit error — SIGINT causes a non-zero exit code which is expected.
	r.cmd.Wait()
	return nil
}

// RenderRequest describes what to render: a route component inside a layout,
// with server data provided to generated modules via ES module live bindings.
type RenderRequest struct {
	Component  string                    `json:"component"`
	Layout     string                    `json:"layout"`
	ServerData map[string]map[string]any `json:"serverData,omitempty"`
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

// frameworkRoot returns the root directory of the rstf framework module,
// derived from the location of this source file.
func frameworkRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	// filename is .../renderer/renderer.go
	// framework root is one directory up
	return filepath.Dir(filepath.Dir(filename))
}
