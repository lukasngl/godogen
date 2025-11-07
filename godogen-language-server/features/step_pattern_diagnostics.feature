Feature: Step Pattern Diagnostics
  As a developer
  I want to see validation errors for step patterns
  So I can fix pattern issues before running tests

  Rule: Pattern validation

    Scenario: Pattern missing both anchors
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given I have cukes
        func IHaveCukes() {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message is "The pattern should start with '^' and end with '$'"

    Scenario: Pattern missing start anchor
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given I have cukes$
        func IHaveCukes() {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message is "The pattern should start with '^' and end with '$'"

    Scenario: Pattern missing end anchor
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes
        func IHaveCukes() {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message is "The pattern should start with '^' and end with '$'"

    Scenario: Invalid regex pattern
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+ cukes$
        func IHaveCukes() {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message contains "regex pattern does not compile"

    Scenario: Valid pattern has no diagnostics
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() {}
        """
      When I request diagnostics for steps.go
      Then I get 0 diagnostics

  Rule: Parameter count matching

    Scenario: Too few parameters
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes() {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message is "pattern has 1 groups, but function has 0 regular parameters"

    Scenario: Too many parameters
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes(count int) {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message is "pattern has 0 groups, but function has 1 regular parameters"

    Scenario: Matching parameter count
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      When I request diagnostics for steps.go
      Then I get 0 diagnostics

  Rule: Workspace file preference

    Scenario: Workspace diagnostics override disk diagnostics
      Given steps.go is added to the filesystem:
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
