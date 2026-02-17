package codegen

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestAnalyzeDeps_SingleRouteNoImports(t *testing.T) {
	root := t.TempDir()

	// routes/dashboard/index.tsx — no local imports
	writeFile(t, filepath.Join(root, "routes", "dashboard", "index.tsx"), `
import { serverData } from "@rstf/routes/dashboard";

export function View() {
  const data = serverData();
  return <div>{data.title}</div>;
}
`)
	// routes/dashboard/index.go — has server data
	writeFile(t, filepath.Join(root, "routes", "dashboard", "index.go"), `
package dashboard

type ServerData struct {
	Title string
}

func SSR() ServerData { return ServerData{} }
`)

	got, err := AnalyzeDeps(root, "routes/dashboard/index.tsx", nil)
	if err != nil {
		t.Fatalf("AnalyzeDeps: %v", err)
	}
	want := []string{"routes/dashboard"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAnalyzeDeps_SharedComponentWithGo(t *testing.T) {
	root := t.TempDir()

	// routes/dashboard/index.tsx — imports shared component
	writeFile(t, filepath.Join(root, "routes", "dashboard", "index.tsx"), `
import { serverData } from "@rstf/routes/dashboard";
import { View as UserAvatar } from "../../shared/ui/user-avatar";

export function View() {
  return <div><UserAvatar /></div>;
}
`)
	writeFile(t, filepath.Join(root, "routes", "dashboard", "index.go"), `
package dashboard
type ServerData struct { Title string }
func SSR() ServerData { return ServerData{} }
`)

	// shared/ui/user-avatar/index.tsx
	writeFile(t, filepath.Join(root, "shared", "ui", "user-avatar", "index.tsx"), `
import { serverData } from "@rstf/shared/ui/user-avatar";
export function View() { return <img />; }
`)
	// shared/ui/user-avatar/index.go — has server data
	writeFile(t, filepath.Join(root, "shared", "ui", "user-avatar", "index.go"), `
package useravatar
type ServerData struct { Name string }
func SSR() ServerData { return ServerData{} }
`)

	got, err := AnalyzeDeps(root, "routes/dashboard/index.tsx", nil)
	if err != nil {
		t.Fatalf("AnalyzeDeps: %v", err)
	}
	want := []string{"routes/dashboard", "shared/ui/user-avatar"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAnalyzeDeps_SharedComponentWithoutGo(t *testing.T) {
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "routes", "dashboard", "index.tsx"), `
import { Button } from "../../shared/ui/button";

export function View() {
  return <Button>Click</Button>;
}
`)
	writeFile(t, filepath.Join(root, "routes", "dashboard", "index.go"), `
package dashboard
type ServerData struct { Title string }
func SSR() ServerData { return ServerData{} }
`)

	// shared/ui/button.tsx — no .go file
	writeFile(t, filepath.Join(root, "shared", "ui", "button.tsx"), `
export function Button({ children }) { return <button>{children}</button>; }
`)

	got, err := AnalyzeDeps(root, "routes/dashboard/index.tsx", nil)
	if err != nil {
		t.Fatalf("AnalyzeDeps: %v", err)
	}
	want := []string{"routes/dashboard"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAnalyzeDeps_TransitiveImports(t *testing.T) {
	root := t.TempDir()

	// A imports B, B imports C (which has .go)
	writeFile(t, filepath.Join(root, "routes", "page", "index.tsx"), `
import { Wrapper } from "../../shared/wrapper/wrapper";
export function View() { return <Wrapper />; }
`)
	writeFile(t, filepath.Join(root, "routes", "page", "index.go"), `
package page
type ServerData struct {}
func SSR() ServerData { return ServerData{} }
`)

	// shared/wrapper — no .go, imports shared/deep
	writeFile(t, filepath.Join(root, "shared", "wrapper", "wrapper.tsx"), `
import { Deep } from "../deep/deep";
export function Wrapper() { return <Deep />; }
`)

	// shared/deep — has .go
	writeFile(t, filepath.Join(root, "shared", "deep", "deep.tsx"), `
export function Deep() { return <div>deep</div>; }
`)
	writeFile(t, filepath.Join(root, "shared", "deep", "deep.go"), `
package deep
type ServerData struct { Value string }
func SSR() ServerData { return ServerData{} }
`)

	got, err := AnalyzeDeps(root, "routes/page/index.tsx", nil)
	if err != nil {
		t.Fatalf("AnalyzeDeps: %v", err)
	}
	want := []string{"routes/page", "shared/deep"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAnalyzeDeps_CycleDetection(t *testing.T) {
	root := t.TempDir()

	// A imports B, B imports A — should not infinite loop
	writeFile(t, filepath.Join(root, "routes", "cycle", "index.tsx"), `
import { Other } from "../../shared/other/other";
export function View() { return <Other />; }
`)
	writeFile(t, filepath.Join(root, "routes", "cycle", "index.go"), `
package cycle
type ServerData struct {}
func SSR() ServerData { return ServerData{} }
`)

	writeFile(t, filepath.Join(root, "shared", "other", "other.tsx"), `
import { View } from "../../routes/cycle/index";
export function Other() { return <View />; }
`)

	got, err := AnalyzeDeps(root, "routes/cycle/index.tsx", nil)
	if err != nil {
		t.Fatalf("AnalyzeDeps: %v", err)
	}
	want := []string{"routes/cycle"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAnalyzeDeps_BareSpecifiersSkipped(t *testing.T) {
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "routes", "page", "index.tsx"), `
import { useState } from "react";
import { serverData } from "@rstf/routes/page";
import type { ReactNode } from "react";

export function View() { return <div />; }
`)
	writeFile(t, filepath.Join(root, "routes", "page", "index.go"), `
package page
type ServerData struct {}
func SSR() ServerData { return ServerData{} }
`)

	got, err := AnalyzeDeps(root, "routes/page/index.tsx", nil)
	if err != nil {
		t.Fatalf("AnalyzeDeps: %v", err)
	}
	want := []string{"routes/page"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAnalyzeDeps_RouteWithoutGoFile(t *testing.T) {
	root := t.TempDir()

	// Route with .tsx but no .go — entry dir should NOT be included
	writeFile(t, filepath.Join(root, "routes", "about", "index.tsx"), `
export function View() { return <div>About</div>; }
`)

	got, err := AnalyzeDeps(root, "routes/about/index.tsx", nil)
	if err != nil {
		t.Fatalf("AnalyzeDeps: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty deps, got %v", got)
	}
}

func TestAnalyzeDeps_IndexTsxResolution(t *testing.T) {
	root := t.TempDir()

	// Import resolves to a directory's index.tsx
	writeFile(t, filepath.Join(root, "routes", "page", "index.tsx"), `
import { Card } from "../../shared/card";
export function View() { return <Card />; }
`)

	// shared/card/index.tsx (directory import resolution)
	writeFile(t, filepath.Join(root, "shared", "card", "index.tsx"), `
export function Card() { return <div>card</div>; }
`)
	writeFile(t, filepath.Join(root, "shared", "card", "card.go"), `
package card
type ServerData struct {}
func SSR() ServerData { return ServerData{} }
`)

	got, err := AnalyzeDeps(root, "routes/page/index.tsx", nil)
	if err != nil {
		t.Fatalf("AnalyzeDeps: %v", err)
	}
	want := []string{"shared/card"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtractLocalImports(t *testing.T) {
	content := []byte(`
import { useState } from "react";
import { serverData } from "@rstf/routes/dashboard";
import { View as UserAvatar } from "../../shared/ui/user-avatar";
import { Button } from "./Button";
import type { ReactNode } from "react";
import { helper } from "../utils/helper";
`)

	got := extractLocalImports(content)
	sort.Strings(got)
	want := []string{
		"../../shared/ui/user-avatar",
		"../utils/helper",
		"./Button",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestResolveImportPath(t *testing.T) {
	root := t.TempDir()

	// Create files for resolution
	writeFile(t, filepath.Join(root, "button.tsx"), `export function Button() {}`)
	writeFile(t, filepath.Join(root, "card", "index.tsx"), `export function Card() {}`)

	tests := []struct {
		specifier string
		wantBase  string // expected filename (basename), "" if not found
	}{
		{"./button", "button.tsx"},           // direct .tsx
		{"./card", "index.tsx"},              // directory index.tsx
		{"./nonexistent", ""},                // not found
	}

	for _, tt := range tests {
		got := resolveImportPath(root, tt.specifier)
		if tt.wantBase == "" {
			if got != "" {
				t.Errorf("resolveImportPath(%q) = %q, want empty", tt.specifier, got)
			}
			continue
		}
		if filepath.Base(got) != tt.wantBase {
			t.Errorf("resolveImportPath(%q) = %q, want basename %q", tt.specifier, got, tt.wantBase)
		}
	}
}

func TestDirHasGoFile(t *testing.T) {
	root := t.TempDir()

	withGo := filepath.Join(root, "with-go")
	writeFile(t, filepath.Join(withGo, "index.go"), `package foo`)

	withoutGo := filepath.Join(root, "without-go")
	writeFile(t, filepath.Join(withoutGo, "index.tsx"), `export function View() {}`)

	if !dirHasGoFile(withGo) {
		t.Error("expected dirHasGoFile=true for directory with .go file")
	}
	if dirHasGoFile(withoutGo) {
		t.Error("expected dirHasGoFile=false for directory without .go file")
	}
}
