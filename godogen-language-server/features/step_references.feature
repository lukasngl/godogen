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
