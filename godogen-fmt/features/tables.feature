Feature: Table Formatting
  As a developer
  I want tables aligned properly
  So they are easy to read

  Rule: Table columns are aligned

    Scenario: Align columns by padding
      Given the input:
        """
        Feature: Test
          Scenario: Test
            Given users:
              | name | age | city |
              | Alice | 30 | New York |
              | Bob | 25 | LA |
        """
      When I format
      Then the output is:
        """
        Feature: Test

          Scenario: Test
            Given users:
              | name  | age | city     |
              | Alice | 30  | New York |
              | Bob   | 25  | LA       |
        """

    Scenario: Single column table
      Given the input:
        """
        Feature: Test
          Scenario: Test
            Given values:
              | value |
              | one |
              | two |
              | three |
        """
      When I format
      Then the output is:
        """
        Feature: Test

          Scenario: Test
            Given values:
              | value |
              | one   |
              | two   |
              | three |
        """

    Scenario: Empty cells
      Given the input:
        """
        Feature: Test
          Scenario: Test
            Given data:
              | a | b | c |
              | 1 |  | 3 |
              |  | 2 |  |
        """
      When I format
      Then the output is:
        """
        Feature: Test

          Scenario: Test
            Given data:
              | a | b | c |
              | 1 |   | 3 |
              |   | 2 |   |
        """

    Scenario: Examples table alignment
      Given the input:
        """
        Feature: Test
          Scenario Outline: Test
            Given <name> has <count> items
            Examples:
              | name | count |
              | Alice | 5 |
              | Bob | 100 |
        """
      When I format
      Then the output is:
        """
        Feature: Test

          Scenario Outline: Test
            Given <name> has <count> items

            Examples:
              | name  | count |
              | Alice | 5     |
              | Bob   | 100   |
        """

  Rule: Preserve special characters in cells

    Scenario: Escaped pipes in cells
      Given the input:
        """
        Feature: Test
          Scenario: Test
            Given data:
              | expression |
              | a \| b |
        """
      When I format
      Then the output is:
        """
        Feature: Test

          Scenario: Test
            Given data:
              | expression |
              | a \| b     |
        """
