package integration_test

import (
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	testProjectRootOnce sync.Once
	testProjectRootDir  string
	testProjectRootErr  error
)

func testProjectRoot() string {
	testProjectRootOnce.Do(func() {
		testProjectRootDir, testProjectRootErr = cloneTestProject(testProjectFixtureRoot())
	})
	if testProjectRootErr != nil {
		panic(fmt.Sprintf("cloning test project fixture: %v", testProjectRootErr))
	}
	return testProjectRootDir
}

func testProjectFixtureRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "test_project")
}

func cloneTestProject(src string) (string, error) {
	dst, err := os.MkdirTemp(filepath.Dir(src), ".tmp-rstf-integration-*")
	if err != nil {
		return "", err
	}
	if err := copyDir(src, dst); err != nil {
		_ = os.RemoveAll(dst)
		return "", err
	}
	return dst, nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if d.Type()&fs.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(linkTarget, target)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func TestMain(m *testing.M) {
	code := m.Run()
	if testProjectRootDir != "" {
		_ = os.RemoveAll(testProjectRootDir)
	}
	os.Exit(code)
}

// freePort finds an available TCP port by binding to :0 then closing.
func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "finding free port")
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return fmt.Sprintf("%d", port)
}

// waitForServer polls a URL until it returns a 200 response or the timeout expires.
func waitForServer(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	require.FailNowf(t, "server not ready", "server at %s not ready after %s", url, timeout)
}

func stopProcessGroup(t *testing.T, cmd *exec.Cmd, grace time.Duration) {
	t.Helper()
	if cmd == nil || cmd.Process == nil {
		return
	}

	pid := cmd.Process.Pid
	_ = syscall.Kill(-pid, syscall.SIGINT)

	waitCh := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		return
	case <-time.After(grace):
	}

	_ = syscall.Kill(-pid, syscall.SIGKILL)

	select {
	case <-waitCh:
	case <-time.After(1 * time.Second):
		t.Logf("process group %d did not exit after SIGKILL", pid)
	}
}
