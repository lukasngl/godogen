Feature: Pattern Anchors
  As a developer using godogen
  I want to be warned about missing regex anchors
  So that I can write more precise step patterns

  Scenario: Valid anchored pattern generates no warning
    Given a Go file "steps.go" with content:
      """go
      package example

      //godogen:given ^I have cucumbers$
      func (s *Suite) iHaveCucumbers() error {
          return nil
      }
      """
    When I run godogen
    Then the command should succeed
    And the command output should be empty
    And the file "steps_initialize.go" should exist

  Scenario: Missing start anchor warns
    Given a Go file "steps.go" with content:
      """go
      package example

      //godogen:given I have cucumbers$
      func (s *Suite) iHaveCucumbers() error {
          return nil
      }
      """
    When I run godogen
    Then the command should succeed
    And the command output should contain:
      """go
      steps.go:3:1: The pattern should start with '^' and end with '$'
      //godogen:given I have cucumbers$
      ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
      """
    And the file "steps_initialize.go" should exist

  Scenario: Missing end anchor warns
    Given a Go file "steps.go" with content:
      """go
      package example

      //godogen:given ^I have cucumbers
      func (s *Suite) iHaveCucumbers() error {
          return nil
      }
      """
    When I run godogen
    Then the command should succeed
    And the command output should contain:
      """go
      steps.go:3:1: The pattern should start with '^' and end with '$'
      //godogen:given ^I have cucumbers
      ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
      """
    And the file "steps_initialize.go" should exist

  Scenario: Missing both anchors warns
    Given a Go file "steps.go" with content:
      """go
      package example

      //godogen:given I have cucumbers
      func (s *Suite) iHaveCucumbers() error {
          return nil
      }
      """
    When I run godogen
    Then the command should succeed
    And the command output should contain:
      """go
      steps.go:3:1: The pattern should start with '^' and end with '$'
      //godogen:given I have cucumbers
      ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
      """
    And the file "steps_initialize.go" should exist
