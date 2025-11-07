Feature: Ambiguous Step Matches
  As a developer
  I want to see warnings when a step matches multiple definitions
  So I can fix ambiguous patterns before runtime

  Rule: Steps matching multiple definitions are reported

    Scenario: Step matches two definitions
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) \w+$
        func IHaveSomething(count int) {}
        """
      When I request diagnostics for test.feature
      Then I get 1 diagnostic
      And diagnostic 0 message is "Ambiguous step: matches 2 definitions (steps.go:3, steps.go:6)"
      And diagnostic 0 severity is "Warning"
      And diagnostic 0 is on line 3

    Scenario: Step matches three definitions
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
        """
      And steps1.go is added to the workspace:
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
      When I request diagnostics for test.feature
      Then I get 1 diagnostic
      And diagnostic 0 message is "Ambiguous step: matches 3 definitions (steps1.go:3, steps2.go:3, steps3.go:3)"

    Scenario: Step matches exactly one definition
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Shopping
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
      When I request diagnostics for test.feature
      Then I get 0 diagnostics

    Scenario: Generic step creates ambiguity
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Shopping
            When I have 5 cukes
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:when ^I have (\d+) cukes$
        func WhenIHaveCukes(count int) {}

        //godogen:step ^I have (\d+) cukes$
        func GenericCukes(count int) {}
        """
      When I request diagnostics for test.feature
      Then I get 1 diagnostic
      And diagnostic 0 message is "Ambiguous step: matches 2 definitions"

    Scenario: Different step kinds do not create ambiguity
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func GivenIHaveCukes(count int) {}

        //godogen:when ^I have (\d+) cukes$
        func WhenIHaveCukes(count int) {}
        """
      When I request diagnostics for test.feature
      Then I get 0 diagnostics

    Scenario: Conjunction steps are checked for ambiguity
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
            And I have 3 melons
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) \w+$
        func IHaveSomething(count int) {}
        """
      When I request diagnostics for test.feature
      Then I get 2 diagnostics
      And diagnostic 0 message is "Ambiguous step: matches 2 definitions"
      And diagnostic 0 is on line 3
      And diagnostic 1 message is "Ambiguous step: matches 2 definitions"
      And diagnostic 1 is on line 4

  Rule: Ambiguity across files is detected

    Scenario: Definitions in different files create ambiguity
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
        """
      And steps1.go is added to the workspace:
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
      When I request diagnostics for test.feature
      Then I get 1 diagnostic
      And diagnostic 0 message is "Ambiguous step: matches 2 definitions (steps1.go:3, steps2.go:3)"

  Rule: No ambiguity warning when step is undefined

    Scenario: Step with no matches has no ambiguity warning
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
        """
      And steps.go is added to the workspace:
        """
        package steps
        """
      When I request diagnostics for test.feature
      Then I get 1 diagnostic
      And diagnostic 0 message is "No step definition found for: Given I have 5 cukes"
      And diagnostic 0 message does not contain "Ambiguous"
