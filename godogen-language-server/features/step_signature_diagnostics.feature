Feature: Step Signature Diagnostics
  As a developer
  I want to see validation errors for step function signatures
  So I can fix signature issues before running tests

  Rule: Function parameter validation

    Scenario: Invalid parameter type
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes(invalid bool) {}
                        ^ ERROR: has unexpected type
        """

    Scenario: context.Context must be first parameter
      Then steps.go has the following diagnostics:
        """go
        package steps
        import "context"

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int, ctx context.Context) {}
                                   ^ ERROR: must be first parameter
        """

    Scenario: godog.Table must be last parameter
      Then steps.go has the following diagnostics:
        """go
        package steps
        import "github.com/cucumber/godog"

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(table *godog.Table, count int) error { return nil }
                        ^ ERROR: must be last parameter
        """

    Scenario: Valid parameter types
      Then steps.go has the following diagnostics:
        """go
        package steps
        import "context"
        import "github.com/cucumber/godog"

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(ctx context.Context, count int, table *godog.Table) error {
          return nil
        }
        """

  Rule: Function return type validation

    Scenario: Invalid return type
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() int { return 0 }
                          ^ ERROR: should return error
        """

    Scenario: Too many return values
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() (int, string, error) { return 0, "", nil }
                          ^ ERROR: should return at most two values
        """

    Scenario: Valid single return value - error
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() error { return nil }
        """

    Scenario: Valid single return value - godog.Steps
      Then steps.go has the following diagnostics:
        """go
        package steps
        import "github.com/cucumber/godog"

        //godogen:given ^I have cukes$
        func IHaveCukes() godog.Steps { return nil }
        """

    Scenario: Valid single return value - context.Context
      Then steps.go has the following diagnostics:
        """go
        package steps
        import "context"

        //godogen:given ^I have cukes$
        func IHaveCukes() context.Context { return nil }
        """

    Scenario: Valid two return values
      Then steps.go has the following diagnostics:
        """go
        package steps
        import "context"

        //godogen:given ^I have cukes$
        func IHaveCukes() (context.Context, error) { return nil, nil }
        """

    Scenario: Valid no return values
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() {}
        """

  Rule: Combined validation

    Scenario: Valid function signature
      Then steps.go has the following diagnostics:
        """go
        package steps
        import "context"

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(ctx context.Context, count int) error { return nil }
        """

    Scenario: Multiple errors are reported
      Then steps.go has the following diagnostics:
        """go
        package steps
        import "context"

        //godogen:given ^I have cukes$
        ^ ERROR: pattern has 0 groups, but function has 1 regular parameters
        func IHaveCukes(count int, ctx context.Context, invalid bool) int { return 0 }
        ^ ERROR: must be first parameter
        ^ ERROR: has unexpected type
        ^ ERROR: should return error
        """
