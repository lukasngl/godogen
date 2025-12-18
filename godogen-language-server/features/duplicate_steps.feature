Feature: Duplicate Step Definitions
  As a developer
  I want to see errors for duplicate step definitions
  So I can avoid ambiguous behavior at runtime

  Rule: Duplicate patterns in same file are reported

    Scenario: Same pattern and kind in one file
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukesAgain(count int) {}
        """
      When I request diagnostics for steps.go
      Then I get 2 diagnostics
      And diagnostic 0 message is "Duplicate step definition: pattern already defined at steps.go:3"
      And diagnostic 0 severity is "Error"
      And diagnostic 0 is on line 3
      And diagnostic 1 message is "Duplicate step definition: pattern already defined at steps.go:3"
      And diagnostic 1 is on line 6

    Scenario: Same pattern but different kinds
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:when ^I have (\d+) cukes$
        func WhenIHaveCukes(count int) {}
        """
      When I request diagnostics for steps.go
      Then I get 0 diagnostics

    Scenario: Generic step duplicates specific kind
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:step ^I have (\d+) cukes$
        func GenericCukes(count int) {}
        """
      When I request diagnostics for steps.go
      Then I get 2 diagnostics
      And diagnostic 0 message contains "Duplicate step definition"
      And diagnostic 1 message contains "Duplicate step definition"

    Scenario: Different patterns are not duplicates
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) melons$
        func IHaveMelons(count int) {}
        """
      When I request diagnostics for steps.go
      Then I get 0 diagnostics

  Rule: Duplicate patterns across files are reported

    Scenario: Same pattern in different files
      Given steps1.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      And steps2.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukesAgain(count int) {}
        """
      When I request diagnostics for steps1.go
      Then I get 1 diagnostic
      And diagnostic 0 message is "Duplicate step definition: pattern already defined at steps2.go:3"
      When I request diagnostics for steps2.go
      Then I get 1 diagnostic
      And diagnostic 0 message is "Duplicate step definition: pattern already defined at steps1.go:3"

    Scenario: Workspace file overrides disk file for duplication check
      Given steps.go is added to the filesystem:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) melons$
        func IHaveMelons(count int) {}
        """
      And other.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) melons$
        func IHaveMelonsAgain(count int) {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message contains "Duplicate step definition"
      And diagnostic 0 message contains "other.go"
      And diagnostic 0 message does not contain "cukes"

  Rule: Invalid patterns are not checked for duplication

    Scenario: Invalid regex is not checked for duplicates
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+ cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+ cukes$
        func IHaveCukesAgain(count int) {}
        """
      When I request diagnostics for steps.go
      Then I get 2 diagnostics
      And diagnostic 0 message contains "regex pattern does not compile"
      And diagnostic 1 message contains "regex pattern does not compile"
      And diagnostic 0 message does not contain "Duplicate"
      And diagnostic 1 message does not contain "Duplicate"

  Rule: Diagnostics update when duplicates are added or removed

    Scenario: Adding a duplicate step creates diagnostic
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      When I request diagnostics for steps.go
      Then I get 0 diagnostics
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukesAgain(count int) {}
        """
      And I request diagnostics for steps.go
      Then I get 2 diagnostics
      And diagnostic 0 message contains "Duplicate step definition"
      And diagnostic 1 message contains "Duplicate step definition"

    Scenario: Removing a duplicate step removes diagnostic
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukesAgain(count int) {}
        """
      When I request diagnostics for steps.go
      Then I get 2 diagnostics
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      And I request diagnostics for steps.go
      Then I get 0 diagnostics

    Scenario: Removing duplicate from one file removes diagnostic from both files
      Given steps1.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      And steps2.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukesAgain(count int) {}
        """
      When I request diagnostics for steps1.go
      Then I get 1 diagnostic
      When I request diagnostics for steps2.go
      Then I get 1 diagnostic
      Given steps2.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) melons$
        func IHaveMelons(count int) {}
        """
      And I request diagnostics for steps1.go
      Then I get 0 diagnostics
      When I request diagnostics for steps2.go
      Then I get 0 diagnostics
