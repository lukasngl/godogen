Feature: Finding step references
  As a developer
  I want to find feature steps that use a Go pattern
  So I can see where step implementations are used

  Rule: Step kind matching
    Given/When/Then patterns match their respective kinds
    Step (universal) patterns match all kinds

    Scenario: Find references for Given step pattern
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      And example.feature is added to the workspace:
        """
        Feature: Example
          Scenario: Test
            Given I have 5 cukes
        """
      When I request step references for steps.go line 2
      Then I get 1 result
      And the results are:
        | path            | line | column |
        | example.feature | 3    | 5      |

    Scenario: Find references for When step pattern
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:when ^I add (\d+) cukes$
        func IAddCukes(count int) {}
        """
      And example.feature is added to the workspace:
        """
        Feature: Example
          Scenario: Test
            When I add 10 cukes
        """
      When I request step references for steps.go line 2
      Then I get 1 result
      And the results are:
        | path            | line | column |
        | example.feature | 3    | 5      |

    Scenario: Find references for Then step pattern
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:then ^I should have (\d+) cukes$
        func IShouldHaveCukes(count int) {}
        """
      And example.feature is added to the workspace:
        """
        Feature: Example
          Scenario: Test
            Then I should have 15 cukes
        """
      When I request step references for steps.go line 2
      Then I get 1 result
      And the results are:
        | path            | line | column |
        | example.feature | 3    | 5      |

    Scenario: Step universal pattern matches all kinds
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:step ^I do something$
        func IDoSomething() {}
        """
      And example.feature is added to the workspace:
        """
        Feature: Example
          Scenario: Test
            Given I do something
            When I do something
            Then I do something
        """
      When I request step references for steps.go line 2
      Then I get 3 results
      And the results are:
        | path            | line | column |
        | example.feature | 3    | 5      |
        | example.feature | 4    | 5      |
        | example.feature | 5    | 5      |

  Rule: And conjunction inheritance
    And/But steps inherit the kind from the previous step

    Scenario: And conjunction inherits previous step kind
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() {}
        """
      And example.feature is added to the workspace:
        """
        Feature: Example
          Scenario: Test
            Given I have cukes
            And I have cukes
        """
      When I request step references for steps.go line 2
      Then I get 2 results
      And the results are:
        | path            | line | column |
        | example.feature | 3    | 5      |
        | example.feature | 4    | 5      |

  Rule: Pattern matching
    Regex patterns must match step text

    Scenario: No references to pattern
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have bananas$
        func IHaveBananas() {}
        """
      And example.feature is added to the workspace:
        """
        Feature: Example
          Scenario: Test
            Given I have cukes
        """
      When I request step references for steps.go line 2
      Then I get 0 results

    Scenario: Multiple references to same pattern
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() {}
        """
      And example.feature is added to the workspace:
        """
        Feature: Example
          Scenario: First
            Given I have cukes
          Scenario: Second
            Given I have cukes
        """
      When I request step references for steps.go line 2
      Then I get 2 results
      And the results are:
        | path            | line | column |
        | example.feature | 3    | 5      |
        | example.feature | 5    | 5      |

    Scenario: References in multiple feature files
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() {}
        """
      And first.feature is added to the workspace:
        """
        Feature: First
          Scenario: Test
            Given I have cukes
        """
      And second.feature is added to the workspace:
        """
        Feature: Second
          Scenario: Test
            Given I have cukes
        """
      When I request step references for steps.go line 2
      Then I get 2 results

  Rule: Pattern comment references
    Clicking on a pattern comment returns references for that specific pattern only

    Scenario: Find references from specific pattern with multiple patterns
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        //godogen:when ^I add cukes$
        func IHaveCukes() {}
        """
      And example.feature is added to the workspace:
        """
        Feature: Example
          Scenario: Test
            Given I have cukes
            When I add cukes
        """
      When I request step references for steps.go line 2
      Then I get 1 result
      And the results are:
        | path            | line | column |
        | example.feature | 3    | 5      |

  Rule: Function declaration references
    Clicking on a function declaration returns references for all patterns on that function

    Scenario: Find references from function declaration
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      And example.feature is added to the workspace:
        """
        Feature: Example
          Scenario: Test
            Given I have 5 cukes
        """
      When I request step references for steps.go line 3 column 6
      Then I get 1 result
      And the results are:
        | path            | line | column |
        | example.feature | 3    | 5      |

    Scenario: Find references from function with multiple step patterns
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        //godogen:when ^I add cukes$
        func IHaveCukes() {}
        """
      And example.feature is added to the workspace:
        """
        Feature: Example
          Scenario: Test
            Given I have cukes
            When I add cukes
        """
      When I request step references for steps.go line 4 column 6
      Then I get 2 results
      And the results are:
        | path            | line | column |
        | example.feature | 3    | 5      |
        | example.feature | 4    | 5      |

    Scenario: No references from function body
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() {
            return nil
        }
        """
      And example.feature is added to the workspace:
        """
        Feature: Example
          Scenario: Test
            Given I have cukes
        """
      When I request step references for steps.go line 4 column 5
      Then I get 0 results

    Scenario Outline: Clicking on <scenario> of one-liner function
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() error { return nil }
        """
      And example.feature is added to the workspace:
        """
        Feature: Example
          Scenario: Test
            Given I have cukes
        """
      When I request step references for steps.go line 3 column <column>
      Then I get <count> results

      Examples:
        | scenario                | column | count |
        | func keyword            | 1      | 0     |
        | start of function name  | 6      | 1     |
        | middle of function name | 10     | 1     |
        | end of function name    | 15     | 1     |
        | opening parenthesis     | 16     | 0     |
        | return type             | 25     | 0     |
        | opening brace           | 32     | 0     |

    Scenario Outline: Clicking on <scenario> of multi-line function
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() {
            return nil
        }
        """
      And example.feature is added to the workspace:
        """
        Feature: Example
          Scenario: Test
            Given I have cukes
        """
      When I request step references for steps.go line <line> column <column>
      Then I get <count> results

      Examples:
        | scenario                | line | column | count |
        | func keyword            | 3    | 1      | 0     |
        | start of function name  | 3    | 6      | 1     |
        | middle of function name | 3    | 10     | 1     |
        | end of function name    | 3    | 15     | 1     |
        | opening parenthesis     | 3    | 16     | 0     |
        | closing parenthesis     | 3    | 17     | 0     |
        | opening brace           | 3    | 19     | 0     |
        | function body           | 4    | 5      | 0     |
        | closing brace           | 5    | 1      | 0     |

  Rule: Workspace file preference
    Workspace files override disk files for queries

    Scenario: Workspace Go file overrides disk file
      When steps.go is added to the filesystem:
        """
        package steps

        //godogen:given ^I have disk cukes$
        func IHaveDiskCukes() {}
        """
      And steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have workspace cukes$
        func IHaveWorkspaceCukes() {}
        """
      And example.feature is added to the workspace:
        """
        Feature: Example
          Scenario: Test
            Given I have workspace cukes
        """
      When I request step references for steps.go line 2
      Then I get 1 result
      And the results are:
        | path            | line | column |
        | example.feature | 3    | 5      |

    Scenario: Workspace feature file overrides disk file
      When steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() {}
        """
      And example.feature is added to the filesystem:
        """
        Feature: Example
          Scenario: Test
            Given I have disk cukes
        """
      And example.feature is added to the workspace:
        """
        Feature: Example
          Scenario: Test
            Given I have cukes
        """
      When I request step references for steps.go line 2
      Then I get 1 result
      And the results are:
        | path            | line | column |
        | example.feature | 3    | 5      |
