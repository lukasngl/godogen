Feature: Diagnostics
  As a developer
  I want to see validation errors for Go step definitions
  So I can fix issues before running tests

  Rule: Pattern validation

    Scenario: Pattern missing both anchors
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given I have cukes
        func IHaveCukes() {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message is "The pattern should start with '^' and end with '$'"

    Scenario: Pattern missing start anchor
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given I have cukes$
        func IHaveCukes() {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message is "The pattern should start with '^' and end with '$'"

    Scenario: Pattern missing end anchor
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes
        func IHaveCukes() {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message is "The pattern should start with '^' and end with '$'"

    Scenario: Invalid regex pattern
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+ cukes$
        func IHaveCukes() {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message contains "regex pattern does not compile"

    Scenario: Valid pattern has no diagnostics
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() {}
        """
      When I request diagnostics for steps.go
      Then I get 0 diagnostics

  Rule: Parameter count matching
                                                                                                                            Regex groups must match function parameters

    Scenario: Too few parameters
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes() {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message is "pattern has 1 groups, but function has 0 regular parameters"

    Scenario: Too many parameters
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes(count int) {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message is "pattern has 0 groups, but function has 1 regular parameters"

    Scenario: Matching parameter count
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      When I request diagnostics for steps.go
      Then I get 0 diagnostics

  Rule: Function signature validation
                                            Step functions must have valid parameter and return types

    Scenario: Invalid parameter type
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes(invalid bool) {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message contains "has unexpected type"

    Scenario: context.Context must be first parameter
      When steps.go is added to the workspace:
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
      When steps.go is added to the workspace:
        """
        package steps
        import "github.com/cucumber/godog"

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(table *godog.Table, count int) error { return nil }
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message contains "must be last parameter"

    Scenario: Invalid return type
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() int { return 0 }
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message contains "should return error"

    Scenario: Too many return values
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() (int, string, error) { return 0, "", nil }
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message contains "should return at most two values"

    Scenario: Valid function signature
      When steps.go is added to the workspace:
        """
        package steps
        import "context"

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(ctx context.Context, count int) error { return nil }
        """
      When I request diagnostics for steps.go
      Then I get 0 diagnostics

  Rule: Workspace file preference
                                            Workspace files used for diagnostics

    Scenario: Workspace diagnostics override disk diagnostics
      When steps.go is added to the filesystem:
        """
        package steps

        //godogen:given I have disk cukes
        func IHaveDiskCukes() {}
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have workspace cukes$
        func IHaveWorkspaceCukes() {}
        """
      When I request diagnostics for steps.go
      Then I get 0 diagnostics
