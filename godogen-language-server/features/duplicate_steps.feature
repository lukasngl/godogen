Feature: Duplicate Step Definitions
  As a developer
  I want to see errors for duplicate step definitions
  So I can avoid ambiguous behavior at runtime

  Rule: Duplicate patterns in same file are reported

    Scenario: Same pattern and kind in one file
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        ^ ERROR: Duplicate step definition
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) cukes$
        ^ ERROR: Duplicate step definition
        func IHaveCukesAgain(count int) {}
        """

    Scenario: Same pattern but different kinds
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:when ^I have (\d+) cukes$
        func WhenIHaveCukes(count int) {}
        """

    Scenario: Generic step duplicates specific kind
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        ^ ERROR: Duplicate step definition
        func IHaveCukes(count int) {}

        //godogen:step ^I have (\d+) cukes$
        ^ ERROR: Duplicate step definition
        func GenericCukes(count int) {}
        """

    Scenario: Different patterns are not duplicates
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) melons$
        func IHaveMelons(count int) {}
        """

  Rule: Duplicate patterns across files are reported

    Scenario: Same pattern in different files
      Given steps1.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      Then steps2.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        ^ ERROR: Duplicate step definition
        func IHaveCukesAgain(count int) {}
        """

    Scenario: Workspace file overrides disk file for duplication check
      Given steps.go is added to the filesystem:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      And other.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) melons$
        func IHaveMelonsAgain(count int) {}
        """
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) melons$
        ^ ERROR: Duplicate step definition
        func IHaveMelons(count int) {}
        """

  Rule: Invalid patterns are not checked for duplication

    Scenario: Invalid regex is not checked for duplicates
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+ cukes$
        ^ ERROR: regex pattern does not compile
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+ cukes$
        ^ ERROR: regex pattern does not compile
        func IHaveCukesAgain(count int) {}
        """

  Rule: Diagnostics update when duplicates are added or removed

    Scenario: Adding a duplicate step creates diagnostic
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        ^ ERROR: Duplicate step definition
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) cukes$
        ^ ERROR: Duplicate step definition
        func IHaveCukesAgain(count int) {}
        """

    Scenario: Removing a duplicate step removes diagnostic
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        ^ ERROR: Duplicate step definition
        func IHaveCukes(count int) {}

        //godogen:given ^I have (\d+) cukes$
        ^ ERROR: Duplicate step definition
        func IHaveCukesAgain(count int) {}
        """
      Then steps.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """

    Scenario: Removing duplicate from one file removes diagnostic from both files
      Given steps1.go is added to the workspace:
        """
        package steps

        //godogen:given ^I have (\d+) cukes$
        func IHaveCukes(count int) {}
        """
      Then steps2.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) cukes$
        ^ ERROR: Duplicate step definition
        func IHaveCukesAgain(count int) {}
        """
      Then steps2.go has the following diagnostics:
        """go
        package steps

        //godogen:given ^I have (\d+) melons$
        func IHaveMelons(count int) {}
        """
