# Godogen Installation

## Go Install

```bash
# Code generator
go install github.com/lukasngl/godogen@latest

# Language server (for IDE integration and CLI analysis)
go install github.com/lukasngl/godogen/godogen-language-server@latest

# Linter
go install github.com/lukasngl/godogen/godogen-lint@latest
```

Ensure `$GOPATH/bin` (typically `~/go/bin`) is in your `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Go Tool Directive (Go 1.24+)

Add tools to your `go.mod`:

```go
tool (
    github.com/lukasngl/godogen
    github.com/lukasngl/godogen/godogen-language-server
    github.com/lukasngl/godogen/godogen-lint
)
```

Then run via `go tool`:

```bash
go tool godogen
go tool godogen-language-server diagnose
go tool godogen-lint ./...
```

## Nix Flake

Run directly without installing:

```bash
# All tools
nix run github:lukasngl/godogen

# Specific tool
nix run github:lukasngl/godogen#godogen
nix run github:lukasngl/godogen#godogen-language-server
nix run github:lukasngl/godogen#godogen-lint
```

Use in a project via `.envrc` (with direnv):

```bash
use flake github:lukasngl/godogen#tools
```

Add to your flake inputs:

```nix
{
  inputs.godogen.url = "github:lukasngl/godogen";

  # Use in devShell
  devShells.default = pkgs.mkShell {
    packages = [ inputs.godogen.packages.${system}.default ];
  };
}
```

Or use in NixOS/home-manager configuration:

```nix
environment.systemPackages = [ inputs.godogen.packages.${system}.default ];
```

## Claude Code Plugin

```bash
# Add the godogen marketplace
/plugin marketplace add lukasngl/godogen

# Install the plugin
/plugin install godogen@lukasngl/godogen

# Update to latest version
/plugin update godogen@lukasngl/godogen
```

The plugin provides:

- LSP integration for `.feature` files (real-time diagnostics)
- `godogen-guide` agent for BDD analysis
- `/godogen` skill (this reference)

## Editor Setup

### Neovim (nvim-lspconfig)

```lua
local lspconfig = require('lspconfig')
local configs = require('lspconfig.configs')

if not configs.godogen then
  configs.godogen = {
    default_config = {
      cmd = { 'godogen-language-server', 'stdio' },
      filetypes = { 'feature', 'gherkin' },
      root_dir = lspconfig.util.root_pattern('.git', 'go.mod'),
    },
  }
end

lspconfig.godogen.setup({
  init_options = {
    stepPatterns = { "**/*_steps.go", "**/*.feature" }
  }
})
```

### Helix

Add to `languages.toml`:

```toml
[[language]]
name = "gherkin"
scope = "source.gherkin"
file-types = ["feature"]
language-servers = ["godogen-language-server"]

[language-server.godogen-language-server]
command = "godogen-language-server"
args = ["stdio"]
```

### VS Code

Extension coming soon. For now, use a generic LSP client extension.
