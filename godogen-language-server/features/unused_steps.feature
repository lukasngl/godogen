Feature: Unused Step Definitions
  As a developer
  I want to see hints for unused step definitions
  So I can identify and remove dead code

  Rule: Unused step definitions are reported

    Scenario: Step definition with no matching feature steps
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Simple test
            Given I have 5 cukes
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) melons$
        func IHaveMelons(count int) {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message is "Step definition is not used in any feature file"
      And diagnostic 0 severity is "Hint"
      And diagnostic 0 is on line 6

    Scenario: All step definitions are used
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Simple test
            Given I have 5 cukes
            And I have 3 melons
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) melons$
        func IHaveMelons(count int) {}
        """
      When I request diagnostics for steps.go
      Then I get 0 diagnostics

    Scenario: Step kind must match
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Simple test
            When I have 5 cukes
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message is "Step definition is not used in any feature file"

    Scenario: Generic step matches any kind
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Simple test
            When I have 5 cukes
            Then I have 3 melons
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:step ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:step ^I have (\d+) melons$
        func IHaveMelons(count int) {}
        """
      When I request diagnostics for steps.go
      Then I get 0 diagnostics

    Scenario: Multiple patterns on same function
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Simple test
            Given I have 5 cukes
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        //godogen:given ^I have (\d+) melons$
        func IHaveFruits(count int) {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message is "Step definition is not used in any feature file"
      And diagnostic 0 is on line 4

  Rule: Invalid patterns are not checked for usage

    Scenario: Invalid regex pattern is not checked
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Simple test
            Given I have 5 cukes
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+ cukes$
        func IHaveCukes(count int) {}
        """
      When I request diagnostics for steps.go
      Then I get 1 diagnostic
      And diagnostic 0 message contains "regex pattern does not compile"
      And diagnostic 0 does not contain "not used"
