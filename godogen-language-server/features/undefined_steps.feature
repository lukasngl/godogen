Feature: Undefined Steps
  As a developer
  I want to see errors for steps without definitions
  So I can implement missing step definitions before running tests

  Rule: Steps without matching definitions are reported

    Scenario: Feature step with no matching definition
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      Then test.feature has the following diagnostics:
        """gherkin
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
            When I eat 2 cukes
            ^ ERROR: No step definition found
            Then I have 3 cukes
            ^ ERROR: No step definition found
        """

    Scenario: All steps have matching definitions
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:when ^I eat (\d+) cukes$
        func IEatCukes(count int) {}
        """
      Then test.feature has the following diagnostics:
        """gherkin
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
            When I eat 2 cukes
        """

    Scenario: Generic step definition matches any kind
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:step ^I have (\d+) \w+$
        func IHaveSomething(count int) {}
        """
      Then test.feature has the following diagnostics:
        """gherkin
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
            When I have 3 melons
            Then I have 2 apples
        """

    Scenario: Step kind must match
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      Then test.feature has the following diagnostics:
        """gherkin
        Feature: Test
          Scenario: Shopping
            When I have 5 cukes
            ^ ERROR: No step definition found
        """

    Scenario: Conjunction steps inherit kind
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) melons$
        func IHaveMelons(count int) {}
        """
      Then test.feature has the following diagnostics:
        """gherkin
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
            And I have 3 melons
            But I have no apples
            ^ ERROR: No step definition found
        """

    Scenario: Steps in background are checked
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      Then test.feature has the following diagnostics:
        """gherkin
        Feature: Test
          Background:
            Given I have 5 cukes

          Scenario: Eat some
            When I eat 2 cukes
            ^ ERROR: No step definition found
        """

    Scenario: Steps in rules are checked
      Given steps.go is added to the workspace:
        """
        package steps
        """
      Then test.feature has the following diagnostics:
        """gherkin
        Feature: Test
          Rule: Shopping rules
            Scenario: Buy cukes
              Given I have 5 cukes
              ^ ERROR: No step definition found
        """

  Rule: Scenario Outline placeholder steps are checked

    Scenario: Placeholder step matches regex with capture group
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      Then test.feature has the following diagnostics:
        """gherkin
        Feature: Shopping
          Scenario Outline: Buy fruits
            Given I have <count> cukes

            Examples:
              | count |
              | 5     |
              | 10    |
        """

    Scenario: Undefined placeholder step is reported
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      Then test.feature has the following diagnostics:
        """gherkin
        Feature: Test
          Scenario Outline: Shopping
            Given I have <count> cukes
            When I eat them
            ^ ERROR: No step definition found

            Examples:
              | count |
              | 5     |
        """

    Scenario: Undefined step includes related info for each example row
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      Then test.feature has the following diagnostics:
        """gherkin
        Feature: Shopping
          Scenario Outline: Buy fruits
            Given I have <count> cukes
            When I eat them
            ^ ERROR: No step definition found
            ^ ERROR: No step definition found
            ^ ERROR: No step definition found

            Examples:
              | count |
              | 5     |
              | 10    |
              | 15    |
        """

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
      Then test.feature has the following diagnostics:
        """gherkin
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
            ^ ERROR: No step definition found
            Given I have 3 melons
        """
