Feature: Workspace File Handling
  As a developer
  I want the language server to handle edge cases in workspace files
  So I can work with various file states gracefully

  Rule: Empty feature files are not indexed

    Scenario: Empty feature file is not indexed
      Given empty.feature is added to the workspace:
        """
        """
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """

    Scenario: Feature file with only comments is not indexed
      Given comments.feature is added to the workspace:
        """
        # This is just a comment
        # No Feature declaration
        """
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """

    Scenario: Empty feature file alongside valid feature file
      Given empty.feature is added to the workspace:
        """
        """
      And test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Simple test
            Given I have 5 cukes
        """
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) melons$
        ^ HINT: Step definition is not used
        func IHaveMelons(count int) {}
        """
