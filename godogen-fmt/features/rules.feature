Feature: Rules Support
  As a developer
  I want Rules formatted correctly
  So I can organize my scenarios logically

  Rule: Rules are indented correctly

    Scenario: Single rule with scenarios
      Given the input:
        ```
        Feature: Test
        Rule: Business rule
        Scenario: First
        Given something
        Scenario: Second
        Given something else
        ```
      When I format
      Then the output is:
        ```
        Feature: Test

          Rule: Business rule

            Scenario: First
              Given something

            Scenario: Second
              Given something else
        ```

    Scenario: Multiple rules
      Given the input:
        ```
        Feature: Test
        Rule: First rule
        Scenario: A
        Given a
        Rule: Second rule
        Scenario: B
        Given b
        ```
      When I format
      Then the output is:
        ```
        Feature: Test

          Rule: First rule

            Scenario: A
              Given a

          Rule: Second rule

            Scenario: B
              Given b
        ```

    Scenario: Rule with background
      Given the input:
        ```
        Feature: Test
        Rule: With background
        Background:
        Given setup
        Scenario: Test
        Given test
        ```
      When I format
      Then the output is:
        ```
        Feature: Test

          Rule: With background

            Background:
              Given setup

            Scenario: Test
              Given test
        ```

    Scenario: Scenarios before and after rules
      Given the input:
        ```
        Feature: Test
        Scenario: Before rules
        Given before
        Rule: A rule
        Scenario: Inside rule
        Given inside
        ```
      When I format
      Then the output is:
        ```
        Feature: Test

          Scenario: Before rules
            Given before

          Rule: A rule

            Scenario: Inside rule
              Given inside
        ```

  Rule: Rule descriptions are preserved

    Scenario: Rule with description
      Given the input:
        ```
        Feature: Test
        Rule: Named rule
        This rule covers important behavior.
        It spans multiple lines.
        Scenario: Test
        Given test
        ```
      When I format
      Then the output is:
        ```
        Feature: Test

          Rule: Named rule
            This rule covers important behavior.
            It spans multiple lines.

            Scenario: Test
              Given test
        ```
