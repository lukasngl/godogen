Feature: Undefined Steps
  As a developer
  I want to see errors for steps without definitions
  So I can implement missing step definitions before running tests

  Rule: Steps without matching definitions are reported

    Scenario: Feature step with no matching definition
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
            When I eat 2 cukes
            Then I have 3 cukes
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      When I request diagnostics for test.feature
      Then I get 2 diagnostics
      And diagnostic 0 message is "No step definition found for: When I eat 2 cukes"
      And diagnostic 0 severity is "Error"
      And diagnostic 0 is on line 4
      And diagnostic 1 message is "No step definition found for: Then I have 3 cukes"
      And diagnostic 1 is on line 5

    Scenario: All steps have matching definitions
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
            When I eat 2 cukes
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:when ^I eat (\d+) cukes$
        func IEatCukes(count int) {}
        """
      When I request diagnostics for test.feature
      Then I get 0 diagnostics

    Scenario: Generic step definition matches any kind
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
            When I have 3 melons
            Then I have 2 apples
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:step ^I have (\d+) \w+$
        func IHaveSomething(count int) {}
        """
      When I request diagnostics for test.feature
      Then I get 0 diagnostics

    Scenario: Step kind must match
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Shopping
            When I have 5 cukes
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      When I request diagnostics for test.feature
      Then I get 1 diagnostic
      And diagnostic 0 message is "No step definition found for: When I have 5 cukes"

    Scenario: Conjunction steps inherit kind
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
            And I have 3 melons
            But I have no apples
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) melons$
        func IHaveMelons(count int) {}
        """
      When I request diagnostics for test.feature
      Then I get 1 diagnostic
      And diagnostic 0 message is "No step definition found for: But I have no apples"
      And diagnostic 0 is on line 5

    Scenario: Steps in background are checked
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Background:
            Given I have 5 cukes

          Scenario: Eat some
            When I eat 2 cukes
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      When I request diagnostics for test.feature
      Then I get 1 diagnostic
      And diagnostic 0 message is "No step definition found for: When I eat 2 cukes"

    Scenario: Steps in rules are checked
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Rule: Shopping rules
            Scenario: Buy cukes
              Given I have 5 cukes
        """
      And steps.go is added to the workspace:
        """
        package steps
        """
      When I request diagnostics for test.feature
      Then I get 1 diagnostic
      And diagnostic 0 message is "No step definition found for: Given I have 5 cukes"

  Rule: Workspace file preference

    Scenario: Workspace step definitions override disk
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
      And test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
            Given I have 3 melons
        """
      When I request diagnostics for test.feature
      Then I get 1 diagnostic
      And diagnostic 0 message is "No step definition found for: Given I have 5 cukes"
