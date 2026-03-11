package renderer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"rogchap.com/v8go"
)

const bootstrapSource = `
globalThis.__RSTF_RENDERERS__ = globalThis.__RSTF_RENDERERS__ || {};

if (typeof MessageChannel === "undefined") {
  class RSTFMessagePort {
    constructor() {
      this.onmessage = null;
      this._target = null;
    }
    postMessage(message) {
      const target = this._target;
      if (!target || typeof target.onmessage !== "function") {
        return;
      }
      Promise.resolve().then(() => target.onmessage({ data: message }));
    }
    start() {}
    close() {}
  }

  globalThis.MessageChannel = class MessageChannel {
    constructor() {
      this.port1 = new RSTFMessagePort();
      this.port2 = new RSTFMessagePort();
      this.port1._target = this.port2;
      this.port2._target = this.port1;
    }
  };
}

if (typeof TextEncoder === "undefined") {
  globalThis.TextEncoder = class TextEncoder {
    encode(input = "") {
      const bytes = [];
      for (const char of String(input)) {
        const code = char.codePointAt(0);
        if (code <= 0x7f) {
          bytes.push(code);
        } else if (code <= 0x7ff) {
          bytes.push(0xc0 | (code >> 6));
          bytes.push(0x80 | (code & 0x3f));
        } else if (code <= 0xffff) {
          bytes.push(0xe0 | (code >> 12));
          bytes.push(0x80 | ((code >> 6) & 0x3f));
          bytes.push(0x80 | (code & 0x3f));
        } else {
          bytes.push(0xf0 | (code >> 18));
          bytes.push(0x80 | ((code >> 12) & 0x3f));
          bytes.push(0x80 | ((code >> 6) & 0x3f));
          bytes.push(0x80 | (code & 0x3f));
        }
      }
      return Uint8Array.from(bytes);
    }
  };
}
`

type Renderer struct {
	root string

	mu            sync.Mutex
	iso           *v8go.Isolate
	ctx           *v8go.Context
	loadedBundles map[string]time.Time
}

func New() *Renderer {
	return &Renderer{
		loadedBundles: map[string]time.Time{},
	}
}

// Start initializes the embedded V8 runtime and records the project root for
// lazy SSR bundle loading.
func (r *Renderer) Start(projectRoot string) error {
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("renderer: resolve project root: %w", err)
	}

	iso := v8go.NewIsolate()
	ctx := v8go.NewContext(iso)
	if _, err := ctx.RunScript(bootstrapSource, "bootstrap.js"); err != nil {
		ctx.Close()
		iso.Dispose()
		return fmt.Errorf("renderer: bootstrap runtime: %w", err)
	}

	r.root = absRoot
	r.iso = iso
	r.ctx = ctx
	r.loadedBundles = map[string]time.Time{}
	return nil
}

// Stop tears down the embedded V8 runtime.
func (r *Renderer) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ctx != nil {
		r.ctx.Close()
		r.ctx = nil
	}
	if r.iso != nil {
		r.iso.Dispose()
		r.iso = nil
	}
	r.loadedBundles = map[string]time.Time{}
	return nil
}

// RenderRequest describes what to render: a route component inside a layout,
// with request-scoped SSR props keyed by component path.
type RenderRequest struct {
	Component string                    `json:"component"`
	Layout    string                    `json:"layout"`
	SSRProps  map[string]map[string]any `json:"ssrProps,omitempty"`
}

// Render loads the route's SSR bundle into the embedded runtime and returns the
// rendered HTML string.
func (r *Renderer) Render(req RenderRequest) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ctx == nil || r.iso == nil {
		return "", fmt.Errorf("renderer: not started")
	}

	if err := r.ensureBundleLoaded(req.Component); err != nil {
		return "", err
	}

	global := r.ctx.Global()
	renderersValue, err := global.Get("__RSTF_RENDERERS__")
	if err != nil {
		return "", fmt.Errorf("renderer: read renderer registry: %w", err)
	}
	renderersObj, err := renderersValue.AsObject()
	if err != nil {
		return "", fmt.Errorf("renderer: renderer registry is not an object: %w", err)
	}
	renderFnValue, err := renderersObj.Get(req.Component)
	if err != nil {
		return "", fmt.Errorf("renderer: get renderer for %s: %w", req.Component, err)
	}
	if renderFnValue == nil || renderFnValue.IsUndefined() || renderFnValue.IsNull() {
		return "", fmt.Errorf("renderer: no SSR bundle registered for %s", req.Component)
	}
	renderFn, err := renderFnValue.AsFunction()
	if err != nil {
		return "", fmt.Errorf("renderer: renderer for %s is not callable: %w", req.Component, err)
	}

	payload, err := json.Marshal(req.SSRProps)
	if err != nil {
		return "", fmt.Errorf("renderer: marshal SSR props: %w", err)
	}
	arg, err := v8go.JSONParse(r.ctx, string(payload))
	if err != nil {
		return "", fmt.Errorf("renderer: parse SSR props JSON: %w", err)
	}

	result, err := renderFn.Call(v8go.Undefined(r.iso), arg)
	if err != nil {
		return "", fmt.Errorf("renderer: render %s: %w", req.Component, err)
	}
	r.ctx.PerformMicrotaskCheckpoint()
	return result.String(), nil
}

func (r *Renderer) ensureBundleLoaded(routeDir string) error {
	bundlePath := filepath.Join(r.root, routeSSRBundlePath(routeDir))
	info, err := os.Stat(bundlePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("renderer: missing SSR bundle %s", bundlePath)
		}
		return fmt.Errorf("renderer: stat SSR bundle %s: %w", bundlePath, err)
	}

	if loadedAt, ok := r.loadedBundles[routeDir]; ok {
		if info.ModTime().Equal(loadedAt) {
			return nil
		}
		// Bundle changed. Reset the context so dev reloads do not accumulate
		// stale component code in the isolate.
		if err := r.resetContext(); err != nil {
			return err
		}
	}

	source, err := os.ReadFile(bundlePath)
	if err != nil {
		return fmt.Errorf("renderer: read SSR bundle %s: %w", bundlePath, err)
	}
	if _, err := r.ctx.RunScript(string(source), bundlePath); err != nil {
		return fmt.Errorf("renderer: load SSR bundle %s: %w", bundlePath, err)
	}

	r.loadedBundles[routeDir] = info.ModTime()
	return nil
}

func (r *Renderer) resetContext() error {
	if r.ctx != nil {
		r.ctx.Close()
	}
	if r.iso != nil {
		r.iso.Dispose()
	}

	iso := v8go.NewIsolate()
	ctx := v8go.NewContext(iso)
	if _, err := ctx.RunScript(bootstrapSource, "bootstrap.js"); err != nil {
		ctx.Close()
		iso.Dispose()
		return fmt.Errorf("renderer: bootstrap runtime: %w", err)
	}

	r.iso = iso
	r.ctx = ctx
	r.loadedBundles = map[string]time.Time{}
	return nil
}

func routeSSRBundlePath(routeDir string) string {
	return filepath.ToSlash(filepath.Join("rstf", "ssr", routeArtifactName(strings.TrimPrefix(routeDir, "routes/"))+".js"))
}

func routeArtifactName(name string) string {
	segments := strings.FieldsFunc(name, func(r rune) bool {
		return r == '.' || r == '/'
	})
	if len(segments) == 0 {
		return name
	}

	for i, seg := range segments {
		if len(seg) > 1 && strings.HasPrefix(seg, "_") {
			segments[i] = seg[1:]
		}
	}
	return strings.Join(segments, "-")
}
