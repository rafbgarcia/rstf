package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rafbgarcia/rstf/internal/codegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodegen(t *testing.T) {
	root := testProjectRoot()

	result, err := codegen.Generate(root)
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(filepath.Join(root, ".rstf")) })

	assert.Equal(t, 9, result.RouteCount)

	// Verify generated files exist.
	expectedFiles := []string{
		".rstf/server_gen.go",
		".rstf/generated/client.ts",
		".rstf/generated/routes.ts",
		".rstf/routes/routes_gen.go",
		".rstf/types/main.d.ts",
		".rstf/types/get-vs-ssr.d.ts",
		".rstf/types/live-chat-id.d.ts",
		".rstf/types/shared-ui-user-avatar.d.ts",
		".rstf/types/users-id.d.ts",
		".rstf/generated/main.ts",
		".rstf/generated/routes/get-vs-ssr.ts",
		".rstf/generated/routes/live-chat.$id.ts",
		".rstf/generated/shared/ui/user-avatar.ts",
		".rstf/entries/get-vs-ssr.entry.tsx",
		".rstf/entries/live-chat-id.entry.tsx",
		".rstf/entries/no-server.entry.tsx",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(root, f)
		_, err := os.Stat(path)
		assert.Falsef(t, os.IsNotExist(err), "expected file %s to exist", f)
	}

	// Verify DTS content.
	mainDTS, err := os.ReadFile(filepath.Join(root, ".rstf/types/main.d.ts"))
	require.NoError(t, err)
	assert.Contains(t, string(mainDTS), "declare namespace Main")
	assert.Contains(t, string(mainDTS), "appName: string")

	dashDTS, err := os.ReadFile(filepath.Join(root, ".rstf/types/get-vs-ssr.d.ts"))
	require.NoError(t, err)
	assert.Contains(t, string(dashDTS), "declare namespace RoutesGetVsSsr")
	assert.Contains(t, string(dashDTS), "posts: Post[]")

	avatarDTS, err := os.ReadFile(filepath.Join(root, ".rstf/types/shared-ui-user-avatar.d.ts"))
	require.NoError(t, err)
	assert.Contains(t, string(avatarDTS), "declare namespace SharedUiUserAvatar")
	assert.Contains(t, string(avatarDTS), "name: string")
	assert.Contains(t, string(avatarDTS), "status: string")

	// Verify runtime module has dual-mode initialization.
	dashMod, err := os.ReadFile(filepath.Join(root, ".rstf/generated/routes/get-vs-ssr.ts"))
	require.NoError(t, err)
	dashModStr := string(dashMod)
	assert.Contains(t, dashModStr, `typeof window !== "undefined"`)
	assert.Contains(t, dashModStr, `__RSTF_SERVER_DATA__["routes/get-vs-ssr"]`)

	avatarMod, err := os.ReadFile(filepath.Join(root, ".rstf/generated/shared/ui/user-avatar.ts"))
	require.NoError(t, err)
	avatarModStr := string(avatarMod)
	assert.Contains(t, avatarModStr, `typeof window !== "undefined"`)
	assert.Contains(t, avatarModStr, `__RSTF_SERVER_DATA__["shared/ui/user-avatar"]`)

	routesMod, err := os.ReadFile(filepath.Join(root, ".rstf/generated/routes.ts"))
	require.NoError(t, err)
	routesModStr := string(routesMod)
	assert.Contains(t, routesModStr, `export const routes = {`)
	assert.Contains(t, routesModStr, `"get-vs-ssr": {`)
	assert.Contains(t, routesModStr, `url(): string {`)
	assert.Contains(t, routesModStr, `import { defineAction, defineMutation, defineQuery, useAction, useMutation, useQuery } from "./client"`)
	assert.Contains(t, routesModStr, `"live-chat.$id": {`)
	assert.Contains(t, routesModStr, `GetMessages: defineQuery<{ id: string }, RoutesLiveChatId.GetMessagesResult>("live-chat.$id", "GetMessages")`)
	assert.Contains(t, routesModStr, `SendMessage: defineMutation<{ id: string }, RoutesLiveChatId.SendMessageInput, void>("live-chat.$id", "SendMessage")`)
	assert.Contains(t, routesModStr, `"users.$id": {`)
	assert.Contains(t, routesModStr, `url(params: { id: string }): string {`)
	assert.Contains(t, routesModStr, `return "/users/" + encodeURIComponent(params.id);`)
	assert.Contains(t, routesModStr, `export type RouteName = keyof typeof routes;`)

	goRoutes, err := os.ReadFile(filepath.Join(root, ".rstf/routes/routes_gen.go"))
	require.NoError(t, err)
	goRoutesStr := string(goRoutes)
	assert.Contains(t, goRoutesStr, "package routes")
	assert.Contains(t, goRoutesStr, "type UsersDotIdRoute struct{}")
	assert.Contains(t, goRoutesStr, `func (UsersDotIdRoute) Name() string { return "users.$id" }`)
	assert.Contains(t, goRoutesStr, "func (UsersDotIdRoute) URL(params UsersDotIdParams) Location {")
	assert.Contains(t, goRoutesStr, "var UsersDotId UsersDotIdRoute")
	assert.Contains(t, goRoutesStr, `return Location("/users/" + url.PathEscape(params.Id))`)
	assert.Contains(t, goRoutesStr, `var LiveChatDotIdGetMessages = QueryKey[LiveChatDotIdParams]{`)

	// Verify server_gen.go content.
	serverCode, err := os.ReadFile(filepath.Join(root, ".rstf/server_gen.go"))
	require.NoError(t, err)
	serverStr := string(serverCode)
	for _, expected := range []string{
		"package main",
		`app "github.com/rafbgarcia/rstf/tests/integration/test_project"`,
		`dashboard "github.com/rafbgarcia/rstf/tests/integration/test_project/routes/get-vs-ssr"`,
		`useravatar "github.com/rafbgarcia/rstf/tests/integration/test_project/shared/ui/user-avatar"`,
		`Component: "routes/get-vs-ssr"`,
		`Layout: "main"`,
		"structToMap(app.SSR(ctx))",
		"structToMap(dashboard.SSR(ctx))",
		`sd["shared/ui/user-avatar"] = structToMap(useravatar.SSR(ctx))`,
		"func assemblePage(",
		`rt.Handle("/.rstf/static/*"`,
		"assemblePage(html, sd,",
	} {
		assert.Contains(t, serverStr, expected, "server_gen.go missing %q", expected)
	}

	// Verify hydration entry content.
	entryContent, err := os.ReadFile(filepath.Join(root, ".rstf/entries/get-vs-ssr.entry.tsx"))
	require.NoError(t, err)
	entryStr := string(entryContent)
	for _, expected := range []string{
		`import { hydrateRoot } from "react-dom/client"`,
		`import { View as Layout } from "../../main"`,
		`import { View as Route } from "../../routes/get-vs-ssr"`,
		`import "@rstf/main"`,
		`import "@rstf/routes/get-vs-ssr"`,
		`import "@rstf/shared/ui/user-avatar"`,
		"hydrateRoot(document,",
	} {
		assert.Containsf(t, entryStr, expected, "get-vs-ssr.entry.tsx missing %q\n\nFull content:\n%s", expected, entryStr)
	}

	noServerEntry, err := os.ReadFile(filepath.Join(root, ".rstf/entries/no-server.entry.tsx"))
	require.NoError(t, err)
	noServerEntryStr := string(noServerEntry)
	for _, expected := range []string{
		`import { hydrateRoot } from "react-dom/client"`,
		`import { View as Layout } from "../../main"`,
		`import { View as Route } from "../../routes/no-server"`,
		`import "@rstf/main"`,
		"hydrateRoot(document,",
	} {
		assert.Containsf(t, noServerEntryStr, expected, "no-server.entry.tsx missing %q\n\nFull content:\n%s", expected, noServerEntryStr)
	}
	assert.NotContains(t, noServerEntryStr, `import "@rstf/routes/no-server"`)

	// Verify Entries map is populated.
	assert.Contains(t, result.Entries, "routes/get-vs-ssr")
	assert.Contains(t, result.Entries, "routes/live-chat.$id")
	assert.Contains(t, result.Entries, "routes/no-server")
}
