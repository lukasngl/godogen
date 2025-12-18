Feature: Ambiguous Step Matches
  As a developer
  I want to see warnings when a step matches multiple definitions
  So I can fix ambiguous patterns before runtime

  Rule: Steps matching multiple definitions are reported

    Scenario: Step matches two definitions
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) \w+$
        func IHaveSomething(count int) {}
        """
      Then test.feature has the following diagnostics:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
            ^ WARNING: Ambiguous step: matches 2 definitions
        """

    Scenario: Step matches three definitions
      Given steps1.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      And steps2.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) \w+$
        func IHaveSomething(count int) {}
        """
      And steps3.go is added to the workspace:
        """
        package steps

        //godogen:step ^I have .*$
        func IHaveAnything() {}
        """
      Then test.feature has the following diagnostics:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
            ^ WARNING: Ambiguous step: matches 3 definitions
        """

    Scenario: Step matches exactly one definition
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) melons$
        func IHaveMelons(count int) {}
        """
      Then test.feature has the following diagnostics:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
        """

    Scenario: Generic step creates ambiguity
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:when ^I have (\d+) cukes$
        func WhenIHaveCukes(count int) {}

        //godogen:step ^I have (\d+) cukes$
        func GenericCukes(count int) {}
        """
      Then test.feature has the following diagnostics:
        """
        Feature: Test
          Scenario: Shopping
            When I have 5 cukes
            ^ WARNING: Ambiguous step: matches 2 definitions
        """

    Scenario: Different step kinds do not create ambiguity
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func GivenIHaveCukes(count int) {}

        //godogen:when ^I have (\d+) cukes$
        func WhenIHaveCukes(count int) {}
        """
      Then test.feature has the following diagnostics:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
        """

    Scenario: Conjunction steps are checked for ambiguity
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) melons$
        func IHaveMelons(count int) {}

        //godogen:given ^I have (\d+) \w+$
        func IHaveSomething(count int) {}
        """
      Then test.feature has the following diagnostics:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
            ^ WARNING: Ambiguous step: matches 2 definitions
            And I have 5 melons
            ^ WARNING: Ambiguous step: matches 2 definitions
        """

  Rule: Ambiguity across files is detected

    Scenario: Definitions in different files create ambiguity
      Given steps1.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      And steps2.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) \w+$
        func IHaveSomething(count int) {}
        """
      Then test.feature has the following diagnostics:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
            ^ WARNING: Ambiguous step: matches 2 definitions
        """

  Rule: No ambiguity warning when step is undefined

    Scenario: Step with no matches has no ambiguity warning
      Given steps.go is added to the workspace:
        """
        package steps
        """
      Then test.feature has the following diagnostics:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
            ^ ERROR: No step definition found for: Given I have 5 cukes
        """

  Rule: Scenario Outline placeholder steps are checked for ambiguity

    Scenario: Ambiguous placeholder step is reported
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) \w+$
        func IHaveSomething(count int) {}
        """
      Then test.feature has the following diagnostics:
        """
        Feature: Test
          Scenario Outline: Shopping
            Given I have <count> cukes
            ^ WARNING: Ambiguous step: matches 2 definitions

            Examples:
              | count |
              | 5     |
        """
