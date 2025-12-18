Feature: Tags and Comments
  As a developer
  I want tags and comments formatted correctly
  So I can organize and document my scenarios

  Rule: Tags are on their own line before the element

    Scenario: Feature tags
      Given the input:
        """
        @wip @smoke
        Feature: Test
        Scenario: Test
        Given test
        """
      When I format
      Then the output is:
        """
        @wip @smoke
        Feature: Test

          Scenario: Test
            Given test
        """

    Scenario: Scenario tags
      Given the input:
        """
        Feature: Test
        @slow @integration
        Scenario: Test
        Given test
        """
      When I format
      Then the output is:
        """
        Feature: Test

          @slow @integration
          Scenario: Test
            Given test
        """

    Scenario: Multiple tag lines
      Given the input:
        """
        Feature: Test
        @first
        @second
        Scenario: Test
        Given test
        """
      When I format
      Then the output is:
        """
        Feature: Test

          @first
          @second
          Scenario: Test
            Given test
        """

    Scenario: Tags on Examples
      Given the input:
        """
        Feature: Test
        Scenario Outline: Test
        Given <value>
        @dataset1
        Examples:
        | value |
        | one   |
        """
      When I format
      Then the output is:
        """
        Feature: Test

          Scenario Outline: Test
            Given <value>

            @dataset1
            Examples:
              | value |
              | one   |
        """

    Scenario: Tags on Rules
      Given the input:
        """
        Feature: Test
        @important
        Rule: Business rule
        Scenario: Test
        Given test
        """
      When I format
      Then the output is:
        """
        Feature: Test

          @important
          Rule: Business rule

            Scenario: Test
              Given test
        """

  Rule: Comments are preserved

    Scenario: Comment before feature
      Given the input:
        """
        # This is a comment
        Feature: Test
        Scenario: Test
        Given test
        """
      When I format
      Then the output is:
        """
        # This is a comment
        Feature: Test

          Scenario: Test
            Given test
        """

    Scenario: Comment before scenario
      Given the input:
        """
        Feature: Test
        # Scenario comment
        Scenario: Test
        Given test
        """
      When I format
      Then the output is:
        """
        Feature: Test

          # Scenario comment
          Scenario: Test
            Given test
        """

    Scenario: Inline comments on steps
      Given the input:
        """
        Feature: Test
        Scenario: Test
        Given test # this is important
        """
      When I format
      Then the output is:
        """
        Feature: Test

          Scenario: Test
            Given test # this is important
        """
