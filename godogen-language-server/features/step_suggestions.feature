Feature: Step Definition Suggestions
  As a developer
  I want to see suggestions when a step is undefined but similar definitions exist
  So I can quickly fix common mistakes like wrong keywords or typos

  Rule: Suggest when And/But inherits wrong kind

    Scenario: And inherits Given but When definition exists
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I am on the login page$
        func OnLoginPage() {}

        //godogen:when ^I click the submit button$
        func ClickSubmit() {}
        """
      Then test.feature has the following diagnostics:
        """gherkin
        Feature: Test
          Scenario: Login
            Given I am on the login page
            And I click the submit button
            ^ ERROR: No step definition found for: And I click the submit button
            ^ HINT: A matching 'When' step exists: ^I click the submit button$
        """

    Scenario: But inherits When but Then definition exists
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:when ^I submit the form$
        func SubmitForm() {}

        //godogen:then ^I should see an error$
        func SeeError() {}
        """
      Then test.feature has the following diagnostics:
        """gherkin
        Feature: Test
          Scenario: Error handling
            When I submit the form
            But I should see an error
            ^ ERROR: No step definition found for: But I should see an error
            ^ HINT: A matching 'Then' step exists: ^I should see an error$
        """

  Rule: Suggest when step kind doesn't match

    Scenario: Given step but only When definition exists
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:when ^I click the submit button$
        func ClickSubmit() {}
        """
      Then test.feature has the following diagnostics:
        """gherkin
        Feature: Test
          Scenario: Login
            Given I click the submit button
            ^ ERROR: No step definition found for: Given I click the submit button
            ^ HINT: A matching pattern exists but is defined as 'When': ^I click the submit button$
        """

  Rule: Suggest similar patterns for typos

    Scenario: Typo in step text suggests similar definition
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:when ^I click the submit button$
        func ClickSubmit() {}
        """
      Then test.feature has the following diagnostics:
        """gherkin
        Feature: Test
          Scenario: Login
            When I click the submitt button
            ^ ERROR: No step definition found for: When I click the submitt button
            ^ HINT: A step with a similar name exists: When ^I click the submit button$
        """

    Scenario: Multiple similar definitions all suggested as fixes
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:when ^I click the submit button$
        func ClickSubmit() {}

        //godogen:when ^I click the submot button$
        func ClickSubmot() {}

        //godogen:when ^I click the subnit button$
        func ClickSubnit() {}
        """
      And test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Login
            When I click the submet button
        """
      When I request diagnostics for test.feature
      Then diagnostic 1 fix title is "Change to 'When I click the submit button'"
      And diagnostic 1 fix title is "Change to 'When I click the submot button'"
      And diagnostic 1 fix title is "Change to 'When I click the subnit button'"

  Rule: No suggestion when no similar definitions exist

    Scenario: Completely different step has no suggestion
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:when ^I click the submit button$
        func ClickSubmit() {}
        """
      Then test.feature has the following diagnostics:
        """gherkin
        Feature: Test
          Scenario: Shopping
            When I buy groceries
            ^ ERROR: No step definition found for: When I buy groceries
        """

  Rule: Fixes produce correct output

    Scenario: Fix for keyword change (And -> When) produces correct output
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:given ^I am on the login page$
        func OnLoginPage() {}

        //godogen:when ^I click the submit button$
        func ClickSubmit() {}
        """
      And test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Login
            Given I am on the login page
            And I click the submit button
        """
      When I request diagnostics for test.feature
      Then diagnostic 1 fix title is "Change 'And' to 'When'"
      And diagnostic 1 fix applied to test.feature produces:
        """
        Feature: Test
          Scenario: Login
            Given I am on the login page
            When I click the submit button
        """

    Scenario: Fix for typo produces correct output
      Given steps.go is added to the workspace:
        """
        package steps

        //godogen:when ^I click the submit button$
        func ClickSubmit() {}
        """
      And test.feature is added to the workspace:
        """
        Feature: Test
          Scenario: Login
            When I click the submitt button
        """
      When I request diagnostics for test.feature
      Then diagnostic 1 fix title is "Change to 'When I click the submit button'"
      And diagnostic 1 fix applied to test.feature produces:
        """
        Feature: Test
          Scenario: Login
            When I click the submit button
        """
