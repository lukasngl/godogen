Feature: Description Formatting
  As a developer
  I want my descriptions preserved with their formatting
  So I can use markdown-style documentation

  Rule: Feature descriptions preserve formatting

    Scenario: Simple feature description
      Given the input:
        ```
        Feature: My Feature
        This is a description.
        Scenario: Test
        Given test
        ```
      When I format
      Then the output is:
        ```
        Feature: My Feature
          This is a description.

          Scenario: Test
            Given test
        ```

    Scenario: Multi-line feature description
      Given the input:
        ```
        Feature: My Feature
        This feature handles:
        - First thing
        - Second thing
        - Third thing
        Scenario: Test
        Given test
        ```
      When I format
      Then the output is:
        ```
        Feature: My Feature
          This feature handles:
          - First thing
          - Second thing
          - Third thing

          Scenario: Test
            Given test
        ```

    Scenario: Feature description with code block
      Given the input:
        ```
        Feature: API Feature
        Example usage:
            curl -X POST /api/endpoint
            curl -X GET /api/endpoint
        Scenario: Test
        Given test
        ```
      When I format
      Then the output is:
        ```
        Feature: API Feature
          Example usage:
              curl -X POST /api/endpoint
              curl -X GET /api/endpoint

          Scenario: Test
            Given test
        ```

  Rule: Scenario descriptions preserve formatting

    Scenario: Scenario with description
      Given the input:
        ```
        Feature: Test
        Scenario: Complex scenario
        This scenario tests:
        1. First condition
        2. Second condition
        Given setup
        When action
        Then result
        ```
      When I format
      Then the output is:
        ```
        Feature: Test

          Scenario: Complex scenario
            This scenario tests:
            1. First condition
            2. Second condition
            Given setup
            When action
            Then result
        ```

  Rule: Empty lines in descriptions are preserved

    Scenario: Description with paragraph breaks
      Given the input:
        ```
        Feature: Test
        First paragraph.

        Second paragraph.
        Scenario: Test
        Given test
        ```
      When I format
      Then the output is:
        ```
        Feature: Test
          First paragraph.

          Second paragraph.

          Scenario: Test
            Given test
        ```
