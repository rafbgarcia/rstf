package scaffold

import (
	"fmt"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"github.com/rafbgarcia/rstf/internal/codegen"
	"github.com/rafbgarcia/rstf/internal/release"
)

type Config struct {
	Name             string
	DisplayName      string
	Module           string
	PackageName      string
	TargetDir        string
	FrameworkModule  string
	FrameworkVersion string
	FrameworkReplace string
	CLIPackage       string
	CLIRef           string
}

type Options struct {
	InstallDependencies bool
}

type fileTemplate struct {
	path     string
	contents string
}

var scaffoldTemplates = []fileTemplate{
	{path: ".gitignore", contents: gitignoreTemplate},
	{path: "go.mod", contents: goModTemplate},
	{path: "package.json", contents: packageJSONTemplate},
	{path: "tsconfig.json", contents: tsconfigTemplate},
	{path: "postcss.config.mjs", contents: postCSSConfigTemplate},
	{path: "main.css", contents: mainCSSTemplate},
	{path: "main.go", contents: mainGoTemplate},
	{path: "main.tsx", contents: mainTSXTemplate},
	{path: "routes/index/index.go", contents: indexGoTemplate},
	{path: "routes/index/index.tsx", contents: indexTSXTemplate},
	{path: "routes/live-chat._id/index.go", contents: liveChatGoTemplate},
	{path: "routes/live-chat._id/index.tsx", contents: liveChatTSXTemplate},
	{path: "routes/users._id/index.go", contents: userProfileGoTemplate},
	{path: "routes/users._id/index.tsx", contents: userProfileTSXTemplate},
	{path: "shared/ui/app-badge/index.go", contents: appBadgeGoTemplate},
	{path: "shared/ui/app-badge/index.tsx", contents: appBadgeTSXTemplate},
	{path: "README.md", contents: readmeTemplate},
}

func DeriveConfig(name string, module string) (Config, error) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Config{}, fmt.Errorf("app name is required")
	}

	targetDir, err := filepath.Abs(trimmedName)
	if err != nil {
		return Config{}, fmt.Errorf("resolving target dir: %w", err)
	}

	baseName := filepath.Base(targetDir)
	if baseName == "." || baseName == string(filepath.Separator) || baseName == "" {
		return Config{}, fmt.Errorf("invalid app name %q", name)
	}

	moduleName := strings.TrimSpace(module)
	if moduleName == "" {
		moduleName = baseName
	}

	packageName := sanitizePackageName(baseName)
	if packageName == "" {
		return Config{}, fmt.Errorf("could not derive a valid Go package name from %q", baseName)
	}

	releaseCfg := release.CurrentScaffoldConfig()

	return Config{
		Name:             baseName,
		DisplayName:      humanizeName(baseName),
		Module:           moduleName,
		PackageName:      packageName,
		TargetDir:        targetDir,
		FrameworkModule:  releaseCfg.FrameworkModule,
		FrameworkVersion: releaseCfg.FrameworkRef,
		FrameworkReplace: releaseCfg.FrameworkReplace,
		CLIPackage:       releaseCfg.CLIPackage,
		CLIRef:           releaseCfg.CLIRef,
	}, nil
}

func Create(cfg Config, opts Options) error {
	if err := validateTargetDir(cfg.TargetDir); err != nil {
		return err
	}

	if err := os.MkdirAll(cfg.TargetDir, 0755); err != nil {
		return fmt.Errorf("creating %s: %w", cfg.TargetDir, err)
	}

	for _, file := range scaffoldTemplates {
		if err := writeTemplateFile(cfg, file); err != nil {
			return err
		}
	}

	gen, err := codegen.NewGenerator(cfg.TargetDir)
	if err != nil {
		return fmt.Errorf("initializing codegen for scaffold: %w", err)
	}
	if _, err := gen.Generate(); err != nil {
		return fmt.Errorf("generating initial rstf artifacts: %w", err)
	}

	if !opts.InstallDependencies {
		return nil
	}

	if err := runCommand(cfg.TargetDir, "npm", "install"); err != nil {
		return fmt.Errorf("installing npm dependencies: %w", err)
	}
	if err := runCommand(cfg.TargetDir, "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("tidying Go module: %w", err)
	}

	return nil
}

func validateTargetDir(targetDir string) error {
	info, err := os.Stat(targetDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat %s: %w", targetDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s already exists and is not a directory", targetDir)
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", targetDir, err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("%s already exists and is not empty", targetDir)
	}
	return nil
}

func writeTemplateFile(cfg Config, file fileTemplate) error {
	targetPath := filepath.Join(cfg.TargetDir, file.path)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("creating parent dir for %s: %w", targetPath, err)
	}

	tmpl, err := template.New(file.path).Parse(file.contents)
	if err != nil {
		return fmt.Errorf("parsing template %s: %w", file.path, err)
	}

	f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("opening %s: %w", targetPath, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, cfg); err != nil {
		return fmt.Errorf("writing %s: %w", targetPath, err)
	}

	return nil
}

func runCommand(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func humanizeName(name string) string {
	replacer := strings.NewReplacer("-", " ", "_", " ", ".", " ")
	parts := strings.Fields(replacer.Replace(name))
	if len(parts) == 0 {
		return "rstf App"
	}
	for i, part := range parts {
		runes := []rune(strings.ToLower(part))
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}

func sanitizePackageName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}

	pkg := b.String()
	if pkg == "" {
		pkg = "app"
	}
	if unicode.IsDigit(rune(pkg[0])) {
		pkg = "app" + pkg
	}
	if !token.IsIdentifier(pkg) || token.Lookup(pkg).IsKeyword() {
		pkg = "app"
	}
	return pkg
}

const gitignoreTemplate = `rstf/
dist/
node_modules/
`

const goModTemplate = `module {{ .Module }}

go ` + release.GoVersion + `

require {{ .FrameworkModule }} {{ .FrameworkVersion }}

{{- if .FrameworkReplace }}
replace {{ .FrameworkModule }} => {{ .FrameworkReplace }}
{{- end }}
`

const packageJSONTemplate = `{
  "name": "{{ .Name }}",
  "private": true,
  "scripts": {
    "dev": "rstf dev",
    "build": "rstf build",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "react": "^19.1.0",
    "react-dom": "^19.1.0"
  },
  "devDependencies": {
    "{{ .CLIPackage }}": "{{ .CLIRef }}",
    "@tailwindcss/postcss": "^4.2.1",
    "@types/react": "^19.1.0",
    "@types/react-dom": "^19.1.0",
    "postcss": "^8.5.0",
    "tailwindcss": "^4.2.1",
    "typescript": "^5.9.3"
  }
}
`

const tsconfigTemplate = `{
  "compilerOptions": {
    "jsx": "react-jsx",
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "node",
    "lib": ["DOM", "DOM.Iterable", "ES2022"],
    "strict": true,
    "noEmit": true,
    "paths": {
      "@rstf/*": ["./rstf/generated/*"]
    }
  },
  "include": ["rstf/types", "rstf/generated/**/*.ts", "**/*.ts", "**/*.tsx"]
}
`

const postCSSConfigTemplate = `export default {
  plugins: {
    "@tailwindcss/postcss": {},
  },
};
`

const mainCSSTemplate = `@import "tailwindcss";

@theme {
  --font-sans: "Inter Tight", "Inter", ui-sans-serif, system-ui, sans-serif;
}

@layer base {
  :root {
    color-scheme: light;
  }

  body {
    @apply bg-stone-50 text-stone-900 antialiased;
    background-image:
      radial-gradient(circle at top left, rgba(251, 191, 36, 0.18), transparent 24rem),
      radial-gradient(circle at top right, rgba(56, 189, 248, 0.18), transparent 22rem),
      linear-gradient(180deg, rgba(255, 255, 255, 0.96), rgba(245, 245, 244, 0.96));
    min-height: 100vh;
  }

  a {
    @apply transition-colors duration-150;
  }

  ::selection {
    @apply bg-amber-200 text-stone-900;
  }
}

@layer components {
  .shell {
    @apply mx-auto min-h-screen max-w-6xl px-6 py-8 sm:px-8 lg:px-10;
  }

  .glass {
    @apply rounded-[2rem] border border-white/70 bg-white/85 shadow-[0_24px_80px_-32px_rgba(24,24,27,0.32)] backdrop-blur;
  }

  .chip {
    @apply inline-flex items-center gap-2 rounded-full border border-amber-300/70 bg-amber-100/80 px-3 py-1 text-xs font-semibold uppercase tracking-[0.22em] text-amber-900;
  }

  .nav-link {
    @apply rounded-full px-4 py-2 text-sm font-medium text-stone-600 hover:bg-stone-900 hover:text-stone-50;
  }

  .feature-card {
    @apply rounded-[2rem] border border-white/70 bg-white/85 p-6 shadow-[0_24px_80px_-32px_rgba(24,24,27,0.32)] backdrop-blur;
  }
}
`

const mainGoTemplate = `package {{ .PackageName }}

import rstf "github.com/rafbgarcia/rstf"

type ServerData struct {
	AppName string ` + "`json:\"appName\"`" + `
	Tagline string ` + "`json:\"tagline\"`" + `
}

func SSR(ctx *rstf.Context) ServerData {
	return ServerData{
		AppName: "{{ .DisplayName }}",
		Tagline: "Go-first server rendering with React islands, live queries, and a clean local-to-production workflow.",
	}
}
`

const mainTSXTemplate = `import { SSR, type MainSSRProps } from "@rstf/main";

export const View = SSR(function View({ children, appName, tagline }: MainSSRProps) {

  return (
    <html lang="en">
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <title>{appName}</title>
        <link rel="stylesheet" href="/rstf/static/main.css" />
      </head>
      <body>
        <div className="shell">
          <div className="glass overflow-hidden">
            <div className="border-b border-stone-200/80 px-6 py-5 sm:px-8">
              <div className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
                <div className="max-w-2xl">
                  <span className="chip">rstf app</span>
                  <h1 className="mt-4 text-3xl font-semibold tracking-tight text-stone-950 sm:text-4xl">
                    {appName}
                  </h1>
                  <p className="mt-3 text-sm leading-6 text-stone-600 sm:text-base">{tagline}</p>
                </div>
                <nav className="flex flex-wrap gap-2">
                  <a className="nav-link" href="/">
                    Overview
                  </a>
                  <a className="nav-link" href="/live-chat/studio">
                    Live chat
                  </a>
                  <a className="nav-link" href="/users/ada">
                    Dynamic route
                  </a>
                </nav>
              </div>
            </div>
            <main className="px-6 py-8 sm:px-8">{children}</main>
          </div>
        </div>
      </body>
    </html>
  );
});
`

const indexGoTemplate = `package index

import rstf "github.com/rafbgarcia/rstf"

type Feature struct {
	Title       string ` + "`json:\"title\"`" + `
	Description string ` + "`json:\"description\"`" + `
	Href        string ` + "`json:\"href\"`" + `
}

type ServerData struct {
	Headline string    ` + "`json:\"headline\"`" + `
	Intro    string    ` + "`json:\"intro\"`" + `
	Features []Feature ` + "`json:\"features\"`" + `
}

type APIResponse struct {
	Route   string ` + "`json:\"route\"`" + `
	Message string ` + "`json:\"message\"`" + `
}

func SSR(ctx *rstf.Context) ServerData {
	return ServerData{
		Headline: "Ship server-rendered pages with interactive islands and typed server contracts.",
		Intro:    "This demo includes shared server data, a dynamic profile page, and a live query route with mutation and action examples.",
		Features: []Feature{
			{
				Title:       "Shared server data",
				Description: "The badge below is rendered from Go data in a shared component.",
				Href:        "/",
			},
			{
				Title:       "Live queries",
				Description: "Watch a room update in place with query invalidation.",
				Href:        "/live-chat/studio",
			},
			{
				Title:       "Dynamic routes",
				Description: "Route folders like users._id map cleanly to URL params.",
				Href:        "/users/ada",
			},
		},
	}
}

func GET(ctx *rstf.Context) error {
	return ctx.JSON(200, APIResponse{
		Route:   "/",
		Message: "This route serves HTML by default and JSON when requested explicitly.",
	})
}
`

const indexTSXTemplate = `import { SSR, type RoutesIndexSSRProps } from "@rstf/routes/index";
import { AppBadge } from "../../shared/ui/app-badge";

export const View = SSR(function View({ headline, intro, features }: RoutesIndexSSRProps) {

  return (
    <div className="space-y-8">
      <section className="grid gap-6 lg:grid-cols-[1.4fr_0.8fr]">
        <div className="feature-card">
          <span className="chip">Typed SSR</span>
          <h2 className="mt-5 max-w-2xl text-3xl font-semibold tracking-tight text-stone-950 sm:text-4xl">
            {headline}
          </h2>
          <p className="mt-4 max-w-2xl text-base leading-7 text-stone-600">{intro}</p>
          <div className="mt-8 flex flex-wrap gap-3">
            <a
              className="rounded-full bg-stone-950 px-5 py-3 text-sm font-semibold text-stone-50 hover:bg-stone-800"
              href="/live-chat/studio"
            >
              Open live demo
            </a>
            <a
              className="rounded-full border border-stone-300 px-5 py-3 text-sm font-semibold text-stone-700 hover:border-stone-900 hover:text-stone-900"
              href="/users/ada"
            >
              View dynamic route
            </a>
          </div>
        </div>

        <div className="feature-card bg-linear-to-br from-amber-50 to-white">
          <p className="text-sm font-medium uppercase tracking-[0.22em] text-amber-700">Shared component</p>
          <div className="mt-5">
            <AppBadge />
          </div>
          <p className="mt-5 text-sm leading-6 text-stone-600">
            The badge renders server data from <code>@rstf/shared/ui/app-badge</code>.
          </p>
        </div>
      </section>

      <section className="grid gap-4 md:grid-cols-3">
        {features.map((feature) => (
          <a key={feature.title} href={feature.href} className="feature-card block hover:-translate-y-0.5">
            <p className="text-lg font-semibold text-stone-950">{feature.title}</p>
            <p className="mt-3 text-sm leading-6 text-stone-600">{feature.description}</p>
          </a>
        ))}
      </section>
    </div>
  );
});
`

const liveChatGoTemplate = `package livechat

import (
	"strings"

	rstf "github.com/rafbgarcia/rstf"
	"{{ .Module }}/rstf/routes"
)

type Message struct {
	Body string ` + "`json:\"body\"`" + `
}

type GetMessagesResult struct {
	Messages []Message ` + "`json:\"messages\"`" + `
}

type SendMessageInput struct {
	Body string ` + "`json:\"body\"`" + `
}

type EchoActionResult struct {
	Value string ` + "`json:\"value\"`" + `
}

var chatRooms = map[string][]Message{
	"studio": {
		{Body: "Welcome to the studio room."},
		{Body: "Try a mutation below to invalidate the query."},
	},
}

func GetMessages(ctx *rstf.QueryContext) GetMessagesResult {
	roomID := ctx.Param("id")
	messages := append([]Message(nil), chatRooms[roomID]...)
	return GetMessagesResult{Messages: messages}
}

func SendMessage(ctx *rstf.MutationContext, input SendMessageInput) error {
	roomID := ctx.Param("id")
	body := strings.TrimSpace(input.Body)
	if body == "" {
		return rstf.ValidationError("message is required", map[string]any{
			"field": "body",
		})
	}

	chatRooms[roomID] = append(chatRooms[roomID], Message{Body: body})
	routes.LiveChatDotIdGetMessages.Invalidate(ctx, routes.LiveChatDotIdParams{Id: roomID})
	return nil
}

func EchoAction(ctx *rstf.ActionContext, input string) EchoActionResult {
	return EchoActionResult{Value: strings.ToUpper(strings.TrimSpace(input))}
}
`

const liveChatTSXTemplate = `import { useState } from "react";
import { routes, useAction, useMutation, useQuery } from "@rstf/routes";

const roomId = "studio";

export function View() {
  const [draft, setDraft] = useState("");
  const [command, setCommand] = useState("ship it");
  const [echoed, setEchoed] = useState("");

  const query = useQuery(routes["live-chat._id"].GetMessages, { id: roomId });
  const sendMessage = useMutation(routes["live-chat._id"].SendMessage, { id: roomId });
  const echoAction = useAction(routes["live-chat._id"].EchoAction, { id: roomId });

  return (
    <div className="grid gap-6 lg:grid-cols-[1.15fr_0.85fr]">
      <section className="feature-card">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <span className="chip">Live query</span>
            <h2 className="mt-4 text-2xl font-semibold tracking-tight text-stone-950">Studio room</h2>
          </div>
          <a className="text-sm font-medium text-sky-700 hover:text-sky-900" href="/">
            Back home
          </a>
        </div>

        {query.status === "loading" && <p className="mt-6 text-sm text-stone-500">Loading messages…</p>}
        {query.status === "error" && (
          <p className="mt-6 rounded-2xl bg-rose-50 px-4 py-3 text-sm text-rose-700">{query.error.message}</p>
        )}

        {query.status === "ready" && (
          <ul data-testid="messages-list" className="mt-6 space-y-3">
            {query.data.messages.map((message, index) => (
              <li
                key={message.body + index}
                className="rounded-2xl border border-stone-200 bg-stone-50 px-4 py-3 text-sm text-stone-700"
              >
                {message.body}
              </li>
            ))}
          </ul>
        )}

        <div className="mt-6 flex flex-col gap-3 sm:flex-row">
          <input
            data-testid="chat-input"
            className="min-w-0 flex-1 rounded-full border border-stone-300 bg-white px-4 py-3 text-sm outline-none ring-0 placeholder:text-stone-400 focus:border-sky-500"
            placeholder="Post a fresh message"
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
          />
          <button
            data-testid="send-message"
            className="rounded-full bg-stone-950 px-5 py-3 text-sm font-semibold text-stone-50 hover:bg-stone-800"
            onClick={async () => {
              await sendMessage({ body: draft });
              setDraft("");
            }}
          >
            Send mutation
          </button>
        </div>
      </section>

      <aside className="feature-card bg-linear-to-br from-sky-50 to-white">
        <span className="chip">Action</span>
        <h3 className="mt-4 text-xl font-semibold tracking-tight text-stone-950">Command preview</h3>
        <p className="mt-3 text-sm leading-6 text-stone-600">
          Actions use the same typed route contract without creating a persistent subscription.
        </p>
        <input
          className="mt-6 w-full rounded-3xl border border-stone-300 bg-white px-4 py-3 text-sm outline-none placeholder:text-stone-400 focus:border-sky-500"
          value={command}
          onChange={(event) => setCommand(event.target.value)}
        />
        <button
          className="mt-4 rounded-full border border-stone-300 px-5 py-3 text-sm font-semibold text-stone-700 hover:border-stone-900 hover:text-stone-900"
          onClick={async () => {
            const result = await echoAction(command);
            setEchoed(result.value);
          }}
        >
          Run action
        </button>
        <div className="mt-6 rounded-3xl border border-dashed border-stone-300 bg-white/80 p-4">
          <p className="text-xs font-semibold uppercase tracking-[0.22em] text-stone-500">Action result</p>
          <p className="mt-3 text-sm text-stone-700">{echoed || "No action result yet."}</p>
        </div>
      </aside>
    </div>
  );
}
`

const userProfileGoTemplate = `package users

import (
	"strings"

	rstf "github.com/rafbgarcia/rstf"
)

type ServerData struct {
	Name  string ` + "`json:\"name\"`" + `
	Role  string ` + "`json:\"role\"`" + `
	Bio   string ` + "`json:\"bio\"`" + `
	Color string ` + "`json:\"color\"`" + `
}

func SSR(ctx *rstf.Context) ServerData {
	id := strings.TrimSpace(ctx.Param("id"))
	if id == "" {
		id = "guest"
	}

	switch id {
	case "ada":
		return ServerData{
			Name:  "Ada Lovelace",
			Role:  "Prototype Architect",
			Bio:   "Ada owns the first draft of every idea and sharpens the product story into something users can feel.",
			Color: "amber",
		}
	case "rafa":
		return ServerData{
			Name:  "Rafa Garcia",
			Role:  "Framework Designer",
			Bio:   "Rafa keeps the Go-first shape tight and pushes the framework toward a real local-to-production workflow.",
			Color: "sky",
		}
		default:
			return ServerData{
			Name:  strings.ToUpper(id[:1]) + id[1:],
			Role:  "Guest Builder",
			Bio:   "Dynamic params are available in SSR, GET handlers, mutations, and live queries.",
			Color: "stone",
		}
	}
}
`

const userProfileTSXTemplate = `import { SSR, type RoutesUsersIdSSRProps } from "@rstf/routes/users._id";
import { AppBadge } from "../../shared/ui/app-badge";

const colorClasses: Record<string, string> = {
  amber: "from-amber-100 to-white text-amber-900 ring-amber-200",
  sky: "from-sky-100 to-white text-sky-900 ring-sky-200",
  stone: "from-stone-100 to-white text-stone-900 ring-stone-200",
};

export const View = SSR(function View({ name, role, bio, color }: RoutesUsersIdSSRProps) {

  return (
    <div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
      <section className="feature-card">
        <span className="chip">Dynamic route</span>
        <h2 className="mt-5 text-3xl font-semibold tracking-tight text-stone-950">{name}</h2>
        <p className="mt-3 text-sm font-medium uppercase tracking-[0.22em] text-stone-500">{role}</p>
        <p className="mt-5 max-w-2xl text-base leading-7 text-stone-600">{bio}</p>
        <div className="mt-8 flex flex-wrap gap-3">
          <a className="nav-link !bg-stone-900 !text-stone-50" href="/users/ada">
            Ada
          </a>
          <a className="nav-link !bg-white" href="/users/rafa">
            Rafa
          </a>
          <a className="nav-link !bg-white" href="/users/guest">
            Guest
          </a>
        </div>
      </section>

      <aside
        className={
          "feature-card bg-linear-to-br ring-1 " + (colorClasses[color] || colorClasses.stone)
        }
      >
        <p className="text-sm font-medium uppercase tracking-[0.22em]">Server-fed badge</p>
        <div className="mt-5">
          <AppBadge />
        </div>
        <p className="mt-5 text-sm leading-6 text-current/80">
          This page uses the route param and still composes a shared component with its own server data.
        </p>
      </aside>
    </div>
  );
});
`

const appBadgeGoTemplate = `package appbadge

import rstf "github.com/rafbgarcia/rstf"

type ServerData struct {
	Label   string ` + "`json:\"label\"`" + `
	Version string ` + "`json:\"version\"`" + `
}

func SSR(ctx *rstf.Context) ServerData {
	return ServerData{
		Label:   "Framework status",
		Version: "current features online",
	}
}
`

const appBadgeTSXTemplate = `import { SSR, type SharedUiAppBadgeSSRProps } from "@rstf/shared/ui/app-badge";

export const AppBadge = SSR(function AppBadge({ label, version }: SharedUiAppBadgeSSRProps) {

  return (
    <div className="rounded-[1.75rem] border border-stone-200 bg-white px-5 py-5 shadow-[0_18px_40px_-30px_rgba(24,24,27,0.4)]">
      <p className="text-xs font-semibold uppercase tracking-[0.22em] text-stone-500">{label}</p>
      <p className="mt-3 text-lg font-semibold text-stone-950">{version}</p>
      <div className="mt-4 h-2 rounded-full bg-stone-100">
        <div className="h-2 w-4/5 rounded-full bg-linear-to-r from-amber-400 via-lime-400 to-sky-500" />
      </div>
    </div>
  );
});
`

const readmeTemplate = "# {{ .DisplayName }}\n\n" +
	"Generated with `rstf init`.\n\n" +
	"## Commands\n\n" +
	"```bash\n" +
	"npm run dev\n" +
	"npm run build\n" +
	"```\n\n" +
	"The scaffold includes:\n\n" +
	"- typed SSR on `/`\n" +
	"- a live query, mutation, and action on `/live-chat/studio`\n" +
	"- a dynamic route on `/users/ada`\n" +
	"- a shared server-data component\n" +
	"- Tailwind v4 light styling\n"
