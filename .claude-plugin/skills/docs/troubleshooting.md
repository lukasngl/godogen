# Godogen Troubleshooting

## No step definitions found

The `.godogen-language-server.json` config file is **optional**. By default (without config), the language server watches all files in the workspace (`**`).

If steps aren't being found:

1. Check Go files have `//godogen:` comments (no space after `//`)
2. Ensure you're running from the correct workspace root (`--root` flag)
3. If using a config file, verify `stepPatterns` includes your files:
   ```json
   {
     "stepPatterns": ["**/*_steps.go", "**/*.feature"]
   }
   ```
4. Try removing the config file to use default behavior

## Pattern validation errors

- Patterns must start with `^` and end with `$`
- Escape regex special characters: `\.`, `\(`, `\)`, `\[`, `\]`
- Test pattern: `echo "step text" | grep -E '^pattern$'`

## Step shows unused but is used

- Check feature file is in `stepPatterns` (or default `**`)
- Verify step text matches pattern exactly (case-sensitive)
- Check capture groups match the step text

## And/But keyword mismatch

`And` and `But` inherit the keyword type from the previous step. This is a common source of bugs:

```gherkin
Given I am logged in
And I click the login button    # This is a "Given", not a "When"!
```

If your step is defined as `//godogen:when`, it won't match because `And` inherits `Given`.

**Solutions:**
1. Use `//godogen:step` instead of specific keywords (matches all)
2. Change the feature to use explicit keywords:
   ```gherkin
   Given I am logged in
   When I click the login button
   ```
3. Define the step with the inherited keyword type

The language server reports this as a diagnostic - look for "step kind mismatch" warnings.

## Ambiguous step matches

When multiple step definitions match the same feature step, you'll get an "ambiguous match" warning.

**Example:**
```go
//godogen:when ^I click "([^"]*)"$
func (s *Suite) iClick(element string) error { ... }

//godogen:when ^I click "([^"]*)"$  // Duplicate!
func (s *Suite) iClickButton(element string) error { ... }
```

**Common causes:**
- Duplicate patterns in different files
- Overly broad patterns that overlap (e.g., `.*` matching too much)
- Copy-paste errors when creating new steps

**Solutions:**
1. Remove duplicate definitions
2. Make patterns more specific:
   ```go
   //godogen:when ^I click the "([^"]*)" button$
   //godogen:when ^I click the "([^"]*)" link$
   ```
3. Use `godogen-language-server list-steps` to find all definitions with similar patterns

## go generate not updating

1. Ensure the file has a `//go:generate` directive:
   ```go
   //go:generate go run github.com/lukasngl/godogen@latest
   ```
2. Run from the directory containing the directive, or use `./...`:
   ```bash
   go generate ./...
   ```
3. Check for Go compilation errors in step files - the generator won't run if code doesn't compile
4. Verify the output file isn't gitignored or read-only
5. The generator creates `*_initializer.go` next to your step file
