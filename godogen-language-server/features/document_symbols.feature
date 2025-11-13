Feature: Document Symbols
  As a developer
  I want to see an outline of step definitions and feature elements
  So I can navigate large files efficiently

  Rule: Go files show step definitions as symbols

    Scenario: Step definitions appear in symbol list
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:when ^I eat (\d+) cukes$
        func IEatCukes(count int) {}

        //godogen:then ^I have (\d+) cukes left$
        func IHaveCukesLeft(count int) {}
        """
      When I request document symbols for steps.go
      Then I get 3 symbols
      And symbol 0 name is "Given: ^I have (\\d+) cukes$"
      And symbol 0 kind is "Function"
      And symbol 0 is on line 3
      And symbol 1 name is "When: ^I eat (\\d+) cukes$"
      And symbol 1 kind is "Function"
      And symbol 1 is on line 6
      And symbol 2 name is "Then: ^I have (\\d+) cukes left$"
      And symbol 2 kind is "Function"
      And symbol 2 is on line 9

    Scenario: Generic steps show as Step kind
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:step ^I .*$
        func IDoAnything() {}
        """
      When I request document symbols for steps.go
      Then I get 1 symbol
      And symbol 0 name is "Step: ^I .*$"

    Scenario: Multiple patterns on same function show multiple symbols
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        //godogen:given ^I have (\d+) melons$
        func IHaveFruits(count int) {}
        """
      When I request document symbols for steps.go
      Then I get 2 symbols
      And symbol 0 name is "Given: ^I have (\\d+) cukes$"
      And symbol 0 is on line 3
      And symbol 1 name is "Given: ^I have (\\d+) melons$"
      And symbol 1 is on line 4

    Scenario: Hooks appear in symbol list
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:before
        func SetupTest() {}

        //godogen:after
        func TeardownTest() {}

        //godogen:before_step
        func BeforeEachStep() {}

        //godogen:after_step
        func AfterEachStep() {}
        """
      When I request document symbols for steps.go
      Then I get 4 symbols
      And symbol 0 name is "Before Hook"
      And symbol 0 kind is "Function"
      And symbol 1 name is "After Hook"
      And symbol 1 kind is "Function"
      And symbol 2 name is "Before Step Hook"
      And symbol 2 kind is "Function"
      And symbol 3 name is "After Step Hook"
      And symbol 3 kind is "Function"

    Scenario: Functions without directives are not symbols
      Given steps.go is added to the workspace:
        """
        package steps

        func helperFunction() {}

        //godogen:given ^I have cukes$
        func IHaveCukes() {}

        func anotherHelper() {}
        """
      When I request document symbols for steps.go
      Then I get 1 symbol
      And symbol 0 name is "Given: ^I have cukes$"

  Rule: Feature files show scenarios and steps as symbols

    Scenario: Scenarios appear as symbols
      Given test.feature is added to the workspace:
        """
        Feature: Shopping
          Scenario: Buy cukes
            Given I have money
            When I buy cukes

          Scenario: Eat cukes
            Given I have cukes
            When I eat them
        """
      When I request document symbols for test.feature
      Then I get 2 symbols
      And symbol 0 name is "Scenario: Buy cukes"
      And symbol 0 kind is "Method"
      And symbol 0 is on line 2
      And symbol 1 name is "Scenario: Eat cukes"
      And symbol 1 kind is "Method"
      And symbol 1 is on line 6

    Scenario: Scenarios contain steps as children
      Given test.feature is added to the workspace:
        """
        Feature: Shopping
          Scenario: Buy cukes
            Given I have money
            When I buy cukes
            Then I have cukes
        """
      When I request document symbols for test.feature
      Then I get 1 symbol
      And symbol 0 has 3 children
      And symbol 0 child 0 name is "Given I have money"
      And symbol 0 child 0 kind is "Property"
      And symbol 0 child 1 name is "When I buy cukes"
      And symbol 0 child 2 name is "Then I have cukes"

    Scenario: Background appears as symbol with steps
      Given test.feature is added to the workspace:
        """
        Feature: Shopping
          Background:
            Given I have a wallet

          Scenario: Buy cukes
            When I buy cukes
        """
      When I request document symbols for test.feature
      Then I get 2 symbols
      And symbol 0 name is "Background"
      And symbol 0 kind is "Method"
      And symbol 0 has 1 child
      And symbol 0 child 0 name is "Given I have a wallet"
      And symbol 1 name is "Scenario: Buy cukes"

    Scenario: Rules group scenarios
      Given test.feature is added to the workspace:
        """
        Feature: Shopping
          Rule: Payment rules
            Scenario: Pay with cash
              Given I have cash

            Scenario: Pay with card
              Given I have a card
        """
      When I request document symbols for test.feature
      Then I get 1 symbol
      And symbol 0 name is "Rule: Payment rules"
      And symbol 0 kind is "Class"
      And symbol 0 has 2 children
      And symbol 0 child 0 name is "Scenario: Pay with cash"
      And symbol 0 child 1 name is "Scenario: Pay with card"

    Scenario: Feature appears as root container
      Given test.feature is added to the workspace:
        """
        Feature: Shopping
          Scenario: Buy cukes
            Given I have money
        """
      When I request document symbols for test.feature with hierarchy
      Then I get 1 symbol
      And symbol 0 name is "Feature: Shopping"
      And symbol 0 kind is "Module"
      And symbol 0 has 1 child
      And symbol 0 child 0 name is "Scenario: Buy cukes"

    Scenario: Scenario outlines appear as symbols
      Given test.feature is added to the workspace:
        """
        Feature: Shopping
          Scenario Outline: Buy fruits
            Given I have <count> <fruit>

            Examples:
              | count | fruit  |
              | 5     | cukes  |
              | 3     | melons |
        """
      When I request document symbols for test.feature
      Then I get 1 symbol
      And symbol 0 name is "Scenario Outline: Buy fruits"
      And symbol 0 kind is "Method"
