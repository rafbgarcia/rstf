package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func FuzzParseGoSource(f *testing.F) {
	// Seed corpus: valid Go files with various SSR patterns.
	f.Add(`package page
type ServerData struct {
	Title string
}
func SSR() ServerData { return ServerData{} }
`)
	f.Add(`package dashboard
import "github.com/rafbgarcia/rstf"
type Post struct {
	Title     string  ` + "`json:\"title\"`" + `
	Published bool    ` + "`json:\"published\"`" + `
}
type ServerData struct {
	Posts []Post ` + "`json:\"posts\"`" + `
	Count int    ` + "`json:\"count\"`" + `
}
func SSR(ctx *rstf.Context) ServerData { return ServerData{} }
`)
	f.Add(`package settings
type Config struct {
	Theme    string
	FontSize int32
}
type ServerData struct {
	UserName string
	Config   Config
}
func SSR() ServerData { return ServerData{} }
`)
	f.Add(`package helpers
func DoSomething() string { return "hi" }
`)
	f.Add(`package api
func SSR() string { return "test" }
`)
	f.Add(`package nested
type Author struct {
	Name string
}
type Post struct {
	Title  string
	Author Author
}
type ServerData struct {
	Posts []*Post
}
func SSR() ServerData { return ServerData{} }
`)

	f.Fuzz(func(t *testing.T, src string) {
		dir := t.TempDir()
		pkg := filepath.Join(dir, "route")
		if err := os.MkdirAll(pkg, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pkg, "route.go"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}

		routes, err := ParseDir(dir)
		if err != nil {
			return // Parse errors on invalid Go are expected
		}

		for _, rf := range routes {
			for _, fn := range rf.Funcs {
				if fn.Name == "" {
					t.Error("RouteFunc.Name is empty")
				}
				if fn.ReturnType == "" {
					t.Error("RouteFunc.ReturnType is empty")
				}
			}
			for _, sd := range rf.Structs {
				for _, field := range sd.Fields {
					if field.JSONName == "" {
						t.Errorf("StructField.JSONName is empty in struct %s", sd.Name)
					}
					if field.Type == "" {
						t.Errorf("StructField.Type is empty in struct %s", sd.Name)
					}
				}
			}

			// End-to-end: verify generated TypeScript is structurally valid.
			dts := GenerateDTS(rf)
			if !strings.Contains(dts, "declare namespace ") {
				t.Error("GenerateDTS output missing 'declare namespace'")
			}
			ns := Namespace(rf.Dir)
			if ns != "" && !strings.Contains(dts, "declare namespace "+ns) {
				t.Errorf("GenerateDTS output missing namespace %q", ns)
			}
			if strings.Count(dts, "{") != strings.Count(dts, "}") {
				t.Errorf("GenerateDTS output has unbalanced braces: %d '{' vs %d '}'",
					strings.Count(dts, "{"), strings.Count(dts, "}"))
			}

			rtmod := GenerateRuntimeModule(rf, rf.Dir)
			if rtmod != "" {
				if !strings.Contains(rtmod, "export function serverData()") {
					t.Error("GenerateRuntimeModule missing serverData export")
				}
				if strings.Count(rtmod, "{") != strings.Count(rtmod, "}") {
					t.Errorf("GenerateRuntimeModule output has unbalanced braces: %d '{' vs %d '}'",
						strings.Count(rtmod, "{"), strings.Count(rtmod, "}"))
				}
			}
		}
	})
}

func FuzzExtractLocalImports(f *testing.F) {
	f.Add([]byte(`
import { useState } from "react";
import { serverData } from "@rstf/routes/dashboard";
import { View as UserAvatar } from "../../shared/ui/user-avatar";
import { Button } from "./Button";
import type { ReactNode } from "react";
import { helper } from "../utils/helper";
`))
	f.Add([]byte(`import { Foo } from "./foo";`))
	f.Add([]byte(`import { Foo } from '../bar';`))
	f.Add([]byte(`export function View() { return <div />; }`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, content []byte) {
		specifiers := extractLocalImports(content)
		for _, s := range specifiers {
			if !strings.HasPrefix(s, "./") && !strings.HasPrefix(s, "../") {
				t.Errorf("specifier %q does not start with ./ or ../", s)
			}
		}
	})
}

func FuzzNamespace(f *testing.F) {
	f.Add(".")
	f.Add("dashboard")
	f.Add("routes/dashboard")
	f.Add("routes/users/$id")
	f.Add("auth/forgot-password")
	f.Add("shared/ui/user-avatar")
	f.Add("")
	f.Add("a/b/c/d/e")

	f.Fuzz(func(t *testing.T, dir string) {
		ns := Namespace(dir)

		// "." maps to "Main"; everything else should produce non-empty output
		// unless the input is entirely empty/slashes.
		if dir == "." && ns != "Main" {
			t.Errorf("Namespace(\".\") = %q, want \"Main\"", ns)
		}
		if strings.Contains(ns, "/") {
			t.Errorf("Namespace(%q) = %q contains /", dir, ns)
		}
		if strings.Contains(ns, "-") {
			t.Errorf("Namespace(%q) = %q contains -", dir, ns)
		}
	})
}

func FuzzJsonTagParsing(f *testing.F) {
	f.Add(`json:"name"`)
	f.Add(`json:"name,omitempty"`)
	f.Add(`json:"-"`)
	f.Add(`json:"camelCase" xml:"other"`)
	f.Add(``)
	f.Add(`json:"with spaces"`)
	f.Add(`json:""`)

	f.Fuzz(func(t *testing.T, tag string) {
		// Embed the fuzzed tag in a minimal Go source file.
		src := "package p\ntype S struct {\n\tF string `" + tag + "`\n}\nfunc SSR() S { return S{} }\n"

		dir := t.TempDir()
		pkg := filepath.Join(dir, "route")
		if err := os.MkdirAll(pkg, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pkg, "route.go"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}

		routes, err := ParseDir(dir)
		if err != nil {
			return // Malformed tags can cause parse errors
		}

		for _, rf := range routes {
			for _, sd := range rf.Structs {
				for _, field := range sd.Fields {
					if strings.Contains(field.JSONName, "\"") {
						t.Errorf("JSONName %q contains quote", field.JSONName)
					}
					if strings.Contains(field.JSONName, "`") {
						t.Errorf("JSONName %q contains backtick", field.JSONName)
					}
				}
			}
		}
	})
}
