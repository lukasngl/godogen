# Godogen Language Server

Language Server Protocol (LSP) implementation for [Godogen](https://github.com/lukasngl/godogen), providing IDE features for Gherkin feature files and Go step definitions.

## Features

- **Diagnostics**: Validation errors for step definitions (undefined, ambiguous, unused, duplicate steps)
- **Go to Definition**: Navigate from feature steps to Go step definitions
- **Go to Implementation**: Jump to step implementation from pattern comments
- **Find References**: Find all feature steps that use a step definition
- **Hover**: Show step definition details and usage information
- **Document Symbols**: Navigate feature scenarios and step definitions
- **Code Actions**: Quick fixes for validation errors
- **Completion**: Gherkin keyword completion with multi-language support

## Installation

```bash
go install github.com/lukasngl/godogen/godogen-language-server@latest
```

## Usage

### Language Server Mode

Start the language server for IDE integration:

```bash
# Default (errors only)
godogen-language-server stdio

# With debug logging
godogen-language-server stdio --debug

# Log to file
godogen-language-server stdio --log-file /tmp/lsp.log
```

#### Logging Options

- `--debug`, `-d`: Enable debug logging
- `--log-file <path>`: Write logs to file (default: stderr)
- Output format: JSON for structured filtering

### CLI Mode

The language server also provides CLI commands for batch analysis, CI/CD integration, and scripting.

#### Global Flags

All CLI commands support these flags:

| Flag | Description | Default |
|------|-------------|---------|
| `--root <dir>` | Workspace root directory | `.` (current directory) |
| `--config <file>` | Config file path | `.godogen-language-server.json` in root |
| `--format, -f <fmt>` | Output format: `text` or `json` | `text` |

#### diagnose

Run diagnostics on the workspace to find issues.

```bash
godogen-language-server diagnose [--severity error|warning|hint|all]
```

Reports:
- **Undefined steps** (error): Feature steps without matching definitions
- **Ambiguous steps** (warning): Steps matching multiple definitions
- **Duplicate definitions** (error): Same pattern defined multiple times
- **Unused definitions** (hint): Step definitions not used in any feature
- **Invalid patterns** (error): Patterns missing `^`/`$` anchors or invalid regex

**Examples:**

```bash
# Find all issues
godogen-language-server diagnose

# Only errors (good for CI)
godogen-language-server diagnose --severity error

# JSON output for parsing
godogen-language-server diagnose --format json

# Analyze specific directory
godogen-language-server diagnose --root ./tests/bdd
```

**Output (text):**
```
features/login.feature:10:5: error: No step definition found for: When I click login
steps/auth_steps.go:25:1: hint: Step definition is not used in any feature file

Summary: 1 errors, 0 warnings, 1 hints
```

**Output (json):**
```json
{
  "diagnostics": [
    {
      "file": "features/login.feature",
      "line": 10,
      "column": 5,
      "severity": "error",
      "message": "No step definition found for: When I click login"
    }
  ],
  "summary": {
    "errors": 1,
    "warnings": 0,
    "hints": 1,
    "total": 2
  }
}
```

#### list-steps

List all step definitions in the workspace.

```bash
godogen-language-server list-steps [--kind Given|When|Then|Step|all]
```

**Examples:**

```bash
# List all steps
godogen-language-server list-steps

# Only When steps
godogen-language-server list-steps --kind When

# JSON output
godogen-language-server list-steps --format json
```

**Output (text):**
```
Given: ^I am logged in$           (steps/auth_steps.go:15)
When:  ^I click "([^"]*)"$        (steps/ui_steps.go:42)
Then:  ^I should see "([^"]*)"$   (steps/ui_steps.go:78)

Total: 3 step definitions
```

#### find-definition

Find the step definition for a feature step.

```bash
godogen-language-server find-definition <file:line>
```

**Example:**

```bash
godogen-language-server find-definition features/login.feature:10
```

**Output:**
```
steps/auth_steps.go:15:1
```

#### find-references

Find all feature steps that reference a step definition.

```bash
godogen-language-server find-references <file:line:column>
```

The cursor should be on a `//godogen:` comment or function name.

**Example:**

```bash
godogen-language-server find-references steps/auth_steps.go:15:1
```

**Output:**
```
features/login.feature:10:5
features/signup.feature:8:5
```

#### hover

Get hover information for a position.

```bash
godogen-language-server hover <file:line:column>
```

**Example:**

```bash
godogen-language-server hover features/login.feature:10:5
```

**Output:**
```
**Step Definition**

```go
func (s *Suite) iClickLogin(ctx context.Context) error
```

**File:** auth_steps.go:15
**Pattern:** `^I click login$`
```

#### symbols

Get document symbols for a file.

```bash
godogen-language-server symbols <file>
```

**Examples:**

```bash
# Feature file symbols (scenarios, steps)
godogen-language-server symbols features/login.feature

# Go file symbols (step definitions, hooks)
godogen-language-server symbols steps/auth_steps.go
```

**Output (feature file):**
```
Feature: User Login  (Module:1)
  Scenario: Successful login  (Method:3)
    Given I am on the login page  (Property:4)
    When I click login  (Property:5)
    Then I should see the dashboard  (Property:6)
```

### CI/CD Integration

Use the CLI for continuous integration:

```bash
# Fail if there are any errors
godogen-language-server diagnose --severity error
if [ $? -ne 0 ]; then
  echo "BDD validation failed"
  exit 1
fi
```

Or parse JSON output:

```bash
ERRORS=$(godogen-language-server diagnose --format json | jq '.summary.errors')
if [ "$ERRORS" -gt 0 ]; then
  echo "Found $ERRORS errors"
  exit 1
fi
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

## Claude Code Integration

A Claude Code agent configuration is available for AI-assisted BDD development. The agent can analyze your test suite, find issues, and help navigate between feature files and step definitions.

See [docs/claude-agent.md](docs/claude-agent.md) for the agent configuration. To use it:

1. Copy to your global Claude agents directory:
   ```bash
   cp docs/claude-agent.md ~/.claude/agents/godogen-guide.md
   ```

2. Restart Claude Code to load the agent

3. Ask Claude to analyze your BDD test suite:
   - "Find all undefined steps in my feature files"
   - "Which step definitions are unused?"
   - "List all When steps in the project"

## License

MIT
