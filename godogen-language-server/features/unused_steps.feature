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
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) melons$
        ^ HINT: Step definition is not used
        func IHaveMelons(count int) {}
        """

    Scenario: All step definitions are used
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Simple test
            Given I have 5 cukes
            And I have 3 melons
        """
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) melons$
        func IHaveMelons(count int) {}
        """

    Scenario: Step kind must match
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Simple test
            When I have 5 cukes
        """
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        ^ HINT: Step definition is not used
        func IHaveCukes(count int) {}
        """

    Scenario: Generic step matches any kind
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Simple test
            When I have 5 cukes
            Then I have 3 melons
        """
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:step ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:step ^I have (\d+) melons$
        func IHaveMelons(count int) {}
        """

    Scenario: Multiple patterns on same function
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Simple test
            Given I have 5 cukes
        """
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        //godogen:given ^I have (\d+) melons$
        ^ HINT: Step definition is not used
        func IHaveFruits(count int) {}
        """

  Rule: External step definitions do not receive unused hints

    Steps defined in files outside the workspace root are considered external
    library code. Their usage cannot be fully determined, so unused hints
    are suppressed for them.

    Scenario: Step definition outside workspace root is not flagged as unused
      Given the workspace root is /project
      And /project/test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Simple test
            Given I have 5 cukes
        """
      Then /shared/lib/steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) melons$
        func IHaveMelons(count int) {}
        """

    Scenario: Step definition inside workspace root is still flagged as unused
      Given the workspace root is /project
      And /project/test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Simple test
            Given I have 5 cukes
        """
      Then /project/steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) melons$
        ^ HINT: Step definition is not used
        func IHaveMelons(count int) {}
        """

  Rule: Invalid patterns are not checked for usage

    Scenario: Invalid regex pattern is not checked
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Simple test
            Given I have 5 cukes
        """
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+ cukes$
        ^ ERROR: regex pattern does not compile
        func IHaveCukes(count int) {}
        """

  Rule: Scenario Outline placeholder steps count as usage

    Scenario: Step used only in Scenario Outline is not marked unused
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario Outline: Shopping
            Given I have <count> cukes

            Examples:
              | count |
              | 5     |
        """
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
