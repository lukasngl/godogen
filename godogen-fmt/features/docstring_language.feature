Feature: Doc String Language Annotations
  As a developer
  I want my doc string language hints preserved
  So my editor can provide syntax highlighting

  Rule: Language annotations are preserved

    Scenario: Go code block
      Given the input:
        ```
        Feature: Test
          Scenario: Example
            Given code:
              """go
              package main
              func main() {}
              """
        ```
      When I format
      Then the output is:
        ```
        Feature: Test

          Scenario: Example
            Given code:
              """go
              package main
              func main() {}
              """
        ```

    Scenario: Gherkin code block
      Given the input:
        ```
        Feature: Test
          Scenario: Example
            Given feature:
              """gherkin
              Feature: Nested
                Scenario: Inner
                  Given something
              """
        ```
      When I format
      Then the output is:
        ```
        Feature: Test

          Scenario: Example
            Given feature:
              """gherkin
              Feature: Nested
                Scenario: Inner
                  Given something
              """
        ```

    Scenario: JSON code block
      Given the input:
        ```
        Feature: Test
          Scenario: Example
            Given data:
              """json
              {"key": "value"}
              """
        ```
      When I format
      Then the output is:
        ```
        Feature: Test

          Scenario: Example
            Given data:
              """json
              {"key": "value"}
              """
        ```

    Scenario: Plain doc string without language
      Given the input:
        ```
        Feature: Test
          Scenario: Example
            Given text:
              """
              Some plain text
              """
        ```
      When I format
      Then the output is:
        ```
        Feature: Test

          Scenario: Example
            Given text:
              """
              Some plain text
              """
        ```
