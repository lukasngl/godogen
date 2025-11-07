Feature: Step Signature Diagnostics
  As a developer
  I want to see validation errors for step function signatures
  So I can fix signature issues before running tests

  Rule: Function parameter validation

    Scenario: Invalid parameter type
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes(invalid bool) {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message contains "has unexpected type"

    Scenario: context.Context must be first parameter
      Given steps.go is added to the workspace:
        """
        package steps
        import "context"

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int, ctx context.Context) {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message contains "must be first parameter"

    Scenario: godog.Table must be last parameter
      Given steps.go is added to the workspace:
        """
        package steps
        import "github.com/cucumber/godog"

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(table *godog.Table, count int) error { return nil }
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message contains "must be last parameter"

    Scenario: Valid parameter types
      Given steps.go is added to the workspace:
        """
        package steps
        import "context"
        import "github.com/cucumber/godog"

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(ctx context.Context, count int, table *godog.Table) error {
          return nil
        }
        """
      When I request diagnostics for steps.go
      Then I get 0 diagnostics

  Rule: Function return type validation

    Scenario: Invalid return type
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() int { return 0 }
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message contains "should return error"

    Scenario: Too many return values
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() (int, string, error) { return 0, "", nil }
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message contains "should return at most two values"

    Scenario: Valid single return value - error
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() error { return nil }
        """
      When I request diagnostics for steps.go
      Then I get 0 diagnostics

    Scenario: Valid single return value - godog.Steps
      Given steps.go is added to the workspace:
        """
        package steps
        import "github.com/cucumber/godog"

        //godogen:given ^I have cukes$
        func IHaveCukes() godog.Steps { return nil }
        """
      When I request diagnostics for steps.go
      Then I get 0 diagnostics

    Scenario: Valid single return value - context.Context
      Given steps.go is added to the workspace:
        """
        package steps
        import "context"

        //godogen:given ^I have cukes$
        func IHaveCukes() context.Context { return nil }
        """
      When I request diagnostics for steps.go
      Then I get 0 diagnostics

    Scenario: Valid two return values
      Given steps.go is added to the workspace:
        """
        package steps
        import "context"

        //godogen:given ^I have cukes$
        func IHaveCukes() (context.Context, error) { return nil, nil }
        """
      When I request diagnostics for steps.go
      Then I get 0 diagnostics

    Scenario: Valid no return values
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() {}
        """
      When I request diagnostics for steps.go
      Then I get 0 diagnostics

  Rule: Combined validation

    Scenario: Valid function signature
      Given steps.go is added to the workspace:
        """
        package steps
        import "context"

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(ctx context.Context, count int) error { return nil }
        """
      When I request diagnostics for steps.go
      Then I get 0 diagnostics

    Scenario: Multiple errors are reported
      Given steps.go is added to the workspace:
        """
        package steps
        import "context"

        //godogen:given ^I have cukes$
        func IHaveCukes(count int, ctx context.Context, invalid bool) int { return 0 }
        """
      When I request diagnostics for steps.go
      Then I get 3 diagnostics
