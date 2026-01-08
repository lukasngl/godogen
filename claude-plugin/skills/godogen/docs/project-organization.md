# Godogen Project Organization

## Basic Structure

```
myproject/
├── features/           # Gherkin feature files
│   ├── login.feature
│   └── checkout.feature
├── steps/              # Step definitions
│   ├── auth_steps.go
│   └── cart_steps.go
└── testsuite/
    └── main_test.go    # Test runner
```

## Domain-Based Organization (Recommended)

For larger projects, organize steps by domain alongside the code they test:

```
myproject/
├── features/                    # All feature files
│   ├── auth/
│   └── cart/
├── internal/
│   ├── auth/
│   │   ├── auth.go              # Production code
│   │   └── authtest/            # Test steps for auth
│   │       ├── context.go       # Test context/state
│   │       ├── steps.go         # Step definitions
│   │       └── steps_initialize.go  # Generated
│   └── cart/
│       ├── cart.go
│       └── carttest/
│           ├── context.go
│           ├── steps.go
│           └── steps_initialize.go
└── testsuite/
    └── main_test.go             # Combines all step contexts
```

## TestContext Pattern

Each domain has a context struct holding scenario state:

```go
// internal/auth/authtest/context.go
package authtest

type TestContext struct {
    user    *User
    session *Session
    err     error
}

func NewTestContext() *TestContext {
    return &TestContext{}
}
```

Steps are methods on the context:

```go
// internal/auth/authtest/steps.go
//go:generate go run github.com/lukasngl/godogen

package authtest

//godogen:given ^I am logged in as "([^"]*)"$
func (tc *TestContext) IAmLoggedInAs(username string) error {
    tc.user = &User{Name: username}
    tc.session = NewSession(tc.user)
    return nil
}

//godogen:then ^I should be authenticated$
func (tc *TestContext) IShouldBeAuthenticated() error {
    if tc.session == nil {
        return fmt.Errorf("no session")
    }
    return nil
}
```

## Composable Step Registration

Combine multiple domain contexts in the test runner:

```go
// testsuite/main_test.go
package testsuite

import (
    "testing"
    "github.com/cucumber/godog"
    "myproject/internal/auth/authtest"
    "myproject/internal/cart/carttest"
)

func TestFeatures(t *testing.T) {
    suite := godog.TestSuite{
        ScenarioInitializer: func(sc *godog.ScenarioContext) {
            // Each domain registers its own steps
            authtest.InitializeSteps(sc, authtest.NewTestContext())
            carttest.InitializeSteps(sc, carttest.NewTestContext())
        },
        Options: &godog.Options{
            Format:   "pretty",
            Paths:    []string{"../features"},
            TestingT: t,
        },
    }
    suite.Run()
}
```

## Shared Step Libraries

For steps shared across multiple projects, configure `stepPatterns` in `.godogen-language-server.json`:

```json
{
  "stepPatterns": [
    "**/*.feature",
    "**/*test/steps.go",
    "../shared-steps/**/*.go"
  ]
}
```

## Configuration Examples

**Standard project:**

```json
{
  "stepPatterns": [
    "**/*_steps.go",
    "**/*.feature"
  ]
}
```

**Features in subdirectory:**

```json
{
  "stepPatterns": [
    "tests/features/**/*.feature",
    "tests/steps/**/*.go"
  ]
}
```

**Monorepo:**

```json
{
  "stepPatterns": [
    "services/*/features/**/*.feature",
    "services/*/steps/**/*.go"
  ]
}
```

**External features (parent directory):**

```json
{
  "stepPatterns": [
    "../features/**/*.feature",
    "steps/**/*.go"
  ]
}
```
