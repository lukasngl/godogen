Feature: Step Pattern Diagnostics
  As a developer
  I want to see validation errors for step patterns
  So I can fix pattern issues before running tests

  Rule: Pattern validation

    Scenario: Pattern missing both anchors
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given I have cukes
        ^ ERROR: The pattern should start with '^' and end with '$'
        func IHaveCukes() {}
        """

    Scenario: Pattern missing start anchor
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given I have cukes$
        ^ ERROR: The pattern should start with '^' and end with '$'
        func IHaveCukes() {}
        """

    Scenario: Pattern missing end anchor
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have cukes
        ^ ERROR: The pattern should start with '^' and end with '$'
        func IHaveCukes() {}
        """

    Scenario: Invalid regex pattern
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+ cukes$
        ^ ERROR: regex pattern does not compile
        func IHaveCukes() {}
        """

    Scenario: Valid pattern has no diagnostics
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() {}
        """

  Rule: Parameter count matching

    Scenario: Too few parameters
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        ^ ERROR: pattern has 1 groups, but function has 0 regular parameters
        func IHaveCukes() {}
        """

    Scenario: Too many parameters
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have cukes$
        ^ ERROR: pattern has 0 groups, but function has 1 regular parameters
        func IHaveCukes(count int) {}
        """

    Scenario: Matching parameter count
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """

  Rule: Workspace file preference

    Scenario: Workspace diagnostics override disk diagnostics
      Given steps.go is added to the filesystem:
        """
        package steps

        //godogen:given I have disk cukes
        func IHaveDiskCukes() {}
        """
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have workspace cukes$
        func IHaveWorkspaceCukes() {}
        """
