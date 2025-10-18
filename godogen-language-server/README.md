# Godogen Language Server

Language Server Protocol (LSP) implementation for [Godogen](https://github.com/lukasngl/godogen), providing IDE features for Gherkin feature files and Go step definitions.

## Features

- **Diagnostics**: Validation errors for step definitions
- **Go to Definition**: Navigate from feature steps to Go step definitions
- **Go to Implementation**: Jump to step implementation from pattern comments
- **Find References**: Find all feature steps that use a step definition
- **Code Actions**: Quick fixes for validation errors
- **Completion**: Gherkin keyword completion with multi-language support

## Installation

```bash
go install github.com/lukasngl/godogen/godogen-language-server@latest
```

## Usage

Start the language server using stdio transport:

```bash
godogen-language-server stdio
```

## Configuration

The language server can be configured in two ways:

### 1. Config File (Recommended for Teams)

Create a `.godogen-language-server.json` file in your workspace root:

```json
{
  "stepPatterns": [
    "**/*_steps.go",
    "**/*.feature",
    "../shared-steps/**/*.go"
  ]
}
```

### 2. LSP Initialization Options (User Settings)

Configure via your editor's LSP client settings. These override the config file.

**VS Code** (settings.json):
```json
{
  "godogen-language-server.stepPatterns": [
    "**/*_steps.go",
    "**/*.feature"
  ]
}
```

**Neovim** (lspconfig):
```lua
require('lspconfig').godogen.setup({
  init_options = {
    stepPatterns = {
      "**/*_steps.go",
      "**/*.feature",
    }
  }
})
```

### Configuration Options

#### `stepPatterns`

List of glob patterns for discovering step definitions and feature files.

- Supports `**` for recursive directory matching
- Can be absolute or relative to workspace root
- Relative paths like `../shared-steps/**/*.go` allow discovering steps outside the workspace

**Default**: `["**"]` (watches everything in workspace)

**Examples**:
- `**/*_steps.go` - All Go files ending with `_steps.go`
- `**/*.feature` - All feature files
- `tests/**/*.go` - All Go files in tests directory
- `../shared-steps/**/*.go` - Step definitions in parent directory

### Configuration Precedence

1. **LSP initialization options** (highest priority - user's editor settings)
2. **Config file** (`.godogen-language-server.json` in workspace root)
3. **Default values** (lowest priority)

## Editor Setup

### VS Code

Install the Godogen extension (coming soon) or configure a generic LSP client.

### Neovim

Using [nvim-lspconfig](https://github.com/neovim/nvim-lspconfig):

```lua
local lspconfig = require('lspconfig')
local configs = require('lspconfig.configs')

-- Define godogen LSP if not already defined
if not configs.godogen then
  configs.godogen = {
    default_config = {
      cmd = { 'godogen-language-server', 'stdio' },
      filetypes = { 'feature', 'gherkin' },
      root_dir = lspconfig.util.root_pattern('.git', 'go.mod'),
      settings = {},
    },
  }
end

-- Setup godogen LSP
lspconfig.godogen.setup({
  init_options = {
    stepPatterns = {
      "**/*_steps.go",
      "**/*.feature",
    }
  }
})
```

### Helix

Add to your `languages.toml`:

```toml
[[language]]
name = "gherkin"
scope = "source.gherkin"
file-types = ["feature"]
language-servers = ["godogen-language-server"]

[language-server.godogen-language-server]
command = "godogen-language-server"
args = ["stdio"]

[language-server.godogen-language-server.config]
stepPatterns = ["**/*_steps.go", "**/*.feature"]
```

## Development

### Building

```bash
go build ./cmd/godogen-language-server
```

### Testing

```bash
go test ./...
```

### Generating Step Definitions

The test suite uses Godogen itself:

```bash
go generate ./testsuite
```

## Architecture

### File Indexing

The language server maintains separate indexes for:
- **Workspace files**: Files open in the editor (from LSP `didOpen`/`didChange`)
- **Disk files**: Files discovered via glob patterns

Workspace versions always take precedence over disk versions.

### File Discovery

Uses [doublestar](https://github.com/bmatcuk/doublestar) for glob pattern matching with `**` support. The file watcher:
- Discovers existing files matching patterns
- Watches directories for changes
- Handles file creation, modification, and deletion
- Supports multiple watch directories (e.g., for external step libraries)

## License

MIT
