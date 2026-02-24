Feature: Invalid Patterns
  As a developer using godogen
  I want to get errors for invalid patterns
  So that I can fix mistakes in my step definitions

  Scenario: Empty pattern reports error
    Given a Go file "steps.go" with content:
      """go
      package example

      //godogen:given
      func (s *Suite) emptyPattern() error {
          return nil
      }
      """
    When I run godogen
    Then the command should succeed
    And the command output should contain:
      """
      steps.go:3:1: pattern is empty
      //godogen:given
      ^^^^^^^^^^^^^^^
      """

  Scenario: Invalid regex reports error
    Given a Go file "steps.go" with content:
      """go
      package example

      //godogen:given ^(unclosed$
      func (s *Suite) invalidRegex() error {
          return nil
      }
      """
    When I run godogen
    Then the command should succeed
    And the command output should contain:
      """
      steps.go:3:1: regex pattern does not compile: error parsing regexp
      """

  Scenario: Whitespace-only pattern reports error
    Given a Go file "steps.go" with content:
      """go
      package example

      //godogen:given
      func (s *Suite) whitespacePattern() error {
          return nil
      }
      """
    When I run godogen
    Then the command should succeed
    And the command output should contain:
      """
      steps.go:3:1: pattern is empty
      """
