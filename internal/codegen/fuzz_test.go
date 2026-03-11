package codegen

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sanitizeIdent produces a valid exported Go identifier from fuzzed input.
// Returns "" if no usable letters are found.
func sanitizeIdent(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || (b.Len() > 0 && unicode.IsDigit(r)) {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return ""
	}
	r := b.String()
	return strings.ToUpper(r[:1]) + r[1:]
}

// --- Pure function fuzzing (no filesystem, no Go parser) ---

func FuzzGoTypeToTS(f *testing.F) {
	f.Add("string", false)
	f.Add("int", false)
	f.Add("bool", true)
	f.Add("float64", false)
	f.Add("MyStruct", false)
	f.Add("MyStruct", true)
	f.Add("", false)
	f.Add("uint8", true)

	f.Fuzz(func(t *testing.T, goType string, isSlice bool) {
		ts := goTypeToTS(goType, isSlice)

		if isSlice && goType != "" {
			assert.True(t, strings.HasSuffix(ts, "[]"), "goTypeToTS(%q, true) = %q, missing [] suffix", goType, ts)
		}
		if !isSlice {
			assert.False(t, strings.HasSuffix(ts, "[]"), "goTypeToTS(%q, false) = %q, unexpected [] suffix", goType, ts)
		}

		// Known Go primitives must map to correct TS primitives.
		switch goType {
		case "string":
			assert.True(t, strings.HasPrefix(ts, "string"), "goTypeToTS(%q, %v) = %q, want string prefix", goType, isSlice, ts)
		case "int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64",
			"float32", "float64":
			assert.True(t, strings.HasPrefix(ts, "number"), "goTypeToTS(%q, %v) = %q, want number prefix", goType, isSlice, ts)
		case "bool":
			assert.True(t, strings.HasPrefix(ts, "boolean"), "goTypeToTS(%q, %v) = %q, want boolean prefix", goType, isSlice, ts)
		}
	})
}

func FuzzNamespace(f *testing.F) {
	f.Add(".")
	f.Add("dashboard")
	f.Add("routes/dashboard")
	f.Add("routes/users/_id")
	f.Add("auth/forgot-password")
	f.Add("shared/ui/user-avatar")
	f.Add("")
	f.Add("a/b/c/d/e")
	f.Add("foo.bar")

	f.Fuzz(func(t *testing.T, dir string) {
		ns := Namespace(dir)

		if dir == "." {
			assert.Equal(t, "Main", ns, "Namespace(\".\")")
		}
		assert.NotContains(t, ns, "/", "Namespace(%q) = %q contains /", dir, ns)
		assert.NotContains(t, ns, "-", "Namespace(%q) = %q contains -", dir, ns)
		// Namespace is used as a TypeScript identifier. It must only contain
		// characters valid in JS/TS identifiers.
		for _, r := range ns {
			assert.True(t, unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '$', "Namespace(%q) = %q contains invalid identifier char %q", dir, ns, string(r))
		}
	})
}

func FuzzJsonTagName(f *testing.F) {
	f.Add(`json:"name"`)
	f.Add(`json:"name,omitempty"`)
	f.Add(`json:"-"`)
	f.Add(`json:"camelCase" xml:"other"`)
	f.Add(``)
	f.Add(`json:"with spaces"`)
	f.Add(`json:""`)
	f.Add(`json:"a,b,c"`)
	f.Add(`notjson:"foo"`)

	f.Fuzz(func(t *testing.T, tag string) {
		// Construct an *ast.Field directly instead of embedding in Go source.
		// This exercises jsonTagName without the Go parser bottleneck.
		field := &ast.Field{
			Names: []*ast.Ident{{Name: "X"}},
			Type:  &ast.Ident{Name: "string"},
			Tag:   &ast.BasicLit{Kind: token.STRING, Value: "`" + tag + "`"},
		}

		name := jsonTagName(field)

		assert.NotContains(t, name, "\"", "jsonTagName returned %q containing quote", name)
		assert.NotContains(t, name, "`", "jsonTagName returned %q containing backtick", name)
		assert.NotContains(t, name, ",", "jsonTagName returned %q containing comma (options not stripped)", name)
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
			assert.True(t, strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../"), "specifier %q does not start with ./ or ../", s)
		}
	})
}

// --- Code generation fuzzing (no filesystem, no Go parser) ---

func FuzzGenerateDTS(f *testing.F) {
	f.Add("dashboard", "ServerData", "title", "string", "Post", "name", "string")
	f.Add(".", "Data", "count", "number", "", "", "")
	f.Add("auth/login", "Response", "token", "string", "Session", "id", "number")
	f.Add("routes/users/_id", "Profile", "email", "string", "Settings", "theme", "boolean")

	f.Fuzz(func(t *testing.T, dir, structName, field1JSON, field1Type, struct2Name, field2JSON, field2Type string) {
		rf := RouteFile{
			Dir:     dir,
			Package: "page",
			Funcs:   []RouteFunc{{Name: "SSR", ReturnType: structName}},
		}

		sd := StructDef{Name: structName}
		if field1JSON != "" && field1Type != "" {
			sd.Fields = append(sd.Fields, StructField{Name: "F1", JSONName: field1JSON, Type: field1Type})
		}
		rf.Structs = append(rf.Structs, sd)

		if struct2Name != "" && struct2Name != structName {
			sd2 := StructDef{Name: struct2Name}
			if field2JSON != "" && field2Type != "" {
				sd2.Fields = append(sd2.Fields, StructField{Name: "G1", JSONName: field2JSON, Type: field2Type})
			}
			rf.Structs = append(rf.Structs, sd2)
		}

		dts := GenerateDTS(rf)

		assert.Contains(t, dts, "declare namespace ", "GenerateDTS output missing 'declare namespace'")
		assert.Equal(t, strings.Count(dts, "{"), strings.Count(dts, "}"), "unbalanced braces: %d '{' vs %d '}'", strings.Count(dts, "{"), strings.Count(dts, "}"))

		ns := Namespace(rf.Dir)
		if ns != "" {
			assert.Contains(t, dts, "declare namespace "+ns, "missing namespace %q in output", ns)
		}

		// Every struct should produce an interface declaration.
		for _, s := range rf.Structs {
			if s.Name != "" {
				assert.Contains(t, dts, "interface "+s.Name, "missing interface for struct %q", s.Name)
			}
		}
	})
}

func FuzzGenerateRuntimeModule(f *testing.F) {
	f.Add("dashboard", "ServerData", "dashboard")
	f.Add(".", "Data", "main")
	f.Add("routes/users/_id", "Profile", "routes/users/_id")
	f.Add("auth/login", "Response", "auth/login")

	f.Fuzz(func(t *testing.T, dir, returnType, componentPath string) {
		rf := RouteFile{
			Dir:     dir,
			Package: "page",
			Funcs:   []RouteFunc{{Name: "SSR", ReturnType: returnType}},
			Structs: []StructDef{{
				Name:   returnType,
				Fields: []StructField{{Name: "F", JSONName: "f", Type: "string"}},
			}},
		}

		rtmod := GenerateRuntimeModule(rf, componentPath)
		if rtmod == "" {
			return
		}

		assert.Contains(t, rtmod, "export function serverData()", "missing serverData export")
		assert.Contains(t, rtmod, "__setServerData", "missing __setServerData")
		assert.Equal(t, strings.Count(rtmod, "{"), strings.Count(rtmod, "}"), "unbalanced braces: %d '{' vs %d '}'", strings.Count(rtmod, "{"), strings.Count(rtmod, "}"))
		// componentPath is used as the window lookup key.
		assert.Contains(t, rtmod, componentPath, "componentPath %q not found in output", componentPath)
	})
}

// --- End-to-end fuzzing (structured Go source through parser + codegen) ---

func FuzzParseAndGenerate(f *testing.F) {
	// Fuzz individual components that get assembled into valid Go source.
	// This gets past the parser on nearly every input, exercising codegen.
	f.Add("ServerData", "Title", "string", "title")
	f.Add("Data", "Count", "int", "count")
	f.Add("Post", "Published", "bool", "published")
	f.Add("Config", "FontSize", "int32", "fontSize")

	f.Fuzz(func(t *testing.T, structName, fieldName, fieldGoType, jsonTag string) {
		structName = sanitizeIdent(structName)
		fieldName = sanitizeIdent(fieldName)
		if structName == "" || fieldName == "" {
			return
		}
		if structName == fieldName {
			fieldName += "X"
		}
		if !isPrimitiveGoType(fieldGoType) {
			fieldGoType = "string"
		}

		// Sanitize json tag to not break Go syntax.
		jsonTag = strings.ReplaceAll(jsonTag, "`", "")
		jsonTag = strings.ReplaceAll(jsonTag, "\n", "")
		jsonTag = strings.ReplaceAll(jsonTag, "\"", "")

		var tag string
		if jsonTag != "" {
			tag = fmt.Sprintf(" `json:\"%s\"`", jsonTag)
		}

		src := fmt.Sprintf("package page\ntype %s struct {\n\t%s %s%s\n}\nfunc SSR() %s { return %s{} }\n",
			structName, fieldName, fieldGoType, tag, structName, structName)

		dir := t.TempDir()
		pkg := filepath.Join(dir, "route")
		require.NoError(t, os.MkdirAll(pkg, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(pkg, "route.go"), []byte(src), 0o644))

		routes, err := ParseDir(dir)
		if err != nil {
			return // Edge-case identifier collisions with Go keywords
		}

		for _, rf := range routes {
			for _, fn := range rf.Funcs {
				assert.Equal(t, "SSR", fn.Name, "unexpected func name %q", fn.Name)
				assert.Equal(t, structName, fn.ReturnType, "ReturnType = %q, want %q", fn.ReturnType, structName)
			}
			for _, sd := range rf.Structs {
				for _, field := range sd.Fields {
					assert.NotEmpty(t, field.JSONName, "empty JSONName in struct %s", sd.Name)
					assert.NotEmpty(t, field.Type, "empty Type in struct %s", sd.Name)
					assert.NotContains(t, field.JSONName, "\"", "JSONName %q contains quote", field.JSONName)
				}
			}

			// End-to-end through generators.
			dts := GenerateDTS(rf)
			assert.Contains(t, dts, "declare namespace ", "GenerateDTS missing 'declare namespace'")
			assert.Equal(t, strings.Count(dts, "{"), strings.Count(dts, "}"), "unbalanced braces in DTS")

			rtmod := GenerateRuntimeModule(rf, rf.Dir)
			if rtmod != "" {
				assert.Contains(t, rtmod, "export function serverData()", "missing serverData export")
			}
		}
	})
}
