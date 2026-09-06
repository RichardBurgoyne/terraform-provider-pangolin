# Pangolin Terraform Provider Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build and release v0.1.0 of `terraform-provider-pangolin`, a `terraform-plugin-framework` provider covering the core, IaC-shaped subset of Pangolin's integration API, plus the two reusable GitHub Actions it depends on.

**Architecture:** A hand-rolled `internal/client` package wraps Pangolin's REST API (Bearer auth, JSON envelope `{data, success, error, message, status}`). `internal/provider` implements one `terraform-plugin-framework` resource/data source per Pangolin concept, each backed by client methods. CI and release automation live in `RichardBurgoyne/reusable-workflows` as composite actions the provider repo calls.

**Tech Stack:** Go (module `github.com/RichardBurgoyne/terraform-provider-pangolin`), `github.com/hashicorp/terraform-plugin-framework`, GoReleaser, GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-09-06-pangolin-provider-design.md`

## Global Constraints

- Commits use exactly five types: `feat`, `fix`, `chore`, `test`, `docs`. No scopes required.
- No AI-attribution trailers (Claude/Anthropic) in any commit, anywhere. No em dashes anywhere (code, comments, docs, commits).
- Emoji-suffixed step names in all GitHub Actions workflows/composite actions, matching the existing `RichardBurgoyne/reusable-workflows` convention (e.g. "Checkout 🛒", "Setup Go 🐹").
- All Pangolin API field names, HTTP verbs, and paths below were read directly from `github.com/fosrl/pangolin` source (`server/routers/*` and `server/routers/external.ts`, the route-mounting file that is the single source of truth for verb+path). Where a detail could not be confirmed from source, the task says so explicitly and gives a fallback.
- Every API response is wrapped in an envelope: `{"data": ..., "success": bool, "error": bool, "message": string, "status": int}`. `internal/client`'s `do` helper unwraps this once; individual client methods work with plain Go structs, not the envelope.
- License is MPL-2.0. Module path is `github.com/RichardBurgoyne/terraform-provider-pangolin`.
- Provider's `endpoint` attribute is the full API base URL the caller's Pangolin instance uses (including any version prefix like `/v1` if their instance has one) - the client appends resource paths directly to it and does not hardcode a version prefix itself, since the exact self-hosted base path was not confirmed from source.
- v1 scope is deliberately narrower than Pangolin's full API on a few resources where the real schema (read from source) turned out to be more complex than the original design spec assumed. Each such narrowing is called out in that resource's task with the specific fields deferred to v1.x.

---

## Task 1: `go-provider-test` composite action

**Repo:** `RichardBurgoyne/reusable-workflows` (existing, cloned/available via `gh repo clone RichardBurgoyne/reusable-workflows` if not already local)

**Files:**
- Create: `.github/actions/go-provider-test/action.yaml`

**Interfaces:**
- Consumes: nothing (first task)
- Produces: a composite action callable as `RichardBurgoyne/reusable-workflows/.github/actions/go-provider-test@v1` with inputs `working-directory` (optional, default `.`)

- [ ] **Step 1: Write the composite action**

```yaml
name: Go Provider Test
description: >-
  Format check, vet, lint, test, build, and docs-drift check for a Go
  Terraform provider. Assumes the repo is already checked out by the
  calling job.

inputs:
  working-directory:
    description: Directory containing the Go module
    required: false
    default: '.'

runs:
  using: composite
  steps:
    - name: Setup Go 🐹
      uses: actions/setup-go@v6
      with:
        go-version-file: ${{ inputs.working-directory }}/go.mod

    - name: Format check 🎨
      shell: bash
      working-directory: ${{ inputs.working-directory }}
      run: |
        unformatted="$(gofmt -l .)"
        if [ -n "$unformatted" ]; then
          echo "❌ The following files are not gofmt-formatted:"
          echo "$unformatted"
          exit 1
        fi

    - name: Vet 🔎
      shell: bash
      working-directory: ${{ inputs.working-directory }}
      run: go vet ./...

    - name: Install golangci-lint 🧹
      uses: golangci/golangci-lint-action@v7
      with:
        working-directory: ${{ inputs.working-directory }}
        version: latest

    - name: Test 🧪
      shell: bash
      working-directory: ${{ inputs.working-directory }}
      run: go test -race -cover ./...

    - name: Build 🏗️
      shell: bash
      working-directory: ${{ inputs.working-directory }}
      run: go build ./...

    - name: Install tfplugindocs 📖
      shell: bash
      working-directory: ${{ inputs.working-directory }}
      run: go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

    - name: Docs drift check 📚
      shell: bash
      working-directory: ${{ inputs.working-directory }}
      run: |
        tfplugindocs generate
        if [ -n "$(git status --porcelain docs/)" ]; then
          echo "❌ Generated docs are out of date. Run 'tfplugindocs generate' and commit the result."
          git status --porcelain docs/
          exit 1
        fi
```

- [ ] **Step 2: Validate the YAML**

Run: `cd /home/kiwi/claude/reusable-workflows && python3 -c "import yaml; yaml.safe_load(open('.github/actions/go-provider-test/action.yaml'))"`
Expected: no output, exit code 0 (valid YAML)

- [ ] **Step 3: Run actionlint if available, otherwise yamllint**

Run: `cd /home/kiwi/claude/reusable-workflows && (command -v actionlint && actionlint .github/actions/go-provider-test/action.yaml) || (pip show yamllint >/dev/null 2>&1 && yamllint .github/actions/go-provider-test/action.yaml) || echo "neither actionlint nor yamllint available, skipping"`
Expected: no errors reported

- [ ] **Step 4: Commit**

```bash
cd /home/kiwi/claude/reusable-workflows
git add .github/actions/go-provider-test/action.yaml
git commit -m "feat: add go-provider-test composite action"
```

---

## Task 2: `go-provider-release` composite action

**Repo:** `RichardBurgoyne/reusable-workflows`

**Files:**
- Create: `.github/actions/go-provider-release/action.yaml`

**Interfaces:**
- Consumes: nothing (independent of Task 1)
- Produces: a composite action callable as `RichardBurgoyne/reusable-workflows/.github/actions/go-provider-release@v1` with inputs `working-directory` (optional, default `.`) and secrets passed as inputs `gpg-private-key`, `gpg-passphrase`, `github-token`

- [ ] **Step 1: Write the composite action**

```yaml
name: Go Provider Release
description: >-
  Import a GPG signing key and run GoReleaser for a Go Terraform provider.
  Reads .goreleaser.yml from the calling repo. Assumes the repo is already
  checked out (with full history and tags, i.e. fetch-depth 0) by the
  calling job.

inputs:
  working-directory:
    description: Directory containing the Go module and .goreleaser.yml
    required: false
    default: '.'
  gpg-private-key:
    description: Armored GPG private key used to sign release checksums
    required: true
  gpg-passphrase:
    description: Passphrase for the GPG private key
    required: true
  github-token:
    description: Token GoReleaser uses to publish the GitHub Release
    required: true

runs:
  using: composite
  steps:
    - name: Setup Go 🐹
      uses: actions/setup-go@v6
      with:
        go-version-file: ${{ inputs.working-directory }}/go.mod

    - name: Import GPG key 🔐
      id: import-gpg
      uses: crazy-max/ghaction-import-gpg@v6
      with:
        gpg_private_key: ${{ inputs.gpg-private-key }}
        passphrase: ${{ inputs.gpg-passphrase }}

    - name: Release 🚀
      uses: goreleaser/goreleaser-action@v6
      with:
        workdir: ${{ inputs.working-directory }}
        args: release --clean
      env:
        GITHUB_TOKEN: ${{ inputs.github-token }}
        GPG_FINGERPRINT: ${{ steps.import-gpg.outputs.fingerprint }}
```

- [ ] **Step 2: Validate the YAML**

Run: `cd /home/kiwi/claude/reusable-workflows && python3 -c "import yaml; yaml.safe_load(open('.github/actions/go-provider-release/action.yaml'))"`
Expected: no output, exit code 0

- [ ] **Step 3: Commit**

```bash
cd /home/kiwi/claude/reusable-workflows
git add .github/actions/go-provider-release/action.yaml
git commit -m "feat: add go-provider-release composite action"
```

- [ ] **Step 4: Tag v1 and push (only after the user confirms they want this live)**

Run: `cd /home/kiwi/claude/reusable-workflows && git tag v1 -f && git push origin main --tags -f`
Expected: both new actions available at `RichardBurgoyne/reusable-workflows/.github/actions/{go-provider-test,go-provider-release}@v1`

Note: `terraform-prepare` already uses `v1.1`, not a moving `v1` tag on this repo (its caller pins `@v1.1` exactly). Check whether `v1` already exists as a tag on this repo before force-moving it - if it does and is used by other callers, cut a new tag (e.g. `v1.2`) instead of moving `v1`, and update this task's "Produces" reference accordingly before Task 6/7 use it.

---

## Task 3: Go module scaffold

**Repo:** `RichardBurgoyne/terraform-provider-pangolin` (new, already `git init`'d locally at `/home/kiwi/claude/terraform-provider-pangolin` with the spec committed)

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `internal/provider/provider.go` (empty-shell provider, no attributes yet - Task 5 fills in the schema)
- Create: `internal/provider/provider_test.go`
- Create: `GNUmakefile`
- Create: `LICENSE`
- Create: `.gitignore`
- Create: `README.md`
- Create: `terraform-registry-manifest.json`

**Interfaces:**
- Produces: package `github.com/RichardBurgoyne/terraform-provider-pangolin/internal/provider` exporting `func New() provider.Provider`; `main.go` serves it.

- [ ] **Step 1: Initialize the module**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go mod init github.com/RichardBurgoyne/terraform-provider-pangolin`
Expected: `go.mod` created with `go 1.23` or later (use the current stable Go toolchain).

- [ ] **Step 2: Add the framework dependency**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go get github.com/hashicorp/terraform-plugin-framework@latest`
Expected: `go.mod`/`go.sum` updated.

- [ ] **Step 3: Write the empty-shell provider**

```go
// internal/provider/provider.go
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = &pangolinProvider{}

type pangolinProvider struct {
	version string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &pangolinProvider{version: version}
	}
}

func (p *pangolinProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "pangolin"
	resp.Version = p.version
}

func (p *pangolinProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *pangolinProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
```

Note: `Schema` and `Configure` are added in Task 5 once `internal/client` exists (Task 4). Go will not compile a `provider.Provider` implementation missing those methods, so this file is intentionally incomplete until Task 5 - do not attempt `go build` until Task 5's Step 4 passes.

- [ ] **Step 4: Write main.go**

```go
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/provider"
)

var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with support for debuggers")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/RichardBurgoyne/pangolin",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 5: Write GNUmakefile, LICENSE, .gitignore, README.md, terraform-registry-manifest.json**

`GNUmakefile`:
```makefile
default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	go generate ./...
	tfplugindocs generate

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -race ./...

.PHONY: fmt lint test build install generate
```

`LICENSE`: full text of the Mozilla Public License 2.0 (fetch verbatim from `https://www.mozilla.org/media/MPL/2.0/index.txt` or copy from any existing MPL-2.0 project such as `hashicorp/terraform-plugin-framework`'s own `LICENSE` file).

`.gitignore`:
```
terraform-provider-pangolin
terraform-provider-pangolin_v*
.terraform/
*.tfstate
*.tfstate.*
crash.log
.DS_Store
dist/
```

`README.md`: a short intro (what the provider is, link to the registry page once published, `terraform { required_providers { pangolin = { source = "RichardBurgoyne/pangolin" } } }` usage snippet, link to `docs/`, contribution note pointing at `GNUmakefile` targets). No AI-attribution anywhere.

`terraform-registry-manifest.json`:
```json
{
  "version": 1,
  "metadata": {
    "protocol_versions": ["6.0"]
  }
}
```

- [ ] **Step 6: Write a placeholder test so `go test` has something to run**

```go
// internal/provider/provider_test.go
package provider

import "testing"

func TestNew(t *testing.T) {
	p := New("test")()
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}
```

- [ ] **Step 7: Verify it builds and the placeholder test passes**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go build ./... && go test ./...`
Expected: build succeeds (once Task 5 adds `Schema`/`Configure` - if running this step before Task 5, expect a compile error naming the missing methods; that is correct and expected at this point, do not "fix" it here). If you are doing Task 3 and Task 5 back to back in one sitting, it is fine to defer this verification step to the end of Task 5's Step 4 instead.

- [ ] **Step 8: Commit**

```bash
cd /home/kiwi/claude/terraform-provider-pangolin
git add go.mod go.sum main.go internal/provider/provider.go internal/provider/provider_test.go GNUmakefile LICENSE .gitignore README.md terraform-registry-manifest.json
git commit -m "chore: scaffold Go module and provider shell"
```

---

## Task 4: `internal/client` base

**Files:**
- Create: `internal/client/client.go`
- Test: `internal/client/client_test.go`

**Interfaces:**
- Consumes: nothing beyond the standard library
- Produces: `client.New(baseURL, apiKey string) *client.Client`, `func (c *Client) do(ctx context.Context, method, path string, body, out any) error` (unexported, used by every later resource-specific client file in the same package), `type APIError struct{ StatusCode int; Message string }` with `func IsNotFound(err error) bool`

- [ ] **Step 1: Write the failing test**

```go
// internal/client/client_test.go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDo_SetsAuthHeaderAndUnwrapsEnvelope(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"name": "hello"},
			"success": true,
			"error":   false,
			"message": "ok",
			"status":  200,
		})
	}))
	defer server.Close()

	c := New(server.URL, "test-token")

	var out struct {
		Name string `json:"name"`
	}
	if err := c.do(context.Background(), http.MethodGet, "/whatever", nil, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotAuth != "Bearer test-token" {
		t.Errorf("expected Authorization header %q, got %q", "Bearer test-token", gotAuth)
	}
	if out.Name != "hello" {
		t.Errorf("expected Name %q, got %q", "hello", out.Name)
	}
}

func TestDo_ReturnsAPIErrorOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"data":    nil,
			"success": false,
			"error":   true,
			"message": "Organization with ID x not found",
			"status":  404,
		})
	}))
	defer server.Close()

	c := New(server.URL, "test-token")

	err := c.do(context.Background(), http.MethodGet, "/org/x", nil, nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound(err) to be true, got false for error: %v", err)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -run TestDo -v`
Expected: FAIL - `client.New` undefined (package doesn't exist yet)

- [ ] **Step 3: Write the implementation**

```go
// internal/client/client.go
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("pangolin api error (status %d): %s", e.StatusCode, e.Message)
}

func IsNotFound(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.StatusCode == http.StatusNotFound
}

type envelope struct {
	Data    json.RawMessage `json:"data"`
	Success bool            `json:"success"`
	Message string          `json:"message"`
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	var env envelope
	if len(raw) > 0 {
		if unmarshalErr := json.Unmarshal(raw, &env); unmarshalErr != nil {
			return &APIError{StatusCode: resp.StatusCode, Message: string(raw)}
		}
	}

	if resp.StatusCode >= 400 || !env.Success {
		msg := env.Message
		if msg == "" {
			msg = string(raw)
		}
		return &APIError{StatusCode: resp.StatusCode, Message: msg}
	}

	if out != nil && len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("decoding response data: %w", err)
		}
	}

	return nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -v`
Expected: PASS (both tests)

- [ ] **Step 5: Commit**

```bash
cd /home/kiwi/claude/terraform-provider-pangolin
git add internal/client/client.go internal/client/client_test.go
git commit -m "feat: add base API client with auth and envelope unwrapping"
```

---

## Task 5: Provider configuration schema

**Files:**
- Modify: `internal/provider/provider.go`
- Modify: `internal/provider/provider_test.go`

**Interfaces:**
- Consumes: `client.New(baseURL, apiKey string) *client.Client` from Task 4
- Produces: `resp.DataSourceData`/`resp.ResourceData` set to `*client.Client` during `Configure`, which every resource/data source task from Task 8 onward type-asserts in its own `Configure` method

- [ ] **Step 1: Write the failing test**

```go
// internal/provider/provider_test.go (replace the whole file)
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
)

func TestNew(t *testing.T) {
	p := New("test")()
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestSchema_HasRequiredAttributes(t *testing.T) {
	p := New("test")()
	var resp provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &resp)

	for _, name := range []string{"endpoint", "api_key", "org_id"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("expected schema to have attribute %q", name)
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/provider/... -run TestSchema -v`
Expected: FAIL - `p.Schema` undefined, or compile error (pangolinProvider doesn't implement `provider.Provider` yet because `Schema`/`Configure` are missing)

- [ ] **Step 3: Write the implementation**

```go
// internal/provider/provider.go (full replacement)
package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var _ provider.Provider = &pangolinProvider{}

type pangolinProvider struct {
	version string
}

type pangolinProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	APIKey   types.String `tfsdk:"api_key"`
	OrgID    types.String `tfsdk:"org_id"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &pangolinProvider{version: version}
	}
}

func (p *pangolinProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "pangolin"
	resp.Version = p.version
}

func (p *pangolinProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with a self-hosted or cloud Pangolin instance.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL of the Pangolin integration API, including any version prefix your instance uses (e.g. https://pangolin.example.com/v1). Defaults to the PANGOLIN_ENDPOINT environment variable.",
			},
			"api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Bearer API key (organization or root scoped). Defaults to the PANGOLIN_API_KEY environment variable.",
			},
			"org_id": schema.StringAttribute{
				Optional:    true,
				Description: "Default organization ID used by resources that don't set org_id explicitly. Defaults to the PANGOLIN_ORG_ID environment variable.",
			},
		},
	}
}

func (p *pangolinProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config pangolinProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := config.Endpoint.ValueString()
	if endpoint == "" {
		endpoint = os.Getenv("PANGOLIN_ENDPOINT")
	}
	apiKey := config.APIKey.ValueString()
	if apiKey == "" {
		apiKey = os.Getenv("PANGOLIN_API_KEY")
	}

	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			pathRoot("endpoint"),
			"Missing Pangolin API Endpoint",
			"Set the endpoint attribute or the PANGOLIN_ENDPOINT environment variable.",
		)
	}
	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			pathRoot("api_key"),
			"Missing Pangolin API Key",
			"Set the api_key attribute or the PANGOLIN_API_KEY environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	c := client.New(endpoint, apiKey)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *pangolinProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *pangolinProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
```

Add a small local helper next to the provider (avoids importing `path` just for this one call, and matches how later resource files will import `path` themselves for `ImportState`):

```go
// internal/provider/path.go
package provider

import "github.com/hashicorp/terraform-plugin-framework/path"

func pathRoot(name string) path.Path {
	return path.Root(name)
}
```

- [ ] **Step 4: Run the tests to verify they pass, then verify the whole module builds**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./... && go build ./...`
Expected: PASS, build succeeds. This also resolves Task 3 Step 7's deferred verification.

- [ ] **Step 5: Commit**

```bash
cd /home/kiwi/claude/terraform-provider-pangolin
git add internal/provider/provider.go internal/provider/provider_test.go internal/provider/path.go
git commit -m "feat: add provider configuration schema and client wiring"
```

---

## Task 6: CI test workflow

**Files:**
- Create: `.github/workflows/test.yml`

**Interfaces:**
- Consumes: `RichardBurgoyne/reusable-workflows/.github/actions/go-provider-test@v1` (or whatever tag Task 2 Step 4 actually produced, e.g. `v1.2` - use that exact tag here)

- [ ] **Step 1: Write the workflow**

```yaml
name: Test

on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout 🛒
        uses: actions/checkout@v5

      - name: Test 🧪
        uses: RichardBurgoyne/reusable-workflows/.github/actions/go-provider-test@v1
```

- [ ] **Step 2: Validate the YAML locally**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && python3 -c "import yaml; yaml.safe_load(open('.github/workflows/test.yml'))"`
Expected: no output, exit code 0

- [ ] **Step 3: Commit**

```bash
cd /home/kiwi/claude/terraform-provider-pangolin
git add .github/workflows/test.yml
git commit -m "chore: add CI test workflow"
```

Note: this workflow can only be exercised for real once the repo is pushed to GitHub (Task 18 covers pushing and creating the remote repo). Local validation here is limited to YAML syntax; full verification happens once Task 18 pushes and a PR/push triggers it.

---

## Task 7: GoReleaser config and release workflow

**Files:**
- Create: `.goreleaser.yml`
- Create: `.github/workflows/release.yml`
- Modify: `README.md` (add a short "Releasing" section)

**Interfaces:**
- Consumes: `RichardBurgoyne/reusable-workflows/.github/actions/go-provider-release@v1` (same tag as Task 6)

- [ ] **Step 1: Write `.goreleaser.yml`**

```yaml
version: 2

before:
  hooks:
    - go mod tidy

builds:
  - env:
      - CGO_ENABLED=0
    mod_timestamp: '{{ .CommitTimestamp }}'
    flags:
      - -trimpath
    ldflags:
      - '-s -w -X main.version={{.Version}}'
    goos:
      - freebsd
      - windows
      - linux
      - darwin
    goarch:
      - amd64
      - '386'
      - arm
      - arm64
    ignore:
      - goos: darwin
        goarch: '386'
      - goos: windows
        goarch: arm
      - goos: windows
        goarch: arm64
    binary: '{{ .ProjectName }}_v{{ .Version }}'

archives:
  - format: zip
    name_template: '{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}'

checksum:
  extra_files:
    - glob: 'terraform-registry-manifest.json'
      name_template: '{{ .ProjectName }}_{{ .Version }}_manifest.json'
  name_template: '{{ .ProjectName }}_{{ .Version }}_SHA256SUMS'
  algorithm: sha256

signs:
  - artifacts: checksum
    args:
      - '--batch'
      - '--local-user'
      - '{{ .Env.GPG_FINGERPRINT }}'
      - '--output'
      - '${signature}'
      - '--detach-sign'
      - '${artifact}'

release:
  extra_files:
    - glob: 'terraform-registry-manifest.json'
      name_template: '{{ .ProjectName }}_{{ .Version }}_manifest.json'

changelog:
  sort: asc
  filters:
    exclude:
      - '^test:'
      - '^Merge '
  groups:
    - title: 'Features 🚀'
      regexp: '^feat'
      order: 0
    - title: 'Fixes 🐛'
      regexp: '^fix'
      order: 1
    - title: 'Docs 📖'
      regexp: '^docs'
      order: 2
    - title: 'Chores 🧹'
      regexp: '^chore'
      order: 3
```

This is the standard `hashicorp/terraform-provider-scaffolding-framework` GoReleaser layout adapted to this project's commit-type changelog grouping, written independently (not copied from stackopshq). The `filters.exclude` regexp `^test:` is what drops `test:` commits from release notes entirely, per the global constraint.

- [ ] **Step 2: Write the release workflow**

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout 🛒
        uses: actions/checkout@v5
        with:
          fetch-depth: 0

      - name: Release 🚀
        uses: RichardBurgoyne/reusable-workflows/.github/actions/go-provider-release@v1
        with:
          gpg-private-key: ${{ secrets.GPG_PRIVATE_KEY }}
          gpg-passphrase: ${{ secrets.GPG_PASSPHRASE }}
          github-token: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 3: Add a "Releasing" section to README.md**

```markdown
## Releasing

1. Merge everything intended for the release into `main`.
2. `git tag vX.Y.Z && git push origin vX.Y.Z`
3. The Release workflow builds, signs, and publishes to GitHub Releases.
   `test:` commits are excluded from the generated changelog.

First-time setup (once, by the repo owner): generate a dedicated GPG key
for this repo, add its armored private key and passphrase as the
`GPG_PRIVATE_KEY` and `GPG_PASSPHRASE` repo secrets, and upload the public
key to the Terraform Registry publisher settings so the registry can
verify signed releases.
```

- [ ] **Step 4: Validate both YAML files**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && python3 -c "import yaml; yaml.safe_load(open('.goreleaser.yml')); yaml.safe_load(open('.github/workflows/release.yml'))"`
Expected: no output, exit code 0

- [ ] **Step 5: Commit**

```bash
cd /home/kiwi/claude/terraform-provider-pangolin
git add .goreleaser.yml .github/workflows/release.yml README.md
git commit -m "chore: add GoReleaser config and release workflow"
```

Note: `goreleaser check` (validates `.goreleaser.yml` syntax against the real schema) requires the `goreleaser` binary. If it's available (`command -v goreleaser`), run `goreleaser check` in this repo and fix anything it flags before committing. If it isn't installed, this step is deferred to the first real release run in CI.

---

## Task 8: `pangolin_organization` resource and data source

Confirmed from `server/routers/external.ts`: `PUT /org` (create), `GET /org/:orgId` (read, response wrapped as `{"org": {...}}` - the only endpoint in this plan with that extra wrapping layer), `POST /org/:orgId` (update), `DELETE /org/:orgId` (delete). Create/read fields confirmed from `server/routers/org/createOrg.ts` and `getOrg.ts`: `orgId`, `name`, `subnet`, `utilitySubnet`.

v1 scope: `updateOrg.ts` was not read from source, so no update semantics are assumed here - all four attributes are `RequiresReplace`. Changing any of them destroys and recreates the organization rather than guessing at an unconfirmed update body.

**Files:**
- Create: `internal/client/organization.go`
- Test: `internal/client/organization_test.go`
- Create: `internal/provider/resource_organization.go`
- Test: `internal/provider/resource_organization_test.go`
- Create: `internal/provider/data_source_organization.go`
- Test: `internal/provider/data_source_organization_test.go`
- Modify: `internal/provider/provider.go` (register both in `Resources`/`DataSources`)
- Create: `examples/resources/pangolin_organization/resource.tf`
- Create: `examples/data-sources/pangolin_organization/data-source.tf`

**Interfaces:**
- Consumes: `client.Client.do` (Task 4), `resp.ResourceData`/`resp.DataSourceData` as `*client.Client` (Task 5)
- Produces: `client.Organization{OrgID, Name, Subnet, UtilitySubnet string}`, `client.Client.CreateOrganization/GetOrganization/DeleteOrganization`, `provider.NewOrganizationResource()`, `provider.NewOrganizationDataSource()`

- [ ] **Step 1: Write the failing client test**

```go
// internal/client/organization_test.go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateOrganization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/org" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"orgId": "acme", "name": "Acme", "subnet": "10.0.0.0/24", "utilitySubnet": "10.0.1.0/24",
			},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	org, err := c.CreateOrganization(context.Background(), CreateOrganizationRequest{
		OrgID: "acme", Name: "Acme", Subnet: "10.0.0.0/24", UtilitySubnet: "10.0.1.0/24",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if org.OrgID != "acme" || org.Name != "Acme" {
		t.Errorf("unexpected organization: %+v", org)
	}
}

func TestGetOrganization_UnwrapsOrgWrapper(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/org/acme" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"org": map[string]any{
					"orgId": "acme", "name": "Acme", "subnet": "10.0.0.0/24", "utilitySubnet": "10.0.1.0/24",
				},
			},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	org, err := c.GetOrganization(context.Background(), "acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if org.OrgID != "acme" {
		t.Errorf("expected orgId 'acme', got %+v", org)
	}
}

func TestDeleteOrganization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/org/acme" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteOrganization(context.Background(), "acme"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -run TestCreateOrganization -v`
Expected: FAIL - undefined `CreateOrganizationRequest`/`CreateOrganization`

- [ ] **Step 3: Implement the client methods**

```go
// internal/client/organization.go
package client

import (
	"context"
	"net/http"
)

type Organization struct {
	OrgID         string `json:"orgId"`
	Name          string `json:"name"`
	Subnet        string `json:"subnet"`
	UtilitySubnet string `json:"utilitySubnet"`
}

type CreateOrganizationRequest struct {
	OrgID         string `json:"orgId"`
	Name          string `json:"name"`
	Subnet        string `json:"subnet"`
	UtilitySubnet string `json:"utilitySubnet"`
}

func (c *Client) CreateOrganization(ctx context.Context, in CreateOrganizationRequest) (*Organization, error) {
	var out Organization
	if err := c.do(ctx, http.MethodPut, "/org", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetOrganization(ctx context.Context, orgID string) (*Organization, error) {
	var wrapper struct {
		Org Organization `json:"org"`
	}
	if err := c.do(ctx, http.MethodGet, "/org/"+orgID, nil, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Org, nil
}

func (c *Client) DeleteOrganization(ctx context.Context, orgID string) error {
	return c.do(ctx, http.MethodDelete, "/org/"+orgID, nil, nil)
}
```

- [ ] **Step 4: Run to verify the client tests pass**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -v`
Expected: PASS

- [ ] **Step 5: Write the resource**

```go
// internal/provider/resource_organization.go
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                = &organizationResource{}
	_ resource.ResourceWithImportState = &organizationResource{}
)

func NewOrganizationResource() resource.Resource { return &organizationResource{} }

type organizationResource struct {
	client *client.Client
}

type organizationResourceModel struct {
	OrgID         types.String `tfsdk:"org_id"`
	Name          types.String `tfsdk:"name"`
	Subnet        types.String `tfsdk:"subnet"`
	UtilitySubnet types.String `tfsdk:"utility_subnet"`
}

func (r *organizationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (r *organizationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages a Pangolin organization. All attributes are immutable after creation (update semantics were not confirmed against source); changing any of them replaces the organization.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Required:      true,
				Description:   "Unique organization identifier: lowercase letters, numbers, underscores, and single hyphens only.",
				PlanModifiers: replace,
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Display name of the organization.",
				PlanModifiers: replace,
			},
			"subnet": schema.StringAttribute{
				Required:      true,
				Description:   "IPv4 CIDR block for this organization's client subnet.",
				PlanModifiers: replace,
			},
			"utility_subnet": schema.StringAttribute{
				Required:      true,
				Description:   "IPv4 CIDR block for this organization's utility subnet. Must not overlap with subnet.",
				PlanModifiers: replace,
			},
		},
	}
}

func (r *organizationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *organizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan organizationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org, err := r.client.CreateOrganization(ctx, client.CreateOrganizationRequest{
		OrgID:         plan.OrgID.ValueString(),
		Name:          plan.Name.ValueString(),
		Subnet:        plan.Subnet.ValueString(),
		UtilitySubnet: plan.UtilitySubnet.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating organization", err.Error())
		return
	}

	plan.OrgID = types.StringValue(org.OrgID)
	plan.Name = types.StringValue(org.Name)
	plan.Subnet = types.StringValue(org.Subnet)
	plan.UtilitySubnet = types.StringValue(org.UtilitySubnet)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org, err := r.client.GetOrganization(ctx, state.OrgID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization", err.Error())
		return
	}

	state.Name = types.StringValue(org.Name)
	state.Subnet = types.StringValue(org.Subnet)
	state.UtilitySubnet = types.StringValue(org.UtilitySubnet)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *organizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Every attribute is RequiresReplace, so Terraform core never actually
	// calls this. The interface still requires an implementation.
	var plan organizationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state organizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteOrganization(ctx, state.OrgID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting organization", err.Error())
	}
}

func (r *organizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("org_id"), req, resp)
}
```

- [ ] **Step 6: Write the resource test (schema shape only, matching the pattern established in Task 5)**

```go
// internal/provider/resource_organization_test.go
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestOrganizationResource_Schema(t *testing.T) {
	r := NewOrganizationResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"org_id", "name", "subnet", "utility_subnet"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("expected attribute %q", name)
			continue
		}
		if !attr.IsRequired() {
			t.Errorf("expected attribute %q to be required", name)
		}
	}
}
```

- [ ] **Step 7: Write the data source**

```go
// internal/provider/data_source_organization.go
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var _ datasource.DataSource = &organizationDataSource{}

func NewOrganizationDataSource() datasource.DataSource { return &organizationDataSource{} }

type organizationDataSource struct {
	client *client.Client
}

func (d *organizationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (d *organizationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Pangolin organization by ID.",
		Attributes: map[string]schema.Attribute{
			"org_id":         schema.StringAttribute{Required: true, Description: "Organization ID to look up."},
			"name":           schema.StringAttribute{Computed: true, Description: "Display name of the organization."},
			"subnet":         schema.StringAttribute{Computed: true, Description: "IPv4 CIDR block for the client subnet."},
			"utility_subnet": schema.StringAttribute{Computed: true, Description: "IPv4 CIDR block for the utility subnet."},
		},
	}
}

func (d *organizationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *organizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model organizationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org, err := d.client.GetOrganization(ctx, model.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading organization", err.Error())
		return
	}

	model.Name = types.StringValue(org.Name)
	model.Subnet = types.StringValue(org.Subnet)
	model.UtilitySubnet = types.StringValue(org.UtilitySubnet)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
```

- [ ] **Step 8: Write the data source test**

```go
// internal/provider/data_source_organization_test.go
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestOrganizationDataSource_Schema(t *testing.T) {
	d := NewOrganizationDataSource()
	var resp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if _, ok := resp.Schema.Attributes["org_id"]; !ok {
		t.Error("expected attribute \"org_id\"")
	}
	if !resp.Schema.Attributes["name"].IsComputed() {
		t.Error("expected \"name\" to be computed")
	}
}
```

- [ ] **Step 9: Register both in the provider**

In `internal/provider/provider.go`, change `Resources` and `DataSources`:

```go
func (p *pangolinProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewOrganizationResource,
	}
}

func (p *pangolinProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewOrganizationDataSource,
	}
}
```

- [ ] **Step 10: Write example .tf files**

`examples/resources/pangolin_organization/resource.tf`:
```hcl
resource "pangolin_organization" "example" {
  org_id         = "acme"
  name           = "Acme Corp"
  subnet         = "10.10.0.0/24"
  utility_subnet = "10.10.1.0/24"
}
```

`examples/data-sources/pangolin_organization/data-source.tf`:
```hcl
data "pangolin_organization" "example" {
  org_id = "acme"
}
```

- [ ] **Step 11: Run the full test suite and build**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./... -v && go build ./...`
Expected: PASS, build succeeds

- [ ] **Step 12: Commit**

```bash
cd /home/kiwi/claude/terraform-provider-pangolin
git add internal/client/organization.go internal/client/organization_test.go \
  internal/provider/resource_organization.go internal/provider/resource_organization_test.go \
  internal/provider/data_source_organization.go internal/provider/data_source_organization_test.go \
  internal/provider/provider.go examples/resources/pangolin_organization examples/data-sources/pangolin_organization
git commit -m "feat: add pangolin_organization resource and data source"
```

---

## Task 9: `pangolin_domain` resource and data source

Confirmed from `server/routers/external.ts`: `PUT /org/:orgId/domain` (create), `GET /org/:orgId/domain/:domainId` (read), `POST /org/:orgId/domain/:domainId` (update), `DELETE /org/:orgId/domain/:domainId` (delete). Create body confirmed from `createOrgDomain.ts`: `type` (`ns`|`cname`|`wildcard`), `baseDomain`, `certResolver` (optional), `preferWildcardCert` (optional). Update body confirmed from `updateDomain.ts`: `certResolver`, `preferWildcardCert` only. Read fields confirmed from `listDomains.ts`'s projection (the same table `getDomain.ts` selects from, so this is the safe common subset): `domainId`, `baseDomain`, `verified`, `type`, `configManaged`, `certResolver`, `preferWildcardCert`.

This follows the same pattern as Task 8 (client file with Create/Get/Update/Delete, resource file, data source file, tests, examples, provider registration) - apply that same structure here.

**Files:**
- Create: `internal/client/domain.go` + `internal/client/domain_test.go`
- Create: `internal/provider/resource_domain.go` + `internal/provider/resource_domain_test.go`
- Create: `internal/provider/data_source_domain.go` + `internal/provider/data_source_domain_test.go`
- Modify: `internal/provider/provider.go`
- Create: `examples/resources/pangolin_domain/resource.tf`, `examples/data-sources/pangolin_domain/data-source.tf`

**Interfaces:**
- Consumes: `client.Client.do` (Task 4)
- Produces: `client.Domain{DomainID, BaseDomain, Type, Verified, ConfigManaged, CertResolver, PreferWildcardCert}`, `client.Client.CreateDomain/GetDomain/UpdateDomain/DeleteDomain`, `provider.NewDomainResource()`, `provider.NewDomainDataSource()`

- [ ] **Step 1: Write the failing client test**

```go
// internal/client/domain_test.go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateDomain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/org/acme/domain" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"domainId": "d1"},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	out, err := c.CreateDomain(context.Background(), "acme", CreateDomainRequest{Type: "wildcard", BaseDomain: "example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.DomainID != "d1" {
		t.Errorf("expected domainId 'd1', got %q", out.DomainID)
	}
}

func TestGetDomain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/org/acme/domain/d1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"domainId": "d1", "baseDomain": "example.com", "type": "wildcard", "verified": true},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	d, err := c.GetDomain(context.Background(), "acme", "d1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !d.Verified {
		t.Error("expected domain to be verified")
	}
}

func TestDeleteDomain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/org/acme/domain/d1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteDomain(context.Background(), "acme", "d1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -run TestCreateDomain -v`
Expected: FAIL - undefined symbols

- [ ] **Step 3: Implement the client methods**

```go
// internal/client/domain.go
package client

import (
	"context"
	"net/http"
)

type Domain struct {
	DomainID           string `json:"domainId"`
	BaseDomain         string `json:"baseDomain"`
	Type               string `json:"type"`
	Verified           bool   `json:"verified"`
	ConfigManaged      bool   `json:"configManaged"`
	CertResolver       string `json:"certResolver"`
	PreferWildcardCert bool   `json:"preferWildcardCert"`
}

type CreateDomainRequest struct {
	Type               string `json:"type"`
	BaseDomain         string `json:"baseDomain"`
	CertResolver       string `json:"certResolver,omitempty"`
	PreferWildcardCert *bool  `json:"preferWildcardCert,omitempty"`
}

type CreateDomainResult struct {
	DomainID string `json:"domainId"`
}

type UpdateDomainRequest struct {
	CertResolver       string `json:"certResolver,omitempty"`
	PreferWildcardCert *bool  `json:"preferWildcardCert,omitempty"`
}

func (c *Client) CreateDomain(ctx context.Context, orgID string, in CreateDomainRequest) (*CreateDomainResult, error) {
	var out CreateDomainResult
	if err := c.do(ctx, http.MethodPut, "/org/"+orgID+"/domain", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetDomain(ctx context.Context, orgID, domainID string) (*Domain, error) {
	var out Domain
	if err := c.do(ctx, http.MethodGet, "/org/"+orgID+"/domain/"+domainID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateDomain(ctx context.Context, orgID, domainID string, in UpdateDomainRequest) (*Domain, error) {
	var out Domain
	if err := c.do(ctx, http.MethodPost, "/org/"+orgID+"/domain/"+domainID, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteDomain(ctx context.Context, orgID, domainID string) error {
	return c.do(ctx, http.MethodDelete, "/org/"+orgID+"/domain/"+domainID, nil, nil)
}
```

- [ ] **Step 4: Run to verify the client tests pass**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -v`
Expected: PASS

- [ ] **Step 5: Write the resource**

```go
// internal/provider/resource_domain.go
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                = &domainResource{}
	_ resource.ResourceWithImportState = &domainResource{}
)

func NewDomainResource() resource.Resource { return &domainResource{} }

type domainResource struct {
	client *client.Client
}

type domainResourceModel struct {
	OrgID              types.String `tfsdk:"org_id"`
	DomainID           types.String `tfsdk:"domain_id"`
	Type               types.String `tfsdk:"type"`
	BaseDomain         types.String `tfsdk:"base_domain"`
	CertResolver       types.String `tfsdk:"cert_resolver"`
	PreferWildcardCert types.Bool   `tfsdk:"prefer_wildcard_cert"`
	Verified           types.Bool   `tfsdk:"verified"`
}

func (r *domainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *domainResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages a domain attached to a Pangolin organization. type and base_domain are immutable after creation.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Required:      true,
				Description:   "Organization ID this domain belongs to.",
				PlanModifiers: replace,
			},
			"domain_id": schema.StringAttribute{
				Computed:    true,
				Description: "Server-generated domain ID.",
			},
			"type": schema.StringAttribute{
				Required:      true,
				Description:   "One of ns, cname, or wildcard. Self-hosted (OSS) instances only support wildcard.",
				PlanModifiers: replace,
			},
			"base_domain": schema.StringAttribute{
				Required:      true,
				Description:   "The base domain to attach.",
				PlanModifiers: replace,
			},
			"cert_resolver": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Certificate resolver to use for this domain.",
			},
			"prefer_wildcard_cert": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to prefer a wildcard certificate for this domain.",
			},
			"verified": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the domain has completed verification.",
			},
		},
	}
}

func (r *domainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createIn := client.CreateDomainRequest{
		Type:       plan.Type.ValueString(),
		BaseDomain: plan.BaseDomain.ValueString(),
	}
	if !plan.CertResolver.IsUnknown() {
		createIn.CertResolver = plan.CertResolver.ValueString()
	}
	if !plan.PreferWildcardCert.IsUnknown() && !plan.PreferWildcardCert.IsNull() {
		v := plan.PreferWildcardCert.ValueBool()
		createIn.PreferWildcardCert = &v
	}

	created, err := r.client.CreateDomain(ctx, plan.OrgID.ValueString(), createIn)
	if err != nil {
		resp.Diagnostics.AddError("Error creating domain", err.Error())
		return
	}

	domain, err := r.client.GetDomain(ctx, plan.OrgID.ValueString(), created.DomainID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading newly created domain", err.Error())
		return
	}

	plan.DomainID = types.StringValue(domain.DomainID)
	plan.Type = types.StringValue(domain.Type)
	plan.BaseDomain = types.StringValue(domain.BaseDomain)
	plan.CertResolver = types.StringValue(domain.CertResolver)
	plan.PreferWildcardCert = types.BoolValue(domain.PreferWildcardCert)
	plan.Verified = types.BoolValue(domain.Verified)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := r.client.GetDomain(ctx, state.OrgID.ValueString(), state.DomainID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading domain", err.Error())
		return
	}

	state.Type = types.StringValue(domain.Type)
	state.BaseDomain = types.StringValue(domain.BaseDomain)
	state.CertResolver = types.StringValue(domain.CertResolver)
	state.PreferWildcardCert = types.BoolValue(domain.PreferWildcardCert)
	state.Verified = types.BoolValue(domain.Verified)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan domainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateIn := client.UpdateDomainRequest{CertResolver: plan.CertResolver.ValueString()}
	if !plan.PreferWildcardCert.IsNull() {
		v := plan.PreferWildcardCert.ValueBool()
		updateIn.PreferWildcardCert = &v
	}

	domain, err := r.client.UpdateDomain(ctx, plan.OrgID.ValueString(), plan.DomainID.ValueString(), updateIn)
	if err != nil {
		resp.Diagnostics.AddError("Error updating domain", err.Error())
		return
	}

	plan.CertResolver = types.StringValue(domain.CertResolver)
	plan.PreferWildcardCert = types.BoolValue(domain.PreferWildcardCert)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state domainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteDomain(ctx, state.OrgID.ValueString(), state.DomainID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting domain", err.Error())
	}
}

func (r *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain_id"), req, resp)
}
```

Note the `Update`'s `certResolver` handling matches source exactly: `updateDomain.ts` only writes `certResolver` when the field is present in the request body at all (`!== undefined`), and only writes `preferWildcardCert` when it is present and non-null. Sending an empty string for `cert_resolver` when the user hasn't set it is safe here because the schema marks it `Computed`, so Terraform always has a concrete prior value to send, never an unknown.

- [ ] **Step 6: Write the resource schema test**

```go
// internal/provider/resource_domain_test.go
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestDomainResource_Schema(t *testing.T) {
	r := NewDomainResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"org_id", "domain_id", "type", "base_domain", "cert_resolver", "prefer_wildcard_cert", "verified"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("expected attribute %q", name)
		}
	}
}
```

- [ ] **Step 7: Write the data source**

```go
// internal/provider/data_source_domain.go
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var _ datasource.DataSource = &domainDataSource{}

func NewDomainDataSource() datasource.DataSource { return &domainDataSource{} }

type domainDataSource struct {
	client *client.Client
}

func (d *domainDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (d *domainDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing domain by org_id and domain_id.",
		Attributes: map[string]schema.Attribute{
			"org_id":                schema.StringAttribute{Required: true},
			"domain_id":             schema.StringAttribute{Required: true},
			"type":                  schema.StringAttribute{Computed: true},
			"base_domain":           schema.StringAttribute{Computed: true},
			"cert_resolver":         schema.StringAttribute{Computed: true},
			"prefer_wildcard_cert":  schema.BoolAttribute{Computed: true},
			"verified":              schema.BoolAttribute{Computed: true},
		},
	}
}

func (d *domainDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *domainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model domainResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := d.client.GetDomain(ctx, model.OrgID.ValueString(), model.DomainID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain", err.Error())
		return
	}

	model.Type = types.StringValue(domain.Type)
	model.BaseDomain = types.StringValue(domain.BaseDomain)
	model.CertResolver = types.StringValue(domain.CertResolver)
	model.PreferWildcardCert = types.BoolValue(domain.PreferWildcardCert)
	model.Verified = types.BoolValue(domain.Verified)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
```

- [ ] **Step 8: Write the data source schema test**

```go
// internal/provider/data_source_domain_test.go
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestDomainDataSource_Schema(t *testing.T) {
	d := NewDomainDataSource()
	var resp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if !resp.Schema.Attributes["org_id"].IsRequired() {
		t.Error("expected org_id to be required")
	}
	if !resp.Schema.Attributes["verified"].IsComputed() {
		t.Error("expected verified to be computed")
	}
}
```

- [ ] **Step 9: Register both in the provider**

Add `NewDomainResource` to `Resources` and `NewDomainDataSource` to `DataSources` in `internal/provider/provider.go`, alongside the organization entries from Task 8.

- [ ] **Step 10: Write example .tf files**

`examples/resources/pangolin_domain/resource.tf`:
```hcl
resource "pangolin_domain" "example" {
  org_id      = pangolin_organization.example.org_id
  type        = "wildcard"
  base_domain = "apps.example.com"
}
```

`examples/data-sources/pangolin_domain/data-source.tf`:
```hcl
data "pangolin_domain" "example" {
  org_id    = "acme"
  domain_id = "d1"
}
```

- [ ] **Step 11: Run the full test suite and build**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./... -v && go build ./...`
Expected: PASS, build succeeds

- [ ] **Step 12: Commit**

```bash
cd /home/kiwi/claude/terraform-provider-pangolin
git add internal/client/domain.go internal/client/domain_test.go \
  internal/provider/resource_domain.go internal/provider/resource_domain_test.go \
  internal/provider/data_source_domain.go internal/provider/data_source_domain_test.go \
  internal/provider/provider.go examples/resources/pangolin_domain examples/data-sources/pangolin_domain
git commit -m "feat: add pangolin_domain resource and data source"
```

---

## Task 10: `pangolin_site` resource and data source

Confirmed from `server/routers/external.ts`: `PUT /org/:orgId/site` (create), `GET /site/:siteId` (read), `POST /site/:siteId` (update), `DELETE /site/:siteId` (delete). Create body confirmed from `createSite.ts`: `name`, `exitNodeId` (optional int), `niceId` (optional, server-generates if absent), `pubKey` (optional), `subnet` (optional), `newtId`/`secret` (optional, server-generates if absent - only meaningful for `type = "newt"`), `address` (optional), `type` (required enum `newt`|`wireguard`|`local`). Update body confirmed from `updateSite.ts`: `name`, `niceId`, `dockerSocketEnabled`, `autoUpdateEnabled`, `autoUpdateOverrideOrg`, all optional - these are the only three fields updatable after creation; everything else is `RequiresReplace`. `newtId`/`secret` are write-once credentials (only returned in the create response, per `CreateSiteResponse`), so they are `Computed` + `Sensitive` and never re-derived on `Read`.

Same file/task structure as Tasks 8-9.

**Files:**
- Create: `internal/client/site.go` + `internal/client/site_test.go`
- Create: `internal/provider/resource_site.go` + `internal/provider/resource_site_test.go`
- Create: `internal/provider/data_source_site.go` + `internal/provider/data_source_site_test.go`
- Modify: `internal/provider/provider.go`
- Create: `examples/resources/pangolin_site/resource.tf`, `examples/data-sources/pangolin_site/data-source.tf`

**Interfaces:**
- Consumes: `client.Client.do` (Task 4)
- Produces: `client.Site`, `client.Client.CreateSite/GetSite/UpdateSite/DeleteSite`, `provider.NewSiteResource()`, `provider.NewSiteDataSource()`

- [ ] **Step 1: Write the failing client test**

```go
// internal/client/site_test.go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateSite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/org/acme/site" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"siteId": float64(1), "niceId": "site-1", "name": "Site 1", "type": "newt",
				"newtId": "newt-abc", "secret": "shh",
			},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	site, err := c.CreateSite(context.Background(), "acme", CreateSiteRequest{Name: "Site 1", Type: "newt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if site.SiteID != 1 || site.NewtID != "newt-abc" || site.Secret != "shh" {
		t.Errorf("unexpected site: %+v", site)
	}
}

func TestUpdateSite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/site/1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"siteId": float64(1), "name": "Renamed", "type": "newt"},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	name := "Renamed"
	site, err := c.UpdateSite(context.Background(), 1, UpdateSiteRequest{Name: &name})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if site.Name != "Renamed" {
		t.Errorf("expected name 'Renamed', got %q", site.Name)
	}
}

func TestDeleteSite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/site/1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteSite(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -run TestCreateSite -v`
Expected: FAIL - undefined symbols

- [ ] **Step 3: Implement the client methods**

```go
// internal/client/site.go
package client

import (
	"context"
	"fmt"
	"net/http"
)

type Site struct {
	SiteID                int64  `json:"siteId"`
	NiceID                string `json:"niceId"`
	Name                  string `json:"name"`
	Type                  string `json:"type"`
	ExitNodeID            *int64 `json:"exitNodeId"`
	PubKey                string `json:"pubKey"`
	Subnet                string `json:"subnet"`
	Address               string `json:"address"`
	DockerSocketEnabled   bool   `json:"dockerSocketEnabled"`
	AutoUpdateEnabled     bool   `json:"autoUpdateEnabled"`
	AutoUpdateOverrideOrg bool   `json:"autoUpdateOverrideOrg"`
	NewtID                string `json:"newtId"`
	Secret                string `json:"secret"`
}

type CreateSiteRequest struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	ExitNodeID *int64 `json:"exitNodeId,omitempty"`
	NiceID     string `json:"niceId,omitempty"`
	PubKey     string `json:"pubKey,omitempty"`
	Subnet     string `json:"subnet,omitempty"`
	Address    string `json:"address,omitempty"`
}

type UpdateSiteRequest struct {
	Name                  *string `json:"name,omitempty"`
	NiceID                *string `json:"niceId,omitempty"`
	DockerSocketEnabled   *bool   `json:"dockerSocketEnabled,omitempty"`
	AutoUpdateEnabled     *bool   `json:"autoUpdateEnabled,omitempty"`
	AutoUpdateOverrideOrg *bool   `json:"autoUpdateOverrideOrg,omitempty"`
}

func (c *Client) CreateSite(ctx context.Context, orgID string, in CreateSiteRequest) (*Site, error) {
	var out Site
	if err := c.do(ctx, http.MethodPut, "/org/"+orgID+"/site", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetSite(ctx context.Context, siteID int64) (*Site, error) {
	var out Site
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/site/%d", siteID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateSite(ctx context.Context, siteID int64, in UpdateSiteRequest) (*Site, error) {
	var out Site
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/site/%d", siteID), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteSite(ctx context.Context, siteID int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/site/%d", siteID), nil, nil)
}
```

- [ ] **Step 4: Run to verify the client tests pass**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -v`
Expected: PASS

- [ ] **Step 5: Write the resource**

```go
// internal/provider/resource_site.go
package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                = &siteResource{}
	_ resource.ResourceWithImportState = &siteResource{}
)

func NewSiteResource() resource.Resource { return &siteResource{} }

type siteResource struct {
	client *client.Client
}

type siteResourceModel struct {
	OrgID                 types.String `tfsdk:"org_id"`
	SiteID                types.Int64  `tfsdk:"site_id"`
	NiceID                types.String `tfsdk:"nice_id"`
	Name                  types.String `tfsdk:"name"`
	Type                  types.String `tfsdk:"type"`
	ExitNodeID            types.Int64  `tfsdk:"exit_node_id"`
	PubKey                types.String `tfsdk:"pub_key"`
	Subnet                types.String `tfsdk:"subnet"`
	Address               types.String `tfsdk:"address"`
	NewtID                types.String `tfsdk:"newt_id"`
	Secret                types.String `tfsdk:"secret"`
	DockerSocketEnabled   types.Bool   `tfsdk:"docker_socket_enabled"`
	AutoUpdateEnabled     types.Bool   `tfsdk:"auto_update_enabled"`
	AutoUpdateOverrideOrg types.Bool   `tfsdk:"auto_update_override_org"`
}

func (r *siteResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site"
}

func (r *siteResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages a Pangolin site. Only name, docker_socket_enabled, auto_update_enabled, and auto_update_override_org can be updated after creation; every other attribute replaces the site if changed.",
		Attributes: map[string]schema.Attribute{
			"org_id":     schema.StringAttribute{Required: true, PlanModifiers: replace, Description: "Organization ID this site belongs to."},
			"site_id":    schema.Int64Attribute{Computed: true, Description: "Server-generated site ID."},
			"nice_id":    schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: replace, Description: "Human-readable ID, unique per org. Server-generated if omitted."},
			"name":       schema.StringAttribute{Required: true, Description: "Display name of the site."},
			"type":       schema.StringAttribute{Required: true, PlanModifiers: replace, Description: "One of newt, wireguard, or local."},
			"exit_node_id": schema.Int64Attribute{
				Optional:      true,
				PlanModifiers: []planmodifier.Int64{},
				Description:   "Exit node ID. Required for type = wireguard.",
			},
			"pub_key": schema.StringAttribute{Optional: true, PlanModifiers: replace, Description: "WireGuard public key. Required for type = wireguard."},
			"subnet":  schema.StringAttribute{Optional: true, PlanModifiers: replace, Description: "WireGuard tunnel subnet. Required for type = wireguard."},
			"address": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: replace, Description: "Client subnet address. Server-assigned if omitted."},
			"newt_id": schema.StringAttribute{Computed: true, Sensitive: true, Description: "Newt agent ID (type = newt only). Only ever populated from the create response."},
			"secret":  schema.StringAttribute{Computed: true, Sensitive: true, Description: "Newt agent secret (type = newt only). Only ever populated from the create response."},
			"docker_socket_enabled":    schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether the site can access the Docker socket."},
			"auto_update_enabled":      schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether the site's Newt agent auto-updates."},
			"auto_update_override_org": schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether this site overrides the org's auto-update setting."},
		},
	}
}

func (r *siteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *siteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan siteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.CreateSiteRequest{
		Name: plan.Name.ValueString(),
		Type: plan.Type.ValueString(),
	}
	if !plan.ExitNodeID.IsNull() && !plan.ExitNodeID.IsUnknown() {
		v := plan.ExitNodeID.ValueInt64()
		in.ExitNodeID = &v
	}
	if !plan.NiceID.IsUnknown() {
		in.NiceID = plan.NiceID.ValueString()
	}
	in.PubKey = plan.PubKey.ValueString()
	in.Subnet = plan.Subnet.ValueString()
	if !plan.Address.IsUnknown() {
		in.Address = plan.Address.ValueString()
	}

	site, err := r.client.CreateSite(ctx, plan.OrgID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating site", err.Error())
		return
	}

	setSiteModelFromAPI(&plan, site)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *siteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state siteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site, err := r.client.GetSite(ctx, state.SiteID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading site", err.Error())
		return
	}

	// newtId/secret are write-once: GetSite never returns them, so preserve
	// whatever is already in state instead of overwriting with empty values.
	newtID, secret := state.NewtID, state.Secret
	setSiteModelFromAPI(&state, site)
	state.NewtID, state.Secret = newtID, secret
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *siteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state siteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.UpdateSiteRequest{}
	name := plan.Name.ValueString()
	in.Name = &name
	if !plan.DockerSocketEnabled.IsUnknown() {
		v := plan.DockerSocketEnabled.ValueBool()
		in.DockerSocketEnabled = &v
	}
	if !plan.AutoUpdateEnabled.IsUnknown() {
		v := plan.AutoUpdateEnabled.ValueBool()
		in.AutoUpdateEnabled = &v
	}
	if !plan.AutoUpdateOverrideOrg.IsUnknown() {
		v := plan.AutoUpdateOverrideOrg.ValueBool()
		in.AutoUpdateOverrideOrg = &v
	}

	site, err := r.client.UpdateSite(ctx, state.SiteID.ValueInt64(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error updating site", err.Error())
		return
	}

	newtID, secret := state.NewtID, state.Secret
	setSiteModelFromAPI(&plan, site)
	plan.NewtID, plan.Secret = newtID, secret
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *siteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state siteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteSite(ctx, state.SiteID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Error deleting site", err.Error())
	}
}

func (r *siteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", "site_id must be a numeric ID: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site_id"), id)...)
}

func setSiteModelFromAPI(model *siteResourceModel, site *client.Site) {
	model.SiteID = types.Int64Value(site.SiteID)
	model.NiceID = types.StringValue(site.NiceID)
	model.Name = types.StringValue(site.Name)
	model.Type = types.StringValue(site.Type)
	if site.ExitNodeID != nil {
		model.ExitNodeID = types.Int64Value(*site.ExitNodeID)
	} else {
		model.ExitNodeID = types.Int64Null()
	}
	model.PubKey = types.StringValue(site.PubKey)
	model.Subnet = types.StringValue(site.Subnet)
	model.Address = types.StringValue(site.Address)
	model.DockerSocketEnabled = types.BoolValue(site.DockerSocketEnabled)
	model.AutoUpdateEnabled = types.BoolValue(site.AutoUpdateEnabled)
	model.AutoUpdateOverrideOrg = types.BoolValue(site.AutoUpdateOverrideOrg)
	if site.NewtID != "" {
		model.NewtID = types.StringValue(site.NewtID)
	}
	if site.Secret != "" {
		model.Secret = types.StringValue(site.Secret)
	}
}
```

Note: `exit_node_id` intentionally has no `RequiresReplace` plan modifier even though it behaves like an immutable field in practice - `createSite.ts` only persists `exitNodeId` for `type = "wireguard"` (it's explicitly not passed through for `type = "newt"`, chosen later when the Newt agent connects). Since it's conditionally meaningful, forcing replacement on any change is the wrong default for `type = "newt"` sites where the API doesn't even accept it. Document this nuance in the attribute description rather than encoding it in a plan modifier.

- [ ] **Step 6: Write the resource schema test**

```go
// internal/provider/resource_site_test.go
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestSiteResource_Schema(t *testing.T) {
	r := NewSiteResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"org_id", "site_id", "name", "type", "newt_id", "secret"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("expected attribute %q", name)
		}
	}
	if !resp.Schema.Attributes["secret"].IsSensitive() {
		t.Error("expected secret to be sensitive")
	}
}
```

- [ ] **Step 7: Write the data source**

```go
// internal/provider/data_source_site.go
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var _ datasource.DataSource = &siteDataSource{}

func NewSiteDataSource() datasource.DataSource { return &siteDataSource{} }

type siteDataSource struct {
	client *client.Client
}

type siteDataSourceModel struct {
	SiteID types.Int64  `tfsdk:"site_id"`
	NiceID types.String `tfsdk:"nice_id"`
	Name   types.String `tfsdk:"name"`
	Type   types.String `tfsdk:"type"`
}

func (d *siteDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site"
}

func (d *siteDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Pangolin site by site_id.",
		Attributes: map[string]schema.Attribute{
			"site_id": schema.Int64Attribute{Required: true},
			"nice_id": schema.StringAttribute{Computed: true},
			"name":    schema.StringAttribute{Computed: true},
			"type":    schema.StringAttribute{Computed: true},
		},
	}
}

func (d *siteDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *siteDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model siteDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site, err := d.client.GetSite(ctx, model.SiteID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error reading site", err.Error())
		return
	}

	model.NiceID = types.StringValue(site.NiceID)
	model.Name = types.StringValue(site.Name)
	model.Type = types.StringValue(site.Type)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
```

- [ ] **Step 8: Write the data source schema test**

```go
// internal/provider/data_source_site_test.go
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestSiteDataSource_Schema(t *testing.T) {
	d := NewSiteDataSource()
	var resp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if !resp.Schema.Attributes["site_id"].IsRequired() {
		t.Error("expected site_id to be required")
	}
}
```

- [ ] **Step 9: Register both in the provider**

Add `NewSiteResource` to `Resources` and `NewSiteDataSource` to `DataSources` in `internal/provider/provider.go`.

- [ ] **Step 10: Write example .tf files**

`examples/resources/pangolin_site/resource.tf`:
```hcl
resource "pangolin_site" "example" {
  org_id = pangolin_organization.example.org_id
  name   = "example-site"
  type   = "newt"
}

output "example_site_newt_credentials" {
  value     = { newt_id = pangolin_site.example.newt_id, secret = pangolin_site.example.secret }
  sensitive = true
}
```

`examples/data-sources/pangolin_site/data-source.tf`:
```hcl
data "pangolin_site" "example" {
  site_id = 1
}
```

- [ ] **Step 11: Run the full test suite and build**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./... -v && go build ./...`
Expected: PASS, build succeeds

- [ ] **Step 12: Commit**

```bash
cd /home/kiwi/claude/terraform-provider-pangolin
git add internal/client/site.go internal/client/site_test.go \
  internal/provider/resource_site.go internal/provider/resource_site_test.go \
  internal/provider/data_source_site.go internal/provider/data_source_site_test.go \
  internal/provider/provider.go examples/resources/pangolin_site examples/data-sources/pangolin_site
git commit -m "feat: add pangolin_site resource and data source"
```

---

## Task 11: `pangolin_resource` resource and data source (HTTP/SSH/RDP/VNC modes only)

Confirmed from `server/routers/external.ts`: `PUT /org/:orgId/resource` (create), `GET /resource/:resourceId` (read, flat object, confirmed from `getResource.ts`), `POST /resource/:resourceId` (update), `DELETE /resource/:resourceId` (delete).

**v1 scope narrowing** (this is the resource where the real schema, read from `createResource.ts`/`updateResource.ts`, was substantially larger than the design spec anticipated): only `mode` values `http`, `ssh`, `rdp`, `vnc` are supported. `inference` mode (requires `aiProviders` attachments) and raw `tcp`/`udp` resources (the `proxyPort`-based `createRawResourceSchema` path, gated behind the `allow_raw_resources` config flag) are deferred to v1.x. Within HTTP-mode fields, `sso`, `blockAccess`, `emailWhitelistEnabled`, `applyRules`, `skipToIdpId` (inline-policy/access-control fields), `headers`, the four `maintenance*` fields, `tlsServerName`, `setHostHeader`, and `resourcePolicyId` are all deferred to v1.x as a follow-up "resource access policy" feature - they all interact with the inline-vs-shared-policy semantics in `updateResource.ts` that add significant complexity beyond a first pass.

v1 create fields (confirmed from `createResource.ts`'s `createHttpResourceSchema`): `name`, `mode`, `domainId`, `subdomain` (optional), `stickySession` (optional), `postAuthPath` (optional), `pamMode` (optional, ssh only), `authDaemonPort`/`authDaemonMode` (optional, ssh only). v1 update fields (confirmed from `updateResource.ts`'s `updateHttpResourceBodySchema`, restricted to the v1 subset above): `name`, `niceId`, `subdomain`, `ssl`, `domainId`, `enabled`, `stickySession`, `postAuthPath`, `pamMode`, `authDaemonMode`, `authDaemonPort`.

**Files:**
- Create: `internal/client/resource.go` + `internal/client/resource_test.go`
- Create: `internal/provider/resource_resource.go` + `internal/provider/resource_resource_test.go`
- Create: `internal/provider/data_source_resource.go` + `internal/provider/data_source_resource_test.go`
- Modify: `internal/provider/provider.go`
- Create: `examples/resources/pangolin_resource/resource.tf`, `examples/data-sources/pangolin_resource/data-source.tf`

Note on naming: the Go package `internal/client` already has a type conceptually named "Resource" clashing with Terraform's own `resource` package import alias used throughout `internal/provider`. Name the client struct `PangolinResource` (not `Resource`) to avoid a confusing `client.Resource` vs `resource.Resource` split when reading the diff later.

**Interfaces:**
- Consumes: `client.Client.do` (Task 4)
- Produces: `client.PangolinResource`, `client.Client.CreateResource/GetResource/UpdateResource/DeleteResource`, `provider.NewPangolinResourceResource()`, `provider.NewPangolinResourceDataSource()`

- [ ] **Step 1: Write the failing client test**

```go
// internal/client/resource_test.go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/org/acme/resource" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"resourceId": float64(5), "niceId": "res-1", "name": "Res 1", "mode": "http",
				"domainId": "d1", "fullDomain": "res-1.example.com", "enabled": true, "ssl": true,
			},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	res, err := c.CreateResource(context.Background(), "acme", CreateResourceRequest{Name: "Res 1", Mode: "http", DomainID: "d1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ResourceID != 5 || res.FullDomain != "res-1.example.com" {
		t.Errorf("unexpected resource: %+v", res)
	}
}

func TestDeleteResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/resource/5" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteResource(context.Background(), 5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -run TestCreateResource -v`
Expected: FAIL - undefined symbols

- [ ] **Step 3: Implement the client methods**

```go
// internal/client/resource.go
package client

import (
	"context"
	"fmt"
	"net/http"
)

type PangolinResource struct {
	ResourceID     int64  `json:"resourceId"`
	NiceID         string `json:"niceId"`
	Name           string `json:"name"`
	Mode           string `json:"mode"`
	DomainID       string `json:"domainId"`
	FullDomain     string `json:"fullDomain"`
	Subdomain      string `json:"subdomain"`
	StickySession  bool   `json:"stickySession"`
	PostAuthPath   string `json:"postAuthPath"`
	PamMode        string `json:"pamMode"`
	AuthDaemonMode string `json:"authDaemonMode"`
	AuthDaemonPort int64  `json:"authDaemonPort"`
	Enabled        bool   `json:"enabled"`
	SSL            bool   `json:"ssl"`
}

type CreateResourceRequest struct {
	Name           string `json:"name"`
	Mode           string `json:"mode"`
	DomainID       string `json:"domainId"`
	Subdomain      string `json:"subdomain,omitempty"`
	StickySession  *bool  `json:"stickySession,omitempty"`
	PostAuthPath   string `json:"postAuthPath,omitempty"`
	PamMode        string `json:"pamMode,omitempty"`
	AuthDaemonMode string `json:"authDaemonMode,omitempty"`
	AuthDaemonPort *int64 `json:"authDaemonPort,omitempty"`
}

type UpdateResourceRequest struct {
	Name           *string `json:"name,omitempty"`
	NiceID         *string `json:"niceId,omitempty"`
	Subdomain      *string `json:"subdomain,omitempty"`
	SSL            *bool   `json:"ssl,omitempty"`
	DomainID       *string `json:"domainId,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
	StickySession  *bool   `json:"stickySession,omitempty"`
	PostAuthPath   *string `json:"postAuthPath,omitempty"`
	PamMode        *string `json:"pamMode,omitempty"`
	AuthDaemonMode *string `json:"authDaemonMode,omitempty"`
	AuthDaemonPort *int64  `json:"authDaemonPort,omitempty"`
}

func (c *Client) CreateResource(ctx context.Context, orgID string, in CreateResourceRequest) (*PangolinResource, error) {
	var out PangolinResource
	if err := c.do(ctx, http.MethodPut, "/org/"+orgID+"/resource", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetResource(ctx context.Context, resourceID int64) (*PangolinResource, error) {
	var out PangolinResource
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/resource/%d", resourceID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateResource(ctx context.Context, resourceID int64, in UpdateResourceRequest) (*PangolinResource, error) {
	var out PangolinResource
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/resource/%d", resourceID), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteResource(ctx context.Context, resourceID int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/resource/%d", resourceID), nil, nil)
}
```

- [ ] **Step 4: Run to verify the client tests pass**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -v`
Expected: PASS

- [ ] **Step 5: Write the resource**

```go
// internal/provider/resource_resource.go
package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                = &pangolinResourceResource{}
	_ resource.ResourceWithImportState = &pangolinResourceResource{}
)

func NewPangolinResourceResource() resource.Resource { return &pangolinResourceResource{} }

type pangolinResourceResource struct {
	client *client.Client
}

type pangolinResourceModel struct {
	OrgID          types.String `tfsdk:"org_id"`
	ResourceID     types.Int64  `tfsdk:"resource_id"`
	NiceID         types.String `tfsdk:"nice_id"`
	Name           types.String `tfsdk:"name"`
	Mode           types.String `tfsdk:"mode"`
	DomainID       types.String `tfsdk:"domain_id"`
	Subdomain      types.String `tfsdk:"subdomain"`
	FullDomain     types.String `tfsdk:"full_domain"`
	StickySession  types.Bool   `tfsdk:"sticky_session"`
	PostAuthPath   types.String `tfsdk:"post_auth_path"`
	PamMode        types.String `tfsdk:"pam_mode"`
	AuthDaemonMode types.String `tfsdk:"auth_daemon_mode"`
	AuthDaemonPort types.Int64  `tfsdk:"auth_daemon_port"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	SSL            types.Bool   `tfsdk:"ssl"`
}

func (r *pangolinResourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource"
}

func (r *pangolinResourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages a Pangolin HTTP/SSH/RDP/VNC resource (a proxied endpoint). Raw TCP/UDP resources and inference-mode (AI gateway) resources are not yet supported by this provider. mode and domain_id changes replace the resource.",
		Attributes: map[string]schema.Attribute{
			"org_id":      schema.StringAttribute{Required: true, PlanModifiers: replace, Description: "Organization ID this resource belongs to."},
			"resource_id": schema.Int64Attribute{Computed: true, Description: "Server-generated resource ID."},
			"nice_id":     schema.StringAttribute{Computed: true, Description: "Human-readable ID, unique per org."},
			"name":        schema.StringAttribute{Required: true, Description: "Display name of the resource."},
			"mode": schema.StringAttribute{
				Required:      true,
				PlanModifiers: replace,
				Description:   "One of http, ssh, rdp, vnc. inference and raw tcp/udp modes are not supported by this provider.",
			},
			"domain_id":       schema.StringAttribute{Required: true, Description: "Domain ID this resource is served from."},
			"subdomain":       schema.StringAttribute{Optional: true, Computed: true, Description: "Subdomain under domain_id."},
			"full_domain":     schema.StringAttribute{Computed: true, Description: "Fully-qualified domain this resource is reachable at."},
			"sticky_session":  schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether to enable sticky sessions."},
			"post_auth_path":  schema.StringAttribute{Optional: true, Description: "Path to redirect to after authentication."},
			"pam_mode":        schema.StringAttribute{Optional: true, Description: "SSH PAM mode: passthrough or push. Only meaningful for mode = ssh."},
			"auth_daemon_mode": schema.StringAttribute{Optional: true, Description: "One of site, remote, native. Only meaningful for mode = ssh."},
			"auth_daemon_port": schema.Int64Attribute{Optional: true, Description: "Auth daemon port. Only meaningful for mode = ssh."},
			"enabled":         schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether the resource is enabled."},
			"ssl":             schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether SSL is enabled."},
		},
	}
}

func (r *pangolinResourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *pangolinResourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan pangolinResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.CreateResourceRequest{
		Name:     plan.Name.ValueString(),
		Mode:     plan.Mode.ValueString(),
		DomainID: plan.DomainID.ValueString(),
	}
	if !plan.Subdomain.IsUnknown() {
		in.Subdomain = plan.Subdomain.ValueString()
	}
	if !plan.StickySession.IsUnknown() && !plan.StickySession.IsNull() {
		v := plan.StickySession.ValueBool()
		in.StickySession = &v
	}
	in.PostAuthPath = plan.PostAuthPath.ValueString()
	in.PamMode = plan.PamMode.ValueString()
	in.AuthDaemonMode = plan.AuthDaemonMode.ValueString()
	if !plan.AuthDaemonPort.IsNull() {
		v := plan.AuthDaemonPort.ValueInt64()
		in.AuthDaemonPort = &v
	}

	created, err := r.client.CreateResource(ctx, plan.OrgID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating resource", err.Error())
		return
	}

	setPangolinResourceModelFromAPI(&plan, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pangolinResourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state pangolinResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := r.client.GetResource(ctx, state.ResourceID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading resource", err.Error())
		return
	}

	setPangolinResourceModelFromAPI(&state, res)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *pangolinResourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state pangolinResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.UpdateResourceRequest{}
	name := plan.Name.ValueString()
	in.Name = &name
	if !plan.Subdomain.IsUnknown() {
		v := plan.Subdomain.ValueString()
		in.Subdomain = &v
	}
	if !plan.SSL.IsUnknown() {
		v := plan.SSL.ValueBool()
		in.SSL = &v
	}
	domainID := plan.DomainID.ValueString()
	in.DomainID = &domainID
	if !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		in.Enabled = &v
	}
	if !plan.StickySession.IsUnknown() {
		v := plan.StickySession.ValueBool()
		in.StickySession = &v
	}
	postAuthPath := plan.PostAuthPath.ValueString()
	in.PostAuthPath = &postAuthPath
	pamMode := plan.PamMode.ValueString()
	in.PamMode = &pamMode
	authDaemonMode := plan.AuthDaemonMode.ValueString()
	in.AuthDaemonMode = &authDaemonMode
	if !plan.AuthDaemonPort.IsNull() {
		v := plan.AuthDaemonPort.ValueInt64()
		in.AuthDaemonPort = &v
	}

	updated, err := r.client.UpdateResource(ctx, state.ResourceID.ValueInt64(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error updating resource", err.Error())
		return
	}

	setPangolinResourceModelFromAPI(&plan, updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pangolinResourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state pangolinResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteResource(ctx, state.ResourceID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Error deleting resource", err.Error())
	}
}

func (r *pangolinResourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", "resource_id must be a numeric ID: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_id"), id)...)
}

func setPangolinResourceModelFromAPI(model *pangolinResourceModel, res *client.PangolinResource) {
	model.ResourceID = types.Int64Value(res.ResourceID)
	model.NiceID = types.StringValue(res.NiceID)
	model.Name = types.StringValue(res.Name)
	model.Mode = types.StringValue(res.Mode)
	model.DomainID = types.StringValue(res.DomainID)
	model.Subdomain = types.StringValue(res.Subdomain)
	model.FullDomain = types.StringValue(res.FullDomain)
	model.StickySession = types.BoolValue(res.StickySession)
	model.PostAuthPath = types.StringValue(res.PostAuthPath)
	model.PamMode = types.StringValue(res.PamMode)
	model.AuthDaemonMode = types.StringValue(res.AuthDaemonMode)
	if res.AuthDaemonPort != 0 {
		model.AuthDaemonPort = types.Int64Value(res.AuthDaemonPort)
	}
	model.Enabled = types.BoolValue(res.Enabled)
	model.SSL = types.BoolValue(res.SSL)
}
```

- [ ] **Step 6: Write the resource schema test**

```go
// internal/provider/resource_resource_test.go
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestPangolinResourceResource_Schema(t *testing.T) {
	r := NewPangolinResourceResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"org_id", "resource_id", "name", "mode", "domain_id", "full_domain"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("expected attribute %q", name)
		}
	}
}
```

- [ ] **Step 7: Write the data source**

```go
// internal/provider/data_source_resource.go
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var _ datasource.DataSource = &pangolinResourceDataSource{}

func NewPangolinResourceDataSource() datasource.DataSource { return &pangolinResourceDataSource{} }

type pangolinResourceDataSource struct {
	client *client.Client
}

type pangolinResourceDataSourceModel struct {
	ResourceID types.Int64  `tfsdk:"resource_id"`
	Name       types.String `tfsdk:"name"`
	Mode       types.String `tfsdk:"mode"`
	FullDomain types.String `tfsdk:"full_domain"`
	Enabled    types.Bool   `tfsdk:"enabled"`
}

func (d *pangolinResourceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource"
}

func (d *pangolinResourceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Pangolin resource by resource_id.",
		Attributes: map[string]schema.Attribute{
			"resource_id": schema.Int64Attribute{Required: true},
			"name":        schema.StringAttribute{Computed: true},
			"mode":        schema.StringAttribute{Computed: true},
			"full_domain": schema.StringAttribute{Computed: true},
			"enabled":     schema.BoolAttribute{Computed: true},
		},
	}
}

func (d *pangolinResourceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *pangolinResourceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model pangolinResourceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := d.client.GetResource(ctx, model.ResourceID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error reading resource", err.Error())
		return
	}

	model.Name = types.StringValue(res.Name)
	model.Mode = types.StringValue(res.Mode)
	model.FullDomain = types.StringValue(res.FullDomain)
	model.Enabled = types.BoolValue(res.Enabled)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
```

- [ ] **Step 8: Write the data source schema test**

```go
// internal/provider/data_source_resource_test.go
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestPangolinResourceDataSource_Schema(t *testing.T) {
	d := NewPangolinResourceDataSource()
	var resp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if !resp.Schema.Attributes["resource_id"].IsRequired() {
		t.Error("expected resource_id to be required")
	}
}
```

- [ ] **Step 9: Register both in the provider**

Add `NewPangolinResourceResource` to `Resources` and `NewPangolinResourceDataSource` to `DataSources` in `internal/provider/provider.go`.

- [ ] **Step 10: Write example .tf files**

`examples/resources/pangolin_resource/resource.tf`:
```hcl
resource "pangolin_resource" "example" {
  org_id    = pangolin_organization.example.org_id
  name      = "example-app"
  mode      = "http"
  domain_id = pangolin_domain.example.domain_id
  subdomain = "app"
}
```

`examples/data-sources/pangolin_resource/data-source.tf`:
```hcl
data "pangolin_resource" "example" {
  resource_id = 5
}
```

- [ ] **Step 11: Run the full test suite and build**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./... -v && go build ./...`
Expected: PASS, build succeeds

- [ ] **Step 12: Commit**

```bash
cd /home/kiwi/claude/terraform-provider-pangolin
git add internal/client/resource.go internal/client/resource_test.go \
  internal/provider/resource_resource.go internal/provider/resource_resource_test.go \
  internal/provider/data_source_resource.go internal/provider/data_source_resource_test.go \
  internal/provider/provider.go examples/resources/pangolin_resource examples/data-sources/pangolin_resource
git commit -m "feat: add pangolin_resource resource and data source"
```

---

## Task 12: `pangolin_target` resource

Confirmed from `server/routers/external.ts`: `PUT /resource/:resourceId/target` (create), `GET /target/:targetId` (read), `POST /target/:targetId` (update), `DELETE /target/:targetId` (delete). No data source for targets (not in the approved v1 list - targets are almost always managed alongside the resource they belong to, not looked up independently).

**v1 scope narrowing:** all 14 health-check fields (`hcEnabled`, `hcPath`, `hcScheme`, `hcMode`, `hcHostname`, `hcPort`, `hcInterval`, `hcUnhealthyInterval`, `hcTimeout`, `hcHeaders`, `hcFollowRedirects`, `hcMethod`, `hcStatus`, `hcTlsServerName`, `hcHealthyThreshold`, `hcUnhealthyThreshold`) are deferred to v1.x as a "target health checks" follow-up. Confirmed from `createTarget.ts`/`updateTarget.ts`: `updateTargetBodySchema` requires `siteId` and `ip` on every update call (they are not optional there, unlike every other field), so `Update` always sends both.

**Files:**
- Create: `internal/client/target.go` + `internal/client/target_test.go`
- Create: `internal/provider/resource_target.go` + `internal/provider/resource_target_test.go`
- Modify: `internal/provider/provider.go`
- Create: `examples/resources/pangolin_target/resource.tf`

**Interfaces:**
- Consumes: `client.Client.do` (Task 4)
- Produces: `client.Target`, `client.Client.CreateTarget/GetTarget/UpdateTarget/DeleteTarget`, `provider.NewTargetResource()`

- [ ] **Step 1: Write the failing client test**

```go
// internal/client/target_test.go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/resource/5/target" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"targetId": float64(9), "siteId": float64(1), "ip": "10.0.0.5", "port": float64(80), "enabled": true},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	target, err := c.CreateTarget(context.Background(), 5, CreateTargetRequest{SiteID: 1, IP: "10.0.0.5", Port: 80})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.TargetID != 9 {
		t.Errorf("expected targetId 9, got %d", target.TargetID)
	}
}

func TestDeleteTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/target/9" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteTarget(context.Background(), 9); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -run TestCreateTarget -v`
Expected: FAIL - undefined symbols

- [ ] **Step 3: Implement the client methods**

```go
// internal/client/target.go
package client

import (
	"context"
	"fmt"
	"net/http"
)

type Target struct {
	TargetID        int64  `json:"targetId"`
	ResourceID      int64  `json:"resourceId"`
	SiteID          int64  `json:"siteId"`
	IP              string `json:"ip"`
	Mode            string `json:"mode"`
	Method          string `json:"method"`
	Port            int64  `json:"port"`
	Enabled         bool   `json:"enabled"`
	Path            string `json:"path"`
	PathMatchType   string `json:"pathMatchType"`
	RewritePath     string `json:"rewritePath"`
	RewritePathType string `json:"rewritePathType"`
	Priority        int64  `json:"priority"`
}

type CreateTargetRequest struct {
	SiteID          int64   `json:"siteId"`
	IP              string  `json:"ip"`
	Mode            string  `json:"mode,omitempty"`
	Method          *string `json:"method,omitempty"`
	Port            int64   `json:"port"`
	Enabled         *bool   `json:"enabled,omitempty"`
	Path            *string `json:"path,omitempty"`
	PathMatchType   *string `json:"pathMatchType,omitempty"`
	RewritePath     *string `json:"rewritePath,omitempty"`
	RewritePathType *string `json:"rewritePathType,omitempty"`
	Priority        *int64  `json:"priority,omitempty"`
}

type UpdateTargetRequest struct {
	SiteID          int64   `json:"siteId"`
	IP              string  `json:"ip"`
	Mode            *string `json:"mode,omitempty"`
	Method          *string `json:"method,omitempty"`
	Port            *int64  `json:"port,omitempty"`
	Enabled         *bool   `json:"enabled,omitempty"`
	Path            *string `json:"path,omitempty"`
	PathMatchType   *string `json:"pathMatchType,omitempty"`
	RewritePath     *string `json:"rewritePath,omitempty"`
	RewritePathType *string `json:"rewritePathType,omitempty"`
	Priority        *int64  `json:"priority,omitempty"`
}

func (c *Client) CreateTarget(ctx context.Context, resourceID int64, in CreateTargetRequest) (*Target, error) {
	var out Target
	if err := c.do(ctx, http.MethodPut, fmt.Sprintf("/resource/%d/target", resourceID), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetTarget(ctx context.Context, targetID int64) (*Target, error) {
	var out Target
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/target/%d", targetID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateTarget(ctx context.Context, targetID int64, in UpdateTargetRequest) (*Target, error) {
	var out Target
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/target/%d", targetID), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteTarget(ctx context.Context, targetID int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/target/%d", targetID), nil, nil)
}
```

- [ ] **Step 4: Run to verify the client tests pass**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -v`
Expected: PASS

- [ ] **Step 5: Write the resource**

```go
// internal/provider/resource_target.go
package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                = &targetResource{}
	_ resource.ResourceWithImportState = &targetResource{}
)

func NewTargetResource() resource.Resource { return &targetResource{} }

type targetResource struct {
	client *client.Client
}

type targetResourceModel struct {
	ResourceID      types.Int64  `tfsdk:"resource_id"`
	TargetID        types.Int64  `tfsdk:"target_id"`
	SiteID          types.Int64  `tfsdk:"site_id"`
	IP              types.String `tfsdk:"ip"`
	Mode            types.String `tfsdk:"mode"`
	Method          types.String `tfsdk:"method"`
	Port            types.Int64  `tfsdk:"port"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	Path            types.String `tfsdk:"path"`
	PathMatchType   types.String `tfsdk:"path_match_type"`
	RewritePath     types.String `tfsdk:"rewrite_path"`
	RewritePathType types.String `tfsdk:"rewrite_path_type"`
	Priority        types.Int64  `tfsdk:"priority"`
}

func (r *targetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_target"
}

func (r *targetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replaceInt := []planmodifier.Int64{int64planmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages a backend target on a Pangolin resource. Health checks are not yet supported by this provider.",
		Attributes: map[string]schema.Attribute{
			"resource_id":       schema.Int64Attribute{Required: true, PlanModifiers: replaceInt, Description: "Resource ID this target belongs to."},
			"target_id":         schema.Int64Attribute{Computed: true, Description: "Server-generated target ID."},
			"site_id":           schema.Int64Attribute{Required: true, Description: "Site ID this target routes through."},
			"ip":                schema.StringAttribute{Required: true, Description: "Target IP address or hostname."},
			"mode":              schema.StringAttribute{Optional: true, Computed: true, Description: "One of http, tcp, udp, ssh, rdp, vnc. Defaults to the resource's mode."},
			"method":            schema.StringAttribute{Optional: true, Computed: true, Description: "HTTP method restriction, if any."},
			"port":              schema.Int64Attribute{Required: true, Description: "Target port."},
			"enabled":           schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether this target is enabled."},
			"path":              schema.StringAttribute{Optional: true, Computed: true, Description: "Path match for HTTP-mode routing."},
			"path_match_type":   schema.StringAttribute{Optional: true, Computed: true, Description: "One of exact, prefix, regex."},
			"rewrite_path":      schema.StringAttribute{Optional: true, Computed: true, Description: "Path to rewrite to."},
			"rewrite_path_type": schema.StringAttribute{Optional: true, Computed: true, Description: "One of exact, prefix, regex, stripPrefix."},
			"priority":          schema.Int64Attribute{Optional: true, Computed: true, Description: "Routing priority (1-1000)."},
		},
	}
}

func (r *targetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *targetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan targetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.CreateTargetRequest{
		SiteID: plan.SiteID.ValueInt64(),
		IP:     plan.IP.ValueString(),
		Port:   plan.Port.ValueInt64(),
	}
	if !plan.Mode.IsUnknown() {
		in.Mode = plan.Mode.ValueString()
	}
	if !plan.Method.IsUnknown() {
		v := plan.Method.ValueString()
		in.Method = &v
	}
	if !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		in.Enabled = &v
	}
	if !plan.Path.IsUnknown() {
		v := plan.Path.ValueString()
		in.Path = &v
	}
	if !plan.PathMatchType.IsUnknown() {
		v := plan.PathMatchType.ValueString()
		in.PathMatchType = &v
	}
	if !plan.RewritePath.IsUnknown() {
		v := plan.RewritePath.ValueString()
		in.RewritePath = &v
	}
	if !plan.RewritePathType.IsUnknown() {
		v := plan.RewritePathType.ValueString()
		in.RewritePathType = &v
	}
	if !plan.Priority.IsUnknown() {
		v := plan.Priority.ValueInt64()
		in.Priority = &v
	}

	created, err := r.client.CreateTarget(ctx, plan.ResourceID.ValueInt64(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating target", err.Error())
		return
	}

	setTargetModelFromAPI(&plan, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *targetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state targetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	target, err := r.client.GetTarget(ctx, state.TargetID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading target", err.Error())
		return
	}

	resourceID := state.ResourceID
	setTargetModelFromAPI(&state, target)
	state.ResourceID = resourceID
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *targetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state targetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// siteId and ip are required on every update call per updateTargetBodySchema.
	in := client.UpdateTargetRequest{
		SiteID: plan.SiteID.ValueInt64(),
		IP:     plan.IP.ValueString(),
	}
	if !plan.Mode.IsUnknown() {
		v := plan.Mode.ValueString()
		in.Mode = &v
	}
	if !plan.Method.IsUnknown() {
		v := plan.Method.ValueString()
		in.Method = &v
	}
	port := plan.Port.ValueInt64()
	in.Port = &port
	if !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		in.Enabled = &v
	}
	if !plan.Path.IsUnknown() {
		v := plan.Path.ValueString()
		in.Path = &v
	}
	if !plan.PathMatchType.IsUnknown() {
		v := plan.PathMatchType.ValueString()
		in.PathMatchType = &v
	}
	if !plan.RewritePath.IsUnknown() {
		v := plan.RewritePath.ValueString()
		in.RewritePath = &v
	}
	if !plan.RewritePathType.IsUnknown() {
		v := plan.RewritePathType.ValueString()
		in.RewritePathType = &v
	}
	if !plan.Priority.IsUnknown() {
		v := plan.Priority.ValueInt64()
		in.Priority = &v
	}

	updated, err := r.client.UpdateTarget(ctx, state.TargetID.ValueInt64(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error updating target", err.Error())
		return
	}

	resourceID := plan.ResourceID
	setTargetModelFromAPI(&plan, updated)
	plan.ResourceID = resourceID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *targetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state targetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteTarget(ctx, state.TargetID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Error deleting target", err.Error())
	}
}

func (r *targetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", "target_id must be a numeric ID: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("target_id"), id)...)
}

func setTargetModelFromAPI(model *targetResourceModel, target *client.Target) {
	model.TargetID = types.Int64Value(target.TargetID)
	model.ResourceID = types.Int64Value(target.ResourceID)
	model.SiteID = types.Int64Value(target.SiteID)
	model.IP = types.StringValue(target.IP)
	model.Mode = types.StringValue(target.Mode)
	model.Method = types.StringValue(target.Method)
	model.Port = types.Int64Value(target.Port)
	model.Enabled = types.BoolValue(target.Enabled)
	model.Path = types.StringValue(target.Path)
	model.PathMatchType = types.StringValue(target.PathMatchType)
	model.RewritePath = types.StringValue(target.RewritePath)
	model.RewritePathType = types.StringValue(target.RewritePathType)
	model.Priority = types.Int64Value(target.Priority)
}
```

Note: `Read`/`Update` restore `ResourceID` from state/plan after calling `setTargetModelFromAPI` because `client.Target`'s `ResourceID` field comes back as `0` for AI-provider-owned targets (`providerId` set instead) - a case this provider's `pangolin_target` doesn't create, but being defensive here means a `0` from the API can never accidentally clobber a real resource_id already known from configuration.

- [ ] **Step 6: Write the resource schema test**

```go
// internal/provider/resource_target_test.go
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestTargetResource_Schema(t *testing.T) {
	r := NewTargetResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"resource_id", "target_id", "site_id", "ip", "port"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("expected attribute %q", name)
			continue
		}
		if name != "target_id" && !attr.IsRequired() {
			t.Errorf("expected attribute %q to be required", name)
		}
	}
}
```

- [ ] **Step 7: Register in the provider**

Add `NewTargetResource` to `Resources` in `internal/provider/provider.go`.

- [ ] **Step 8: Write the example .tf file**

`examples/resources/pangolin_target/resource.tf`:
```hcl
resource "pangolin_target" "example" {
  resource_id = pangolin_resource.example.resource_id
  site_id     = pangolin_site.example.site_id
  ip          = "10.0.0.5"
  port        = 8080
}
```

- [ ] **Step 9: Run the full test suite and build**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./... -v && go build ./...`
Expected: PASS, build succeeds

- [ ] **Step 10: Commit**

```bash
cd /home/kiwi/claude/terraform-provider-pangolin
git add internal/client/target.go internal/client/target_test.go \
  internal/provider/resource_target.go internal/provider/resource_target_test.go \
  internal/provider/provider.go examples/resources/pangolin_target
git commit -m "feat: add pangolin_target resource"
```

---

## Task 13: `pangolin_role` resource and data source

Confirmed from `server/routers/external.ts`: `PUT /org/:orgId/role` (create), `GET /org/:orgId/roles` (list, paginated), `POST /role/:roleId` (update), `DELETE /role/:roleId` (delete). **Important:** a direct `GET /role/:roleId` route exists in `role/getRole.ts` but is commented out in `external.ts` (disabled) - there is no single-role read endpoint on the live integration API. `Read` therefore calls `ListRoles` and filters by `roleId` client-side, exactly the same way `getRoleByID` below is implemented.

**v1 scope narrowing:** `sshSudoCommands` and `sshUnixGroups` are stored server-side as JSON-stringified arrays (`JSON.stringify(...)` in `createRole.ts`), and whether `GET`/list returns them as a parsed array or a raw JSON string was not confirmed from source (no dedicated read-side parsing was found for these two fields, unlike e.g. `hcHeaders` elsewhere which explicitly gets `JSON.parse`d back). Rather than guess the wire shape, these two fields plus `sshCreateHomeDir` are deferred to v1.x. v1 fields (confirmed from `createRole.ts`/`updateRole.ts`): `name`, `description`, `requireDeviceApproval`, `allowSsh`, `sshSudoMode`.

**Files:**
- Create: `internal/client/role.go` + `internal/client/role_test.go`
- Create: `internal/provider/resource_role.go` + `internal/provider/resource_role_test.go`
- Create: `internal/provider/data_source_role.go` + `internal/provider/data_source_role_test.go`
- Modify: `internal/provider/provider.go`
- Create: `examples/resources/pangolin_role/resource.tf`, `examples/data-sources/pangolin_role/data-source.tf`

**Interfaces:**
- Consumes: `client.Client.do` (Task 4)
- Produces: `client.Role`, `client.Client.CreateRole/ListRoles/GetRoleByID/UpdateRole/DeleteRole`, `provider.NewRoleResource()`, `provider.NewRoleDataSource()`

- [ ] **Step 1: Write the failing client test**

```go
// internal/client/role_test.go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/org/acme/role" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"roleId": float64(3), "orgId": "acme", "name": "Editors"},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	role, err := c.CreateRole(context.Background(), "acme", CreateRoleRequest{Name: "Editors"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role.RoleID != 3 {
		t.Errorf("expected roleId 3, got %d", role.RoleID)
	}
}

func TestGetRoleByID_FiltersListRoles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/org/acme/roles" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"roles": []map[string]any{
					{"roleId": float64(3), "orgId": "acme", "name": "Editors"},
					{"roleId": float64(4), "orgId": "acme", "name": "Viewers"},
				},
			},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	role, err := c.GetRoleByID(context.Background(), "acme", 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role.Name != "Viewers" {
		t.Errorf("expected role 'Viewers', got %+v", role)
	}

	if _, err := c.GetRoleByID(context.Background(), "acme", 999); !IsNotFound(err) {
		t.Errorf("expected IsNotFound for missing role, got %v", err)
	}
}

func TestDeleteRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/role/3" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteRole(context.Background(), 3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -run 'TestCreateRole|TestGetRoleByID|TestDeleteRole' -v`
Expected: FAIL - undefined symbols

- [ ] **Step 3: Implement the client methods**

```go
// internal/client/role.go
package client

import (
	"context"
	"fmt"
	"net/http"
)

type Role struct {
	RoleID                int64  `json:"roleId"`
	OrgID                 string `json:"orgId"`
	Name                  string `json:"name"`
	Description           string `json:"description"`
	IsAdmin               bool   `json:"isAdmin"`
	RequireDeviceApproval bool   `json:"requireDeviceApproval"`
	SSHSudoMode           string `json:"sshSudoMode"`
	AllowSSH              bool   `json:"allowSsh"`
}

type CreateRoleRequest struct {
	Name                  string `json:"name"`
	Description           string `json:"description,omitempty"`
	RequireDeviceApproval *bool  `json:"requireDeviceApproval,omitempty"`
	AllowSSH              *bool  `json:"allowSsh,omitempty"`
	SSHSudoMode           string `json:"sshSudoMode,omitempty"`
}

type UpdateRoleRequest struct {
	Name                  *string `json:"name,omitempty"`
	Description           *string `json:"description,omitempty"`
	RequireDeviceApproval *bool   `json:"requireDeviceApproval,omitempty"`
	AllowSSH              *bool   `json:"allowSsh,omitempty"`
	SSHSudoMode           *string `json:"sshSudoMode,omitempty"`
}

type listRolesResult struct {
	Roles []Role `json:"roles"`
}

func (c *Client) CreateRole(ctx context.Context, orgID string, in CreateRoleRequest) (*Role, error) {
	var out Role
	if err := c.do(ctx, http.MethodPut, "/org/"+orgID+"/role", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListRoles(ctx context.Context, orgID string) ([]Role, error) {
	var out listRolesResult
	if err := c.do(ctx, http.MethodGet, "/org/"+orgID+"/roles", nil, &out); err != nil {
		return nil, err
	}
	return out.Roles, nil
}

// GetRoleByID has no dedicated API endpoint (GET /role/:roleId is disabled
// server-side), so it lists every role in the org and filters client-side.
func (c *Client) GetRoleByID(ctx context.Context, orgID string, roleID int64) (*Role, error) {
	roles, err := c.ListRoles(ctx, orgID)
	if err != nil {
		return nil, err
	}
	for _, role := range roles {
		if role.RoleID == roleID {
			return &role, nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("role %d not found in org %s", roleID, orgID)}
}

func (c *Client) UpdateRole(ctx context.Context, roleID int64, in UpdateRoleRequest) (*Role, error) {
	var out Role
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/role/%d", roleID), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteRole(ctx context.Context, roleID int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/role/%d", roleID), nil, nil)
}
```

- [ ] **Step 4: Run to verify the client tests pass**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -v`
Expected: PASS

- [ ] **Step 5: Write the resource**

```go
// internal/provider/resource_role.go
package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                = &roleResource{}
	_ resource.ResourceWithImportState = &roleResource{}
)

func NewRoleResource() resource.Resource { return &roleResource{} }

type roleResource struct {
	client *client.Client
}

type roleResourceModel struct {
	OrgID                 types.String `tfsdk:"org_id"`
	RoleID                types.Int64  `tfsdk:"role_id"`
	Name                  types.String `tfsdk:"name"`
	Description           types.String `tfsdk:"description"`
	RequireDeviceApproval types.Bool   `tfsdk:"require_device_approval"`
	AllowSSH              types.Bool   `tfsdk:"allow_ssh"`
	SSHSudoMode           types.String `tfsdk:"ssh_sudo_mode"`
}

func (r *roleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *roleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Pangolin role. Fine-grained SSH sudo command/group lists are not yet supported by this provider.",
		Attributes: map[string]schema.Attribute{
			"org_id":  schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Description: "Organization ID this role belongs to."},
			"role_id": schema.Int64Attribute{Computed: true, Description: "Server-generated role ID."},
			"name":    schema.StringAttribute{Required: true, Description: "Role name, unique per org."},
			"description":             schema.StringAttribute{Optional: true, Computed: true, Description: "Role description."},
			"require_device_approval": schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether devices used by members of this role require approval."},
			"allow_ssh":               schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether members of this role can sign SSH keys."},
			"ssh_sudo_mode":           schema.StringAttribute{Optional: true, Computed: true, Description: "One of none, full, commands."},
		},
	}
}

func (r *roleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.CreateRoleRequest{Name: plan.Name.ValueString()}
	if !plan.Description.IsUnknown() {
		in.Description = plan.Description.ValueString()
	}
	if !plan.RequireDeviceApproval.IsUnknown() {
		v := plan.RequireDeviceApproval.ValueBool()
		in.RequireDeviceApproval = &v
	}
	if !plan.AllowSSH.IsUnknown() {
		v := plan.AllowSSH.ValueBool()
		in.AllowSSH = &v
	}
	if !plan.SSHSudoMode.IsUnknown() {
		in.SSHSudoMode = plan.SSHSudoMode.ValueString()
	}

	role, err := r.client.CreateRole(ctx, plan.OrgID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating role", err.Error())
		return
	}

	setRoleModelFromAPI(&plan, role)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.GetRoleByID(ctx, state.OrgID.ValueString(), state.RoleID.ValueInt64())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading role", err.Error())
		return
	}

	setRoleModelFromAPI(&state, role)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state roleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.UpdateRoleRequest{}
	name := plan.Name.ValueString()
	in.Name = &name
	if !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		in.Description = &v
	}
	if !plan.RequireDeviceApproval.IsUnknown() {
		v := plan.RequireDeviceApproval.ValueBool()
		in.RequireDeviceApproval = &v
	}
	if !plan.AllowSSH.IsUnknown() {
		v := plan.AllowSSH.ValueBool()
		in.AllowSSH = &v
	}
	if !plan.SSHSudoMode.IsUnknown() {
		v := plan.SSHSudoMode.ValueString()
		in.SSHSudoMode = &v
	}

	role, err := r.client.UpdateRole(ctx, state.RoleID.ValueInt64(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error updating role", err.Error())
		return
	}

	setRoleModelFromAPI(&plan, role)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteRole(ctx, state.RoleID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Error deleting role", err.Error())
	}
}

func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", "role_id must be a numeric ID: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_id"), id)...)
}

func setRoleModelFromAPI(model *roleResourceModel, role *client.Role) {
	model.RoleID = types.Int64Value(role.RoleID)
	model.OrgID = types.StringValue(role.OrgID)
	model.Name = types.StringValue(role.Name)
	model.Description = types.StringValue(role.Description)
	model.RequireDeviceApproval = types.BoolValue(role.RequireDeviceApproval)
	model.AllowSSH = types.BoolValue(role.AllowSSH)
	model.SSHSudoMode = types.StringValue(role.SSHSudoMode)
}
```

Note: `ImportState` here cannot set `org_id` (only `role_id` is known from the import ID, and there is no `GET /role/:roleId` to derive the org from). The first `Read` after import will therefore fail because `GetRoleByID` requires `org_id`, which will be empty. Document this in the resource's `Description` and in `docs/` (Task 18): import for `pangolin_role` requires the config to already set `org_id` before running `terraform import`, since the API gives no other way to resolve a role's org from its ID alone.

- [ ] **Step 6: Write the resource schema test**

```go
// internal/provider/resource_role_test.go
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestRoleResource_Schema(t *testing.T) {
	r := NewRoleResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"org_id", "role_id", "name", "allow_ssh"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("expected attribute %q", name)
		}
	}
}
```

- [ ] **Step 7: Write the data source**

```go
// internal/provider/data_source_role.go
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var _ datasource.DataSource = &roleDataSource{}

func NewRoleDataSource() datasource.DataSource { return &roleDataSource{} }

type roleDataSource struct {
	client *client.Client
}

type roleDataSourceModel struct {
	OrgID  types.String `tfsdk:"org_id"`
	RoleID types.Int64  `tfsdk:"role_id"`
	Name   types.String `tfsdk:"name"`
}

func (d *roleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *roleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing role by org_id and either role_id or name. Exactly one of role_id or name must be set.",
		Attributes: map[string]schema.Attribute{
			"org_id":  schema.StringAttribute{Required: true},
			"role_id": schema.Int64Attribute{Optional: true, Computed: true},
			"name":    schema.StringAttribute{Optional: true, Computed: true},
		},
	}
}

func (d *roleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *roleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model roleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, err := d.client.ListRoles(ctx, model.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing roles", err.Error())
		return
	}

	var found *client.Role
	for i := range roles {
		if !model.RoleID.IsNull() && roles[i].RoleID == model.RoleID.ValueInt64() {
			found = &roles[i]
			break
		}
		if !model.Name.IsNull() && roles[i].Name == model.Name.ValueString() {
			found = &roles[i]
			break
		}
	}
	if found == nil {
		resp.Diagnostics.AddError("Role not found", "No role matched the given role_id or name in this org.")
		return
	}

	model.RoleID = types.Int64Value(found.RoleID)
	model.Name = types.StringValue(found.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
```

- [ ] **Step 8: Write the data source schema test**

```go
// internal/provider/data_source_role_test.go
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestRoleDataSource_Schema(t *testing.T) {
	d := NewRoleDataSource()
	var resp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if !resp.Schema.Attributes["org_id"].IsRequired() {
		t.Error("expected org_id to be required")
	}
}
```

- [ ] **Step 9: Register both in the provider**

Add `NewRoleResource` to `Resources` and `NewRoleDataSource` to `DataSources` in `internal/provider/provider.go`.

- [ ] **Step 10: Write example .tf files**

`examples/resources/pangolin_role/resource.tf`:
```hcl
resource "pangolin_role" "example" {
  org_id      = pangolin_organization.example.org_id
  name        = "editors"
  description = "Can edit but not manage billing"
}
```

`examples/data-sources/pangolin_role/data-source.tf`:
```hcl
data "pangolin_role" "example" {
  org_id = "acme"
  name   = "editors"
}
```

- [ ] **Step 11: Run the full test suite and build**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./... -v && go build ./...`
Expected: PASS, build succeeds

- [ ] **Step 12: Commit**

```bash
cd /home/kiwi/claude/terraform-provider-pangolin
git add internal/client/role.go internal/client/role_test.go \
  internal/provider/resource_role.go internal/provider/resource_role_test.go \
  internal/provider/data_source_role.go internal/provider/data_source_role_test.go \
  internal/provider/provider.go examples/resources/pangolin_role examples/data-sources/pangolin_role
git commit -m "feat: add pangolin_role resource and data source"
```

---

## Task 14: `pangolin_user` resource and data source

Confirmed from `server/routers/external.ts`: `PUT /org/:orgId/user` (create), `GET /org/:orgId/user/:userId` (read), `GET /org/:orgId/users` (list), `POST /org/:orgId/user/:userId` (update), `DELETE /org/:orgId/user/:userId` (delete). Confirmed from `createOrgUser.ts`: **the create response body is `{}` - it returns no user data at all**, not even the generated `userId`. `Create` must follow up with `ListUsers` filtered by `username` to discover the ID Terraform needs to track. Confirmed from `createOrgUser.ts`: `type: "internal"` is explicitly rejected server-side ("Internal users are not supported yet"), so this resource only supports `type = "oidc"`, requiring a plain numeric `idp_id` the caller already knows (there is no `pangolin_idp` resource/data source in this plan - IdP configuration is v1.x). Confirmed from `updateOrgUser.ts`: the only field the update endpoint accepts is `autoProvisioned` - nothing else about an org-user membership can be changed via that route. Confirmed from `external.ts`: the only role-membership mutation route reachable after creation is `POST /role/:roleId/add/:userId` (adds one role, no corresponding removal route exists in the integration API), so `role_ids` can only grow, never shrink, without destroying and recreating the resource.

`GetOrgUser`'s exact response shape was not read from source (only its route mount was confirmed) - the fields modeled below (`userId`, `username`, `email`, `name`, `type`, `idpId`) are the columns `createOrgUser.ts` itself inserts into the `users` table, which is a reasonable floor, not a confirmed ceiling. If the live response nests fields differently, adjust `client.User`'s tags during implementation and note the correction in the commit.

**Files:**
- Create: `internal/client/user.go` + `internal/client/user_test.go`
- Create: `internal/provider/resource_user.go` + `internal/provider/resource_user_test.go`
- Create: `internal/provider/data_source_user.go` + `internal/provider/data_source_user_test.go`
- Modify: `internal/provider/provider.go`
- Create: `examples/resources/pangolin_user/resource.tf`, `examples/data-sources/pangolin_user/data-source.tf`

**Interfaces:**
- Consumes: `client.Client.do` (Task 4)
- Produces: `client.User`, `client.Client.CreateOrgUser/GetOrgUser/ListUsers/GetUserByUsername/UpdateOrgUser/AddUserRole/DeleteOrgUser`, `provider.NewUserResource()`, `provider.NewUserDataSource()`

- [ ] **Step 1: Write the failing client test**

```go
// internal/client/user_test.go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateOrgUser_ThenLookUpByUsername(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/org/acme/user":
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}, "success": true})
		case r.Method == http.MethodGet && r.URL.Path == "/org/acme/users":
			json.NewEncoder(w).Encode(map[string]any{
				"data":    map[string]any{"users": []map[string]any{{"userId": "u1", "username": "jane", "type": "oidc"}}},
				"success": true,
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.CreateOrgUser(context.Background(), "acme", CreateOrgUserRequest{Username: "jane", Type: "oidc", IdpID: 1, RoleIDs: []int64{2}}); err != nil {
		t.Fatalf("unexpected error creating user: %v", err)
	}

	user, err := c.GetUserByUsername(context.Background(), "acme", "jane")
	if err != nil {
		t.Fatalf("unexpected error looking up user: %v", err)
	}
	if user.UserID != "u1" {
		t.Errorf("expected userId 'u1', got %q", user.UserID)
	}
}

func TestAddUserRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/role/2/add/u1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.AddUserRole(context.Background(), 2, "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteOrgUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/org/acme/user/u1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteOrgUser(context.Background(), "acme", "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -run 'TestCreateOrgUser|TestAddUserRole|TestDeleteOrgUser' -v`
Expected: FAIL - undefined symbols

- [ ] **Step 3: Implement the client methods**

```go
// internal/client/user.go
package client

import (
	"context"
	"fmt"
	"net/http"
)

type User struct {
	UserID          string `json:"userId"`
	Username        string `json:"username"`
	Email           string `json:"email"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	IdpID           int64  `json:"idpId"`
	AutoProvisioned bool   `json:"autoProvisioned"`
}

type CreateOrgUserRequest struct {
	Username string  `json:"username"`
	Email    string  `json:"email,omitempty"`
	Name     string  `json:"name,omitempty"`
	Type     string  `json:"type"`
	IdpID    int64   `json:"idpId"`
	RoleIDs  []int64 `json:"roleIds"`
}

type UpdateOrgUserRequest struct {
	AutoProvisioned *bool `json:"autoProvisioned,omitempty"`
}

type listUsersResult struct {
	Users []User `json:"users"`
}

func (c *Client) CreateOrgUser(ctx context.Context, orgID string, in CreateOrgUserRequest) error {
	return c.do(ctx, http.MethodPut, "/org/"+orgID+"/user", in, nil)
}

func (c *Client) ListUsers(ctx context.Context, orgID string) ([]User, error) {
	var out listUsersResult
	if err := c.do(ctx, http.MethodGet, "/org/"+orgID+"/users", nil, &out); err != nil {
		return nil, err
	}
	return out.Users, nil
}

func (c *Client) GetUserByUsername(ctx context.Context, orgID, username string) (*User, error) {
	users, err := c.ListUsers(ctx, orgID)
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		if u.Username == username {
			return &u, nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("user %q not found in org %s", username, orgID)}
}

func (c *Client) GetOrgUser(ctx context.Context, orgID, userID string) (*User, error) {
	var out User
	if err := c.do(ctx, http.MethodGet, "/org/"+orgID+"/user/"+userID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateOrgUser(ctx context.Context, orgID, userID string, in UpdateOrgUserRequest) error {
	return c.do(ctx, http.MethodPost, "/org/"+orgID+"/user/"+userID, in, nil)
}

// AddUserRole adds one role to a user. There is no corresponding "remove
// role" route in the integration API, so this is one-directional.
func (c *Client) AddUserRole(ctx context.Context, roleID int64, userID string) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/role/%d/add/%s", roleID, userID), nil, nil)
}

func (c *Client) DeleteOrgUser(ctx context.Context, orgID, userID string) error {
	return c.do(ctx, http.MethodDelete, "/org/"+orgID+"/user/"+userID, nil, nil)
}
```

- [ ] **Step 4: Run to verify the client tests pass**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -v`
Expected: PASS

- [ ] **Step 5: Write the resource**

```go
// internal/provider/resource_user.go
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
)

func NewUserResource() resource.Resource { return &userResource{} }

type userResource struct {
	client *client.Client
}

type userResourceModel struct {
	OrgID           types.String `tfsdk:"org_id"`
	UserID          types.String `tfsdk:"user_id"`
	Username        types.String `tfsdk:"username"`
	Email           types.String `tfsdk:"email"`
	Name            types.String `tfsdk:"name"`
	IdpID           types.Int64  `tfsdk:"idp_id"`
	RoleIDs         types.List   `tfsdk:"role_ids"`
	AutoProvisioned types.Bool   `tfsdk:"auto_provisioned"`
}

func (r *userResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages an OIDC-backed org user in Pangolin. Internal (password-based) users are not yet supported by the Pangolin integration API. role_ids can only grow after creation: the API has no route to remove a role from a user, so removing an entry from role_ids will produce an error rather than silently doing nothing.",
		Attributes: map[string]schema.Attribute{
			"org_id":    schema.StringAttribute{Required: true, PlanModifiers: replace, Description: "Organization ID this user belongs to."},
			"user_id":   schema.StringAttribute{Computed: true, Description: "Server-generated user ID."},
			"username":  schema.StringAttribute{Required: true, PlanModifiers: replace, Description: "Username, lowercased server-side."},
			"email":     schema.StringAttribute{Optional: true, PlanModifiers: replace, Description: "Email address."},
			"name":      schema.StringAttribute{Optional: true, PlanModifiers: replace, Description: "Display name."},
			"idp_id":    schema.Int64Attribute{Required: true, PlanModifiers: []planmodifier.Int64{}, Description: "Numeric ID of the OIDC identity provider this user authenticates through."},
			"role_ids":  schema.ListAttribute{Required: true, ElementType: types.Int64Type, Description: "Role IDs to grant. Can only grow after creation (see resource description)."},
			"auto_provisioned": schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether this user was auto-provisioned."},
		},
	}
}

func (r *userResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var roleIDs []int64
	resp.Diagnostics.Append(plan.RoleIDs.ElementsAs(ctx, &roleIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.CreateOrgUserRequest{
		Username: plan.Username.ValueString(),
		Type:     "oidc",
		IdpID:    plan.IdpID.ValueInt64(),
		RoleIDs:  roleIDs,
	}
	in.Email = plan.Email.ValueString()
	in.Name = plan.Name.ValueString()

	orgID := plan.OrgID.ValueString()
	if err := r.client.CreateOrgUser(ctx, orgID, in); err != nil {
		resp.Diagnostics.AddError("Error creating user", err.Error())
		return
	}

	// createOrgUser returns no data at all, so look the new user up by
	// username to learn its server-generated ID.
	user, err := r.client.GetUserByUsername(ctx, orgID, plan.Username.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error looking up newly created user", err.Error())
		return
	}

	plan.UserID = types.StringValue(user.UserID)
	plan.AutoProvisioned = types.BoolValue(user.AutoProvisioned)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.GetOrgUser(ctx, state.OrgID.ValueString(), state.UserID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}

	state.Username = types.StringValue(user.Username)
	state.AutoProvisioned = types.BoolValue(user.AutoProvisioned)
	// email, name, idp_id, and role_ids are not re-derived from GetOrgUser
	// (its exact response shape wasn't confirmed from source for these
	// fields) - they are left as whatever is already in state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.AutoProvisioned.IsUnknown() {
		v := plan.AutoProvisioned.ValueBool()
		if err := r.client.UpdateOrgUser(ctx, state.OrgID.ValueString(), state.UserID.ValueString(), client.UpdateOrgUserRequest{AutoProvisioned: &v}); err != nil {
			resp.Diagnostics.AddError("Error updating user", err.Error())
			return
		}
	}

	var planRoleIDs, stateRoleIDs []int64
	resp.Diagnostics.Append(plan.RoleIDs.ElementsAs(ctx, &planRoleIDs, false)...)
	resp.Diagnostics.Append(state.RoleIDs.ElementsAs(ctx, &stateRoleIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	existing := make(map[int64]bool, len(stateRoleIDs))
	for _, id := range stateRoleIDs {
		existing[id] = true
	}
	planned := make(map[int64]bool, len(planRoleIDs))
	for _, id := range planRoleIDs {
		planned[id] = true
		if !existing[id] {
			if err := r.client.AddUserRole(ctx, id, state.UserID.ValueString()); err != nil {
				resp.Diagnostics.AddError("Error adding role to user", err.Error())
				return
			}
		}
	}
	for _, id := range stateRoleIDs {
		if !planned[id] {
			resp.Diagnostics.AddError(
				"Cannot remove role from user",
				fmt.Sprintf("The Pangolin integration API has no route to remove role %d from this user. Destroy and recreate the pangolin_user resource, or remove the role via the Pangolin dashboard and then run terraform apply -refresh-only.", id),
			)
			return
		}
	}

	plan.UserID = state.UserID
	plan.AutoProvisioned = types.BoolValue(plan.AutoProvisioned.ValueBool())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteOrgUser(ctx, state.OrgID.ValueString(), state.UserID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting user", err.Error())
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("user_id"), req, resp)
}
```

- [ ] **Step 6: Write the resource schema test**

```go
// internal/provider/resource_user_test.go
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestUserResource_Schema(t *testing.T) {
	r := NewUserResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"org_id", "user_id", "username", "idp_id", "role_ids"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("expected attribute %q", name)
		}
	}
}
```

- [ ] **Step 7: Write the data source**

```go
// internal/provider/data_source_user.go
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var _ datasource.DataSource = &userDataSource{}

func NewUserDataSource() datasource.DataSource { return &userDataSource{} }

type userDataSource struct {
	client *client.Client
}

type userDataSourceModel struct {
	OrgID    types.String `tfsdk:"org_id"`
	UserID   types.String `tfsdk:"user_id"`
	Username types.String `tfsdk:"username"`
}

func (d *userDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *userDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing org user by org_id and either user_id or username. Exactly one of user_id or username must be set.",
		Attributes: map[string]schema.Attribute{
			"org_id":   schema.StringAttribute{Required: true},
			"user_id":  schema.StringAttribute{Optional: true, Computed: true},
			"username": schema.StringAttribute{Optional: true, Computed: true},
		},
	}
}

func (d *userDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model userDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var user *client.User
	var err error
	if !model.Username.IsNull() {
		user, err = d.client.GetUserByUsername(ctx, model.OrgID.ValueString(), model.Username.ValueString())
	} else {
		user, err = d.client.GetOrgUser(ctx, model.OrgID.ValueString(), model.UserID.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}

	model.UserID = types.StringValue(user.UserID)
	model.Username = types.StringValue(user.Username)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
```

- [ ] **Step 8: Write the data source schema test**

```go
// internal/provider/data_source_user_test.go
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestUserDataSource_Schema(t *testing.T) {
	d := NewUserDataSource()
	var resp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if !resp.Schema.Attributes["org_id"].IsRequired() {
		t.Error("expected org_id to be required")
	}
}
```

- [ ] **Step 9: Register both in the provider**

Add `NewUserResource` to `Resources` and `NewUserDataSource` to `DataSources` in `internal/provider/provider.go`.

- [ ] **Step 10: Write example .tf files**

`examples/resources/pangolin_user/resource.tf`:
```hcl
resource "pangolin_user" "example" {
  org_id   = pangolin_organization.example.org_id
  username = "jane"
  email    = "jane@example.com"
  idp_id   = 1
  role_ids = [pangolin_role.example.role_id]
}
```

`examples/data-sources/pangolin_user/data-source.tf`:
```hcl
data "pangolin_user" "example" {
  org_id   = "acme"
  username = "jane"
}
```

- [ ] **Step 11: Run the full test suite and build**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./... -v && go build ./...`
Expected: PASS, build succeeds

- [ ] **Step 12: Commit**

```bash
cd /home/kiwi/claude/terraform-provider-pangolin
git add internal/client/user.go internal/client/user_test.go \
  internal/provider/resource_user.go internal/provider/resource_user_test.go \
  internal/provider/data_source_user.go internal/provider/data_source_user_test.go \
  internal/provider/provider.go examples/resources/pangolin_user examples/data-sources/pangolin_user
git commit -m "feat: add pangolin_user resource and data source"
```

---

## Task 15: `pangolin_api_key` resource

Confirmed from `server/routers/external.ts`: `PUT /org/:orgId/api-key` (create), `GET /org/:orgId/api-key/:apiKeyId` (read), `DELETE /org/:orgId/api-key/:apiKeyId` (delete), `POST /org/:orgId/api-key/:apiKeyId/actions` (replace the full action list), `GET /org/:orgId/api-key/:apiKeyId/actions` (list current actions). No update route for `name` exists - it's `RequiresReplace`. Confirmed from `createOrgApiKey.ts`: the raw `apiKey` secret is only ever present in the create response (`CreateOrgApiKeyResponse`); it cannot be recovered later, so it's `Computed` + `Sensitive` and preserved across `Read` the same way `pangolin_site`'s `secret` is (Task 10). No data source (matches the approved v1 list - API keys are secrets, not something you look up and reference elsewhere).

**Files:**
- Create: `internal/client/api_key.go` + `internal/client/api_key_test.go`
- Create: `internal/provider/resource_api_key.go` + `internal/provider/resource_api_key_test.go`
- Modify: `internal/provider/provider.go`
- Create: `examples/resources/pangolin_api_key/resource.tf`

**Interfaces:**
- Consumes: `client.Client.do` (Task 4)
- Produces: `client.APIKey`, `client.Client.CreateAPIKey/GetAPIKey/SetAPIKeyActions/ListAPIKeyActions/DeleteAPIKey`, `provider.NewAPIKeyResource()`

- [ ] **Step 1: Write the failing client test**

```go
// internal/client/api_key_test.go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/org/acme/api-key" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"apiKeyId": "k1", "name": "ci", "apiKey": "secret-value", "lastChars": "alue", "createdAt": "2026-01-01T00:00:00Z"},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	key, err := c.CreateAPIKey(context.Background(), "acme", CreateAPIKeyRequest{Name: "ci"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key.APIKey != "secret-value" {
		t.Errorf("expected apiKey 'secret-value', got %q", key.APIKey)
	}
}

func TestSetAPIKeyActions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/org/acme/api-key/k1/actions" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.SetAPIKeyActions(context.Background(), "acme", "k1", []string{"listSites"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/org/acme/api-key/k1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteAPIKey(context.Background(), "acme", "k1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -run 'TestCreateAPIKey|TestSetAPIKeyActions|TestDeleteAPIKey' -v`
Expected: FAIL - undefined symbols

- [ ] **Step 3: Implement the client methods**

```go
// internal/client/api_key.go
package client

import (
	"context"
	"net/http"
)

type APIKey struct {
	APIKeyID  string `json:"apiKeyId"`
	Name      string `json:"name"`
	APIKey    string `json:"apiKey"`
	LastChars string `json:"lastChars"`
	CreatedAt string `json:"createdAt"`
}

type CreateAPIKeyRequest struct {
	Name string `json:"name"`
}

type setAPIKeyActionsRequest struct {
	ActionIDs []string `json:"actionIds"`
}

type apiKeyActionsResult struct {
	ActionIDs []string `json:"actionIds"`
}

func (c *Client) CreateAPIKey(ctx context.Context, orgID string, in CreateAPIKeyRequest) (*APIKey, error) {
	var out APIKey
	if err := c.do(ctx, http.MethodPut, "/org/"+orgID+"/api-key", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetAPIKey(ctx context.Context, orgID, apiKeyID string) (*APIKey, error) {
	var out APIKey
	if err := c.do(ctx, http.MethodGet, "/org/"+orgID+"/api-key/"+apiKeyID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) SetAPIKeyActions(ctx context.Context, orgID, apiKeyID string, actionIDs []string) error {
	return c.do(ctx, http.MethodPost, "/org/"+orgID+"/api-key/"+apiKeyID+"/actions", setAPIKeyActionsRequest{ActionIDs: actionIDs}, nil)
}

func (c *Client) ListAPIKeyActions(ctx context.Context, orgID, apiKeyID string) ([]string, error) {
	var out apiKeyActionsResult
	if err := c.do(ctx, http.MethodGet, "/org/"+orgID+"/api-key/"+apiKeyID+"/actions", nil, &out); err != nil {
		return nil, err
	}
	return out.ActionIDs, nil
}

func (c *Client) DeleteAPIKey(ctx context.Context, orgID, apiKeyID string) error {
	return c.do(ctx, http.MethodDelete, "/org/"+orgID+"/api-key/"+apiKeyID, nil, nil)
}
```

Note: `setApiKeyActions.ts`'s response data is `{}` and `listApiKeyActions.ts`'s exact field name was not read from source - `ActionIDs`/`"actionIds"` is inferred from the request body's own field name (`actionIds`) and the general envelope convention seen everywhere else; verify this against a real response during implementation and adjust the tag if the list endpoint nests it differently.

- [ ] **Step 4: Run to verify the client tests pass**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -v`
Expected: PASS

- [ ] **Step 5: Write the resource**

```go
// internal/provider/resource_api_key.go
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var (
	_ resource.Resource                = &apiKeyResource{}
	_ resource.ResourceWithImportState = &apiKeyResource{}
)

func NewAPIKeyResource() resource.Resource { return &apiKeyResource{} }

type apiKeyResource struct {
	client *client.Client
}

type apiKeyResourceModel struct {
	OrgID     types.String `tfsdk:"org_id"`
	APIKeyID  types.String `tfsdk:"api_key_id"`
	Name      types.String `tfsdk:"name"`
	APIKey    types.String `tfsdk:"api_key"`
	LastChars types.String `tfsdk:"last_chars"`
	ActionIDs types.List   `tfsdk:"action_ids"`
}

func (r *apiKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *apiKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an organization-scoped Pangolin API key. The secret key value is only ever available at creation time.",
		Attributes: map[string]schema.Attribute{
			"org_id":      schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Description: "Organization ID this key belongs to."},
			"api_key_id":  schema.StringAttribute{Computed: true, Description: "Server-generated key ID."},
			"name":        schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Description: "Display name for the key."},
			"api_key":     schema.StringAttribute{Computed: true, Sensitive: true, Description: "The secret key value. Only ever populated from the create response; not recoverable afterward."},
			"last_chars":  schema.StringAttribute{Computed: true, Description: "Last 4 characters of the key, for identification."},
			"action_ids":  schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType, Description: "Actions this key is permitted to perform. Replaces the full list on every change."},
		},
	}
}

func (r *apiKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *apiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := plan.OrgID.ValueString()
	key, err := r.client.CreateAPIKey(ctx, orgID, client.CreateAPIKeyRequest{Name: plan.Name.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error creating API key", err.Error())
		return
	}

	plan.APIKeyID = types.StringValue(key.APIKeyID)
	plan.APIKey = types.StringValue(key.APIKey)
	plan.LastChars = types.StringValue(key.LastChars)

	if !plan.ActionIDs.IsUnknown() && !plan.ActionIDs.IsNull() {
		var actionIDs []string
		resp.Diagnostics.Append(plan.ActionIDs.ElementsAs(ctx, &actionIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if err := r.client.SetAPIKeyActions(ctx, orgID, key.APIKeyID, actionIDs); err != nil {
			resp.Diagnostics.AddError("Error setting API key actions", err.Error())
			return
		}
	} else {
		plan.ActionIDs = types.ListValueMust(types.StringType, nil)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrgID.ValueString()
	key, err := r.client.GetAPIKey(ctx, orgID, state.APIKeyID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading API key", err.Error())
		return
	}

	apiKeyValue := state.APIKey // write-once secret, not returned by GetAPIKey
	state.Name = types.StringValue(key.Name)
	state.LastChars = types.StringValue(key.LastChars)
	state.APIKey = apiKeyValue

	actionIDs, err := r.client.ListAPIKeyActions(ctx, orgID, state.APIKeyID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading API key actions", err.Error())
		return
	}
	listValue, diags := types.ListValueFrom(ctx, types.StringType, actionIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.ActionIDs = listValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *apiKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state apiKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var actionIDs []string
	resp.Diagnostics.Append(plan.ActionIDs.ElementsAs(ctx, &actionIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrgID.ValueString()
	if err := r.client.SetAPIKeyActions(ctx, orgID, state.APIKeyID.ValueString(), actionIDs); err != nil {
		resp.Diagnostics.AddError("Error updating API key actions", err.Error())
		return
	}

	plan.APIKeyID = state.APIKeyID
	plan.APIKey = state.APIKey
	plan.LastChars = state.LastChars
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteAPIKey(ctx, state.OrgID.ValueString(), state.APIKeyID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting API key", err.Error())
	}
}

func (r *apiKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("api_key_id"), req, resp)
}
```

- [ ] **Step 6: Write the resource schema test**

```go
// internal/provider/resource_api_key_test.go
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestAPIKeyResource_Schema(t *testing.T) {
	r := NewAPIKeyResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"org_id", "api_key_id", "name", "api_key", "action_ids"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("expected attribute %q", name)
		}
	}
	if !resp.Schema.Attributes["api_key"].IsSensitive() {
		t.Error("expected api_key to be sensitive")
	}
}
```

- [ ] **Step 7: Register in the provider**

Add `NewAPIKeyResource` to `Resources` in `internal/provider/provider.go`.

- [ ] **Step 8: Write the example .tf file**

`examples/resources/pangolin_api_key/resource.tf`:
```hcl
resource "pangolin_api_key" "ci" {
  org_id     = pangolin_organization.example.org_id
  name       = "ci-deploys"
  action_ids = ["listSites", "createResource"]
}

output "ci_api_key" {
  value     = pangolin_api_key.ci.api_key
  sensitive = true
}
```

- [ ] **Step 9: Run the full test suite and build**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./... -v && go build ./...`
Expected: PASS, build succeeds

- [ ] **Step 10: Commit**

```bash
cd /home/kiwi/claude/terraform-provider-pangolin
git add internal/client/api_key.go internal/client/api_key_test.go \
  internal/provider/resource_api_key.go internal/provider/resource_api_key_test.go \
  internal/provider/provider.go examples/resources/pangolin_api_key
git commit -m "feat: add pangolin_api_key resource"
```

---

## Task 16: `pangolin_role_resource_grant` resource

Confirmed from `addRoleToResource.ts`/`removeRoleFromResource.ts` and their mount points: `POST /resource/{resourceId}/roles/add` (body `{roleId}`) grants, `POST /resource/{resourceId}/roles/remove` (body `{roleId}`) revokes. Confirmed from `external.ts`: `GET /resource/:resourceId/roles` lists granted role IDs, used here for `Read` since there's no single-grant GET. No update semantics exist for a grant (it's binary: granted or not) - `Update` is unreachable because every attribute is `RequiresReplace`.

**Files:**
- Create: `internal/client/role_resource_grant.go` + `internal/client/role_resource_grant_test.go`
- Create: `internal/provider/resource_role_resource_grant.go` + `internal/provider/resource_role_resource_grant_test.go`
- Modify: `internal/provider/provider.go`
- Create: `examples/resources/pangolin_role_resource_grant/resource.tf`

**Interfaces:**
- Consumes: `client.Client.do` (Task 4)
- Produces: `client.Client.AddRoleToResource/RemoveRoleFromResource/ListResourceRoles`, `provider.NewRoleResourceGrantResource()`

- [ ] **Step 1: Write the failing client test**

```go
// internal/client/role_resource_grant_test.go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddRoleToResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/resource/5/roles/add" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.AddRoleToResource(context.Background(), 5, 3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListResourceRoles_ContainsGrantedRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"roles": []map[string]any{{"roleId": float64(3)}}},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	roleIDs, err := c.ListResourceRoles(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(roleIDs) != 1 || roleIDs[0] != 3 {
		t.Errorf("expected [3], got %v", roleIDs)
	}
}

func TestRemoveRoleFromResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/resource/5/roles/remove" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.RemoveRoleFromResource(context.Background(), 5, 3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -run 'TestAddRoleToResource|TestListResourceRoles|TestRemoveRoleFromResource' -v`
Expected: FAIL - undefined symbols

- [ ] **Step 3: Implement the client methods**

```go
// internal/client/role_resource_grant.go
package client

import (
	"context"
	"fmt"
	"net/http"
)

type roleIDBody struct {
	RoleID int64 `json:"roleId"`
}

type listResourceRolesResult struct {
	Roles []struct {
		RoleID int64 `json:"roleId"`
	} `json:"roles"`
}

func (c *Client) AddRoleToResource(ctx context.Context, resourceID, roleID int64) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/resource/%d/roles/add", resourceID), roleIDBody{RoleID: roleID}, nil)
}

func (c *Client) RemoveRoleFromResource(ctx context.Context, resourceID, roleID int64) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/resource/%d/roles/remove", resourceID), roleIDBody{RoleID: roleID}, nil)
}

func (c *Client) ListResourceRoles(ctx context.Context, resourceID int64) ([]int64, error) {
	var out listResourceRolesResult
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/resource/%d/roles", resourceID), nil, &out); err != nil {
		return nil, err
	}
	ids := make([]int64, len(out.Roles))
	for i, role := range out.Roles {
		ids[i] = role.RoleID
	}
	return ids, nil
}
```

- [ ] **Step 4: Run to verify the client tests pass**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -v`
Expected: PASS

- [ ] **Step 5: Write the resource**

```go
// internal/provider/resource_role_resource_grant.go
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var _ resource.Resource = &roleResourceGrantResource{}

func NewRoleResourceGrantResource() resource.Resource { return &roleResourceGrantResource{} }

type roleResourceGrantResource struct {
	client *client.Client
}

type roleResourceGrantModel struct {
	ResourceID types.Int64 `tfsdk:"resource_id"`
	RoleID     types.Int64 `tfsdk:"role_id"`
}

func (r *roleResourceGrantResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_resource_grant"
}

func (r *roleResourceGrantResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.Int64{int64planmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Grants a role access to a Pangolin resource. There is no update: changing either ID replaces the grant.",
		Attributes: map[string]schema.Attribute{
			"resource_id": schema.Int64Attribute{Required: true, PlanModifiers: replace, Description: "Resource ID to grant access to."},
			"role_id":     schema.Int64Attribute{Required: true, PlanModifiers: replace, Description: "Role ID being granted access."},
		},
	}
}

func (r *roleResourceGrantResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *roleResourceGrantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleResourceGrantModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.AddRoleToResource(ctx, plan.ResourceID.ValueInt64(), plan.RoleID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Error granting role access to resource", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleResourceGrantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleResourceGrantModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleIDs, err := r.client.ListResourceRoles(ctx, state.ResourceID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error reading resource roles", err.Error())
		return
	}

	found := false
	for _, id := range roleIDs {
		if id == state.RoleID.ValueInt64() {
			found = true
			break
		}
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *roleResourceGrantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan roleResourceGrantModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleResourceGrantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleResourceGrantModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveRoleFromResource(ctx, state.ResourceID.ValueInt64(), state.RoleID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Error revoking role access from resource", err.Error())
	}
}
```

No `ImportState`: both IDs are required attributes with no computed ID of their own, so `terraform import pangolin_role_resource_grant.example <resource_id>/<role_id>` isn't wired up in v1 - grants are cheap to recreate from configuration instead. (v1.x can add `ResourceWithImportState` parsing a composite ID if this turns out to matter in practice.)

- [ ] **Step 6: Write the resource schema test**

```go
// internal/provider/resource_role_resource_grant_test.go
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestRoleResourceGrantResource_Schema(t *testing.T) {
	r := NewRoleResourceGrantResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"resource_id", "role_id"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok || !attr.IsRequired() {
			t.Errorf("expected required attribute %q", name)
		}
	}
}
```

- [ ] **Step 7: Register in the provider**

Add `NewRoleResourceGrantResource` to `Resources` in `internal/provider/provider.go`.

- [ ] **Step 8: Write the example .tf file**

`examples/resources/pangolin_role_resource_grant/resource.tf`:
```hcl
resource "pangolin_role_resource_grant" "example" {
  resource_id = pangolin_resource.example.resource_id
  role_id     = pangolin_role.example.role_id
}
```

- [ ] **Step 9: Run the full test suite and build**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./... -v && go build ./...`
Expected: PASS, build succeeds

- [ ] **Step 10: Commit**

```bash
cd /home/kiwi/claude/terraform-provider-pangolin
git add internal/client/role_resource_grant.go internal/client/role_resource_grant_test.go \
  internal/provider/resource_role_resource_grant.go internal/provider/resource_role_resource_grant_test.go \
  internal/provider/provider.go examples/resources/pangolin_role_resource_grant
git commit -m "feat: add pangolin_role_resource_grant resource"
```

---

## Task 17: `pangolin_user_resource_grant` resource

Identical shape to Task 16, confirmed from `addUserToResource.ts`/`removeUserFromResource.ts` and `external.ts`: `POST /resource/{resourceId}/users/add` (body `{userId}`), `POST /resource/{resourceId}/users/remove` (body `{userId}`), `GET /resource/:resourceId/users` (list, for `Read`).

**Files:**
- Create: `internal/client/user_resource_grant.go` + `internal/client/user_resource_grant_test.go`
- Create: `internal/provider/resource_user_resource_grant.go` + `internal/provider/resource_user_resource_grant_test.go`
- Modify: `internal/provider/provider.go`
- Create: `examples/resources/pangolin_user_resource_grant/resource.tf`

**Interfaces:**
- Consumes: `client.Client.do` (Task 4)
- Produces: `client.Client.AddUserToResource/RemoveUserFromResource/ListResourceUsers`, `provider.NewUserResourceGrantResource()`

- [ ] **Step 1: Write the failing client test**

```go
// internal/client/user_resource_grant_test.go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddUserToResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/resource/5/users/add" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.AddUserToResource(context.Background(), 5, "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListResourceUsers_ContainsGrantedUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"users": []map[string]any{{"userId": "u1"}}},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	userIDs, err := c.ListResourceUsers(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(userIDs) != 1 || userIDs[0] != "u1" {
		t.Errorf("expected [\"u1\"], got %v", userIDs)
	}
}

func TestRemoveUserFromResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/resource/5/users/remove" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.RemoveUserFromResource(context.Background(), 5, "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -run 'TestAddUserToResource|TestListResourceUsers|TestRemoveUserFromResource' -v`
Expected: FAIL - undefined symbols

- [ ] **Step 3: Implement the client methods**

```go
// internal/client/user_resource_grant.go
package client

import (
	"context"
	"fmt"
	"net/http"
)

type userIDBody struct {
	UserID string `json:"userId"`
}

type listResourceUsersResult struct {
	Users []struct {
		UserID string `json:"userId"`
	} `json:"users"`
}

func (c *Client) AddUserToResource(ctx context.Context, resourceID int64, userID string) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/resource/%d/users/add", resourceID), userIDBody{UserID: userID}, nil)
}

func (c *Client) RemoveUserFromResource(ctx context.Context, resourceID int64, userID string) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/resource/%d/users/remove", resourceID), userIDBody{UserID: userID}, nil)
}

func (c *Client) ListResourceUsers(ctx context.Context, resourceID int64) ([]string, error) {
	var out listResourceUsersResult
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/resource/%d/users", resourceID), nil, &out); err != nil {
		return nil, err
	}
	ids := make([]string, len(out.Users))
	for i, u := range out.Users {
		ids[i] = u.UserID
	}
	return ids, nil
}
```

- [ ] **Step 4: Run to verify the client tests pass**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./internal/client/... -v`
Expected: PASS

- [ ] **Step 5: Write the resource**

```go
// internal/provider/resource_user_resource_grant.go
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

var _ resource.Resource = &userResourceGrantResource{}

func NewUserResourceGrantResource() resource.Resource { return &userResourceGrantResource{} }

type userResourceGrantResource struct {
	client *client.Client
}

type userResourceGrantModel struct {
	ResourceID types.Int64  `tfsdk:"resource_id"`
	UserID     types.String `tfsdk:"user_id"`
}

func (r *userResourceGrantResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_resource_grant"
}

func (r *userResourceGrantResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Grants a user access to a Pangolin resource. There is no update: changing either ID replaces the grant.",
		Attributes: map[string]schema.Attribute{
			"resource_id": schema.Int64Attribute{Required: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}, Description: "Resource ID to grant access to."},
			"user_id":     schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Description: "User ID being granted access."},
		},
	}
}

func (r *userResourceGrantResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *userResourceGrantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userResourceGrantModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.AddUserToResource(ctx, plan.ResourceID.ValueInt64(), plan.UserID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error granting user access to resource", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResourceGrantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userResourceGrantModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userIDs, err := r.client.ListResourceUsers(ctx, state.ResourceID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error reading resource users", err.Error())
		return
	}

	found := false
	for _, id := range userIDs {
		if id == state.UserID.ValueString() {
			found = true
			break
		}
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *userResourceGrantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userResourceGrantModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResourceGrantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userResourceGrantModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveUserFromResource(ctx, state.ResourceID.ValueInt64(), state.UserID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error revoking user access from resource", err.Error())
	}
}
```

- [ ] **Step 6: Write the resource schema test**

```go
// internal/provider/resource_user_resource_grant_test.go
package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestUserResourceGrantResource_Schema(t *testing.T) {
	r := NewUserResourceGrantResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"resource_id", "user_id"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok || !attr.IsRequired() {
			t.Errorf("expected required attribute %q", name)
		}
	}
}
```

- [ ] **Step 7: Register in the provider**

Add `NewUserResourceGrantResource` to `Resources` in `internal/provider/provider.go`.

- [ ] **Step 8: Write the example .tf file**

`examples/resources/pangolin_user_resource_grant/resource.tf`:
```hcl
resource "pangolin_user_resource_grant" "example" {
  resource_id = pangolin_resource.example.resource_id
  user_id     = pangolin_user.example.user_id
}
```

- [ ] **Step 9: Run the full test suite and build**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go test ./... -v && go build ./...`
Expected: PASS, build succeeds

- [ ] **Step 10: Commit**

```bash
cd /home/kiwi/claude/terraform-provider-pangolin
git add internal/client/user_resource_grant.go internal/client/user_resource_grant_test.go \
  internal/provider/resource_user_resource_grant.go internal/provider/resource_user_resource_grant_test.go \
  internal/provider/provider.go examples/resources/pangolin_user_resource_grant
git commit -m "feat: add pangolin_user_resource_grant resource"
```

---

## Task 18: Docs generation, GitHub repo creation, and first push

**Files:**
- Create: `docs/` (generated, not hand-written)
- Modify: `README.md` (finalize with real registry link once known)

**Interfaces:**
- Consumes: every resource/data source registered by Tasks 8-17
- Produces: a live `RichardBurgoyne/terraform-provider-pangolin` GitHub repo with CI green on `main`

- [ ] **Step 1: Install tfplugindocs and generate docs**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest && tfplugindocs generate`
Expected: `docs/index.md`, `docs/resources/*.md`, `docs/data-sources/*.md` created from the schema `Description` fields and `examples/` files written throughout Tasks 8-17.

- [ ] **Step 2: Review generated docs for accuracy**

Read through `docs/index.md` and spot-check two or three resource docs against the actual schema in their `.go` files. Confirm the v1.x deferrals (raw resources, inference mode, health checks, SSH sudo lists, IdP-backed users beyond `idp_id`, role removal from users) are mentioned somewhere reachable from `docs/index.md` - either in the provider description or a short "Roadmap" section in `README.md`. Add one if it's missing.

- [ ] **Step 3: Run the full local verification one more time**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && gofmt -l . && go vet ./... && go test -race -cover ./... && go build ./...`
Expected: `gofmt -l .` prints nothing, everything else passes

- [ ] **Step 4: Commit the generated docs**

```bash
cd /home/kiwi/claude/terraform-provider-pangolin
git add docs/
git commit -m "docs: generate provider documentation"
```

- [ ] **Step 5: Create the GitHub repo (ask the user to confirm before this step - it's the first externally-visible action in this plan)**

Run: `gh repo create RichardBurgoyne/terraform-provider-pangolin --public --source=/home/kiwi/claude/terraform-provider-pangolin --remote=origin --description "Terraform provider for Pangolin"`
Expected: repo created, `origin` remote added

- [ ] **Step 6: Push**

Run: `cd /home/kiwi/claude/terraform-provider-pangolin && git push -u origin main`
Expected: push succeeds, `main` branch visible on GitHub

- [ ] **Step 7: Verify the Test workflow runs and passes on GitHub**

Run: `gh run list --repo RichardBurgoyne/terraform-provider-pangolin --limit 5` (after waiting for the push-triggered run to start)
Expected: the "Test" workflow appears and eventually shows a green conclusion. If it fails on the `go-provider-test` composite action, the most likely cause is the reusable-workflows tag mismatch flagged in Task 2 Step 4 - confirm `.github/workflows/test.yml` references the tag that actually exists on `RichardBurgoyne/reusable-workflows`.

- [ ] **Step 8: Report remaining manual setup to the user**

This plan does not automate (and should not, without explicit confirmation for each):
1. Generating the GPG signing key and adding `GPG_PRIVATE_KEY`/`GPG_PASSPHRASE` as repo secrets (Task 7's README section documents the steps; actually generating and storing a private key is a one-time credential-creation action worth doing deliberately, not as an unattended plan step).
2. Cutting the `v0.1.0` tag and pushing it to trigger the first release.
3. Connecting the repo to `registry.terraform.io` under the `RichardBurgoyne` namespace after the first signed release exists.
4. Uploading the GPG public key to the Terraform Registry publisher settings.

Tell the user these four steps are ready whenever they want to do them, and that everything up to a working, tested, buildable provider is done.

---

## Self-Review Notes (for whoever executes this plan)

**Spec coverage:** every v1 resource/data source from the design spec is covered (Tasks 8-17), with `pangolin_organization` and `pangolin_domain` promoted from data-source-only to full resources per the spec's own explicit contingency for exactly this situation (real API evidence showing full CRUD exists). The reusable-actions split (Task 1-2), CI/release automation (Tasks 6-7), and docs/publishing (Task 18) match the spec's sections 1 and 3.

**Deviations from the original spec, and why:**
- `pangolin_organization`: promoted resource, not just data source (confirmed `PUT /org`, `DELETE /org/:orgId` exist).
- `pangolin_domain`: promoted resource, not just data source (confirmed `PUT /org/:orgId/domain` exists), update limited to `cert_resolver`/`prefer_wildcard_cert` only (confirmed from `updateDomain.ts`).
- `pangolin_resource`: scoped to `http`/`ssh`/`rdp`/`vnc` modes only; raw `tcp`/`udp` and `inference` modes, plus most policy/access-control fields, deferred to v1.x. This is a real narrowing of scope discovered only once the actual Zod schemas were read - the design spec's field list for this resource was a guess; this plan's is verified.
- `pangolin_target`: all health-check fields deferred to v1.x.
- `pangolin_role`: `sshSudoCommands`, `sshUnixGroups`, `sshCreateHomeDir` deferred to v1.x (unconfirmed wire format).
- `pangolin_user`: locked to `type = "oidc"` (internal users rejected server-side today); `role_ids` can only grow after creation (no role-removal route exists in the integration API) - `Update` returns an explicit error rather than silently failing if the plan tries to shrink the list.

**Ambiguity flagged for the executor to resolve during implementation, not guessed at in this plan:**
- `DELETE`/`GET` verbs for a small number of routes were confirmed directly from `external.ts`, so there should be no remaining guesses on HTTP verb+path for any v1 endpoint - if `go test` against a real instance disagrees with any path in this plan, trust the real instance and fix the plan's assumption, don't paper over it.
- `listApiKeyActions`'s exact response field name (Task 15).
- `sshSudoCommands`/`sshUnixGroups` wire format if they're ever promoted out of v1.x.
- `GetOrgUser`'s exact response shape beyond the columns `createOrgUser.ts` itself writes (Task 14).

**Type consistency:** `client.Client.do`'s signature (`ctx, method, path string, body, out any`) is used identically across every client file in Tasks 8-17. `client.IsNotFound(err)` and `client.APIError` (Task 4) are the only error-inspection primitives used throughout - no task invents a second error type. Every resource's `Configure` follows the exact same four-line type-assertion pattern established in Task 8, so a reviewer skimming Tasks 9-17 can confirm at a glance they match without re-reading Task 8's full listing each time.
