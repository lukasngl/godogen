Feature: Hover Information
  As a developer
  I want to see step definition details when hovering over feature steps
  So I can understand what implementation will be called

  Rule: Hovering over a feature step shows the matching definition

    Scenario: Hover shows single matching definition
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
        """
      And steps.go is added to the workspace:
        """
        package steps

        // IHaveCukes sets the number of cukes in the inventory.
        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) error {
          return nil
        }
        """
      When I hover over line 3 column 11 in test.feature
      Then I get hover content:
        """
        **Step Definition**

        IHaveCukes sets the number of cukes in the inventory.

        ```go
        func IHaveCukes(count int) error
        ```

        **File:** steps.go:5
        **Pattern:** `^I have (\d+) cukes$`
        """

    Scenario: Hover shows definition with context parameter
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Shopping
            When I eat 2 cukes
        """
      And steps.go is added to the workspace:
        """
        package steps
        import "context"

        // IEatCukes consumes the specified number of cukes from inventory.
        //godogen:when ^I eat (\d+) cukes$
        func IEatCukes(ctx context.Context, count int) error {
          return nil
        }
        """
      When I hover over line 3 column 10 in test.feature
      Then I get hover content:
        """
        **Step Definition**

        IEatCukes consumes the specified number of cukes from inventory.

        ```go
        func IEatCukes(ctx context.Context, count int) error
        ```

        **File:** steps.go:6
        **Pattern:** `^I eat (\d+) cukes$`
        """

    Scenario: Hover shows all ambiguous matches
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
      When I hover over line 3 column 11 in test.feature
      Then I get hover content:
        """
        **Step Definitions (2 matches)**

        ```go
        func IHaveCukes(count int)
        ```
        **File:** steps.go:4
        **Pattern:** `^I have (\d+) cukes$`

        ---

        ```go
        func IHaveSomething(count int)
        ```
        **File:** steps.go:7
        **Pattern:** `^I have (\d+) \w+$`
        """

    Scenario: Hover on undefined step shows error
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
      When I hover over line 3 column 11 in test.feature
      Then I get hover content:
        """
        **No step definition found**

        No matching step definition for:
        ```gherkin
        Given I have 5 cukes
        ```
        """

    Scenario: Hover outside step text returns nothing
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
        """
      When I hover over line 2 column 5 in test.feature
      Then I get no hover content

    Scenario: Hover shows definition without godoc
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
        func IHaveCukes(count int) error {
          return nil
        }
        """
      When I hover over line 3 column 11 in test.feature
      Then I get hover content:
        """
        **Step Definition**

        ```go
        func IHaveCukes(count int) error
        ```

        **File:** steps.go:4
        **Pattern:** `^I have (\d+) cukes$`
        """

    Scenario: Hover shows multiline godoc
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
        """
      And steps.go is added to the workspace:
        """
        package steps

        // IHaveCukes sets the number of cukes in the inventory.
        // This step initializes the state for the scenario and can be
        // used in Given or Background sections.
        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) error {
          return nil
        }
        """
      When I hover over line 3 column 11 in test.feature
      Then I get hover content:
        """
        **Step Definition**

        IHaveCukes sets the number of cukes in the inventory.
        This step initializes the state for the scenario and can be
        used in Given or Background sections.

        ```go
        func IHaveCukes(count int) error
        ```

        **File:** steps.go:7
        **Pattern:** `^I have (\d+) cukes$`
        """

    Scenario: Hover on conjunction step uses inherited kind
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

        //godogen:given ^I have (\d+) melons$
        func IHaveMelons(count int) {}
        """
      When I hover over line 4 column 9 in test.feature
      Then I get hover content:
        """
        **Step Definition**

        ```go
        func IHaveMelons(count int)
        ```

        **File:** steps.go:7
        **Pattern:** `^I have (\d+) melons$`
        """

  Rule: Hovering over Go step definition shows usage information

    Scenario: Hover shows step usage count
      Given test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Shopping
            Given I have 5 cukes
            And I have 3 cukes
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      When I hover over line 3 column 15 in steps.go
      Then I get hover content:
        """
        **Step Definition**

        **Pattern:** `^I have (\d+) cukes$`
        **Kind:** Given
        **Used in:** 2 places

        - test.feature:3
        - test.feature:4
        """

    Scenario: Hover shows unused status
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
      When I hover over line 6 column 15 in steps.go
      Then I get hover content:
        """
        **Step Definition**

        **Pattern:** `^I have (\d+) melons$`
        **Kind:** Given
        **Used in:** 0 places (unused)
        """

    Scenario: Hover on pattern comment shows same information
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
        """
      When I hover over line 3 column 25 in steps.go
      Then I get hover content:
        """
        **Step Definition**

        **Pattern:** `^I have (\d+) cukes$`
        **Kind:** Given
        **Used in:** 1 place

        - test.feature:3
        """
