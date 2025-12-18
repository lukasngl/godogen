Feature: Indentation
  As a developer
  I want consistent indentation
  So my feature files are readable

  Rule: Standard indentation levels

    Scenario: Feature at column 0
      Given the input:
        ```
        Feature: Test
          Scenario: Test
            Given test
        ```
      When I format
      Then the output is:
        ```
        Feature: Test

          Scenario: Test
            Given test
        ```

    Scenario: Scenario indented 2 spaces
      Given the input:
        ```
        Feature: Test
        Scenario: Test
        Given test
        ```
      When I format
      Then the output is:
        ```
        Feature: Test

          Scenario: Test
            Given test
        ```

    Scenario: Steps indented 4 spaces
      Given the input:
        ```
        Feature: Test
          Scenario: Test
        Given first
           When second
              Then third
        ```
      When I format
      Then the output is:
        ```
        Feature: Test

          Scenario: Test
            Given first
            When second
            Then third
        ```

    Scenario: Background indented 2 spaces
      Given the input:
        ```
        Feature: Test
        Background:
        Given setup
          Scenario: Test
            Given test
        ```
      When I format
      Then the output is:
        ```
        Feature: Test

          Background:
            Given setup

          Scenario: Test
            Given test
        ```

    Scenario: Examples indented 4 spaces under Scenario Outline
      Given the input:
        ```
        Feature: Test
          Scenario Outline: Test
            Given <value>
        Examples:
        | value |
        | one   |
        ```
      When I format
      Then the output is:
        ```
        Feature: Test

          Scenario Outline: Test
            Given <value>

            Examples:
              | value |
              | one   |
        ```

  Rule: Data tables indented with step

    Scenario: Table after step
      Given the input:
        ```
        Feature: Test
          Scenario: Test
            Given users:
        | name  | age |
        | Alice | 30  |
        | Bob   | 25  |
        ```
      When I format
      Then the output is:
        ```
        Feature: Test

          Scenario: Test
            Given users:
              | name  | age |
              | Alice | 30  |
              | Bob   | 25  |
        ```

  Rule: Doc strings indented with step

    Scenario: Doc string after step
      Given the input:
        ```
        Feature: Test
          Scenario: Test
            Given content:
        """
        Some content
        """
        ```
      When I format
      Then the output is:
        ```
        Feature: Test

          Scenario: Test
            Given content:
              """
              Some content
              """
        ```
