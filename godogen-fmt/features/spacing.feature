Feature: Spacing
  As a developer
  I want consistent blank lines between elements
  So my feature files have clear visual separation

  Rule: Blank line after Feature line (before description or first child)

    Scenario: Feature with immediate scenario
      Given the input:
        """
        Feature: Test
        Scenario: Test
        Given test
        """
      When I format
      Then the output is:
        """
        Feature: Test

          Scenario: Test
            Given test
        """

  Rule: Blank line between scenarios

    Scenario: Multiple scenarios
      Given the input:
        """
        Feature: Test
        Scenario: First
        Given first
        Scenario: Second
        Given second
        Scenario: Third
        Given third
        """
      When I format
      Then the output is:
        """
        Feature: Test

          Scenario: First
            Given first

          Scenario: Second
            Given second

          Scenario: Third
            Given third
        """

  Rule: Blank line between background and first scenario

    Scenario: Background followed by scenario
      Given the input:
        """
        Feature: Test
        Background:
        Given setup
        Scenario: Test
        Given test
        """
      When I format
      Then the output is:
        """
        Feature: Test

          Background:
            Given setup

          Scenario: Test
            Given test
        """

  Rule: Blank line before Examples

    Scenario: Scenario Outline with Examples
      Given the input:
        """
        Feature: Test
        Scenario Outline: Test
        Given <value>
        Examples:
        | value |
        | one   |
        """
      When I format
      Then the output is:
        """
        Feature: Test

          Scenario Outline: Test
            Given <value>

            Examples:
              | value |
              | one   |
        """

    Scenario: Multiple Examples blocks
      Given the input:
        """
        Feature: Test
        Scenario Outline: Test
        Given <value>
        Examples: First set
        | value |
        | one   |
        Examples: Second set
        | value |
        | two   |
        """
      When I format
      Then the output is:
        """
        Feature: Test

          Scenario Outline: Test
            Given <value>

            Examples: First set
              | value |
              | one   |

            Examples: Second set
              | value |
              | two   |
        """

  Rule: No blank lines between steps

    Scenario: Consecutive steps
      Given the input:
        """
        Feature: Test
        Scenario: Test
        Given first

        When second

        Then third
        """
      When I format
      Then the output is:
        """
        Feature: Test

          Scenario: Test
            Given first
            When second
            Then third
        """

  Rule: No trailing blank lines

    Scenario: Remove trailing blank lines
      Given the input:
        """
        Feature: Test
        Scenario: Test
        Given test


        """
      When I format
      Then the output is:
        """
        Feature: Test

          Scenario: Test
            Given test
        """

  Rule: Single trailing newline

    Scenario: File ends with newline
      Given the input:
        """
        Feature: Test
        Scenario: Test
        Given test
        """
      When I format
      Then the output ends with a single newline
