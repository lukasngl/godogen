Feature: Pattern Groups
  As a developer using godogen
  I want capturing groups to match function parameters
  So that I can extract values from step text

  Scenario: Zero groups with zero params
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

  Scenario: One group with one param
    Given a Go file "steps.go" with content:
      """go
      package example

      //godogen:given ^I have (\d+) cucumbers$
      func (s *Suite) iHaveCucumbers(count int) error {
          return nil
      }
      """
    When I run godogen
    Then the command should succeed
    And the command output should be empty

  Scenario: Two groups with two params
    Given a Go file "steps.go" with content:
      """go
      package example

      //godogen:given ^I have (\d+) cucumbers and (\d+) tomatoes$
      func (s *Suite) iHaveVegetables(cucumbers, tomatoes int) error {
          return nil
      }
      """
    When I run godogen
    Then the command should succeed
    And the command output should be empty

  Scenario: Non-capturing group does not count
    Given a Go file "steps.go" with content:
      """go
      package example

      //godogen:given ^I (?:have|own) (\d+) cucumbers$
      func (s *Suite) iHaveCucumbers(count int) error {
          return nil
      }
      """
    When I run godogen
    Then the command should succeed
    And the command output should be empty

  Scenario: Named group counts as one
    Given a Go file "steps.go" with content:
      """go
      package example

      //godogen:given ^I have (?P<count>\d+) cucumbers$
      func (s *Suite) iHaveCucumbers(count int) error {
          return nil
      }
      """
    When I run godogen
    Then the command should succeed
    And the command output should be empty
