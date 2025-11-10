Feature: Finding step definitions
  As a developer
  I want to find Go step implementations from feature files
  So I can navigate from steps to their implementations

  Rule: Step kind matching
    Given/When/Then patterns match their respective kinds
    Step (universal) patterns match all kinds

    Scenario: Find step definition for Given step
      Given steps.go is added to the workspace:
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
      When I request step definitions for example.feature line 2
      Then I get 1 result
      And the results are:
        | path     | line | column |
        | steps.go | 4    | 1      |

    Scenario: Find step definition for When step
      Given steps.go is added to the workspace:
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
      When I request step definitions for example.feature line 2
      Then I get 1 result
      And the results are:
        | path     | line | column |
        | steps.go | 4    | 1      |

    Scenario: Find step definition for Then step
      Given steps.go is added to the workspace:
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
      When I request step definitions for example.feature line 2
      Then I get 1 result
      And the results are:
        | path     | line | column |
        | steps.go | 4    | 1      |

    Scenario: Step kind matches Step universal pattern
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:step ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      And example.feature is added to the workspace:
        """
        Feature: Example
          Scenario: Test
            Given I have 5 cukes
            When I have 10 cukes
            Then I have 15 cukes
        """
      When I request step definitions for example.feature line 2
      Then I get 1 result
      When I request step definitions for example.feature line 3
      Then I get 1 result
      When I request step definitions for example.feature line 4
      Then I get 1 result

  Rule: And conjunction inheritance
    And/But steps inherit the kind from the previous step

    Scenario: Find step definition with And conjunction
      Given steps.go is added to the workspace:
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
      When I request step definitions for example.feature line 3
      Then I get 1 result
      And the results are:
        | path     | line | column |
        | steps.go | 4    | 1      |

  Rule: Pattern matching
    Regex patterns must match step text

    Scenario: No matching step definition
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes() {}
        """
      And example.feature is added to the workspace:
        """
        Feature: Example
          Scenario: Test
            Given I have bananas
        """
      When I request step definitions for example.feature line 2
      Then I get 0 results

    Scenario: Multiple step definitions match
      Given steps1.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes1() {}
        """
      And steps2.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have cukes$
        func IHaveCukes2() {}
        """
      And example.feature is added to the workspace:
        """
        Feature: Example
          Scenario: Test
            Given I have cukes
        """
      When I request step definitions for example.feature line 2
      Then I get 2 results

  Rule: Workspace file preference
    Workspace files override disk files for queries

    Scenario: Workspace file overrides disk file
      Given steps.go is added to the filesystem:
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
      When I request step definitions for example.feature line 2
      Then I get 1 result
      And the results are:
        | path     | line | column |
        | steps.go | 4    | 1      |

    Scenario: Closing workspace file shows disk file
      Given steps.go is added to the filesystem:
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
            Given I have disk cukes
        """
      And steps.go workspace version is removed
      When I request step definitions for example.feature line 2
      Then I get 1 result
      And the results are:
        | path     | line | column |
        | steps.go | 4    | 1      |

    Scenario: Disk file changes while workspace file exists
      Given steps.go is added to the filesystem:
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
      And steps.go is updated on the filesystem:
        """
        package steps

        //godogen:given ^I have new disk cukes$
        func IHaveNewDiskCukes() {}
        """
      And example.feature is added to the workspace:
        """
        Feature: Example
          Scenario: Test
            Given I have workspace cukes
        """
      When I request step definitions for example.feature line 2
      Then I get 1 result
      And the results are:
        | path     | line | column |
        | steps.go | 4    | 1      |
