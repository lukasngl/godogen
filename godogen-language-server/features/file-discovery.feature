Feature: File Discovery and Watching

  Rule: Discovery matches glob patterns

    Scenario: Discover files with ** pattern
      Given we watch pattern "**/*_steps.go"
      And file "steps.go" exists
      And file "foo/bar_steps.go" exists
      And file "foo/baz.go" exists
      When discovery runs
      Then "foo/bar_steps.go" should be indexed
      And "steps.go" should not be indexed
      And "foo/baz.go" should not be indexed

    Scenario: Discover files with specific directory pattern
      Given we watch pattern "tests/**/*.go"
      And file "tests/foo_test.go" exists
      And file "tests/integration/bar_test.go" exists
      And file "other/baz.go" exists
      When discovery runs
      Then "tests/foo_test.go" should be indexed
      And "tests/integration/bar_test.go" should be indexed
      And "other/baz.go" should not be indexed

    Scenario: Discover feature files
      Given we watch pattern "**/*.feature"
      And file "features/login.feature" exists
      And file "login.go" exists
      When discovery runs
      Then "features/login.feature" should be indexed
      And "login.go" should not be indexed

  Rule: Watch directories recursively

    Scenario: Watch all subdirectories for pattern with **
      Given we watch pattern "**/*_steps.go"
      And directory "foo" exists
      And directory "foo/bar" exists
      When discovery runs
      Then directory "." should be watched
      And directory "foo" should be watched
      And directory "foo/bar" should be watched

    Scenario: Watch only directories under pattern base
      Given we watch pattern "tests/**/*.go"
      And directory "tests" exists
      And directory "tests/integration" exists
      And directory "other" exists
      When discovery runs
      Then directory "tests" should be watched
      And directory "tests/integration" should be watched
      And directory "other" should not be watched

  Rule: Handle new directories

    Scenario: New directory is watched when created
      Given we watch pattern "**/*_steps.go"
      And discovery has run
      When directory "newdir" is created
      Then directory "newdir" should be watched

    Scenario: New nested directory is watched
      Given we watch pattern "**/*_steps.go"
      And discovery has run
      When directory "foo/bar/baz" is created
      Then directory "foo" should be watched
      And directory "foo/bar" should be watched
      And directory "foo/bar/baz" should be watched

    Scenario: Discover files in newly created directory
      Given we watch pattern "**/*_steps.go"
      And discovery has run
      When directory "newdir" is created with file "test_steps.go"
      Then "newdir/test_steps.go" should be indexed

  Rule: Handle file changes

    Scenario: New file matching pattern is indexed
      Given we watch pattern "**/*_steps.go"
      And discovery has run
      When file "new_steps.go" is created
      Then "new_steps.go" should be indexed

    Scenario: New file not matching pattern is ignored
      Given we watch pattern "**/*_steps.go"
      And discovery has run
      When file "other.go" is created
      Then "other.go" should not be indexed

    Scenario: Modified file is reindexed
      Given we watch pattern "**/*_steps.go"
      And file "test_steps.go" exists
      And discovery has run
      When file "test_steps.go" is modified
      Then "test_steps.go" should be reindexed

    Scenario: Deleted file is removed from index
      Given we watch pattern "**/*_steps.go"
      And file "test_steps.go" exists
      And discovery has run
      When file "test_steps.go" is deleted
      Then "test_steps.go" should be removed from index

  Rule: Skip hidden files and directories

    Scenario: Hidden directories are not watched
      Given we watch pattern "**/*_steps.go"
      And directory ".git" exists
      And directory ".vscode" exists
      And directory "src" exists
      When discovery runs
      Then directory ".git" should not be watched
      And directory ".vscode" should not be watched
      And directory "src" should be watched

    Scenario: Files in hidden directories are not indexed
      Given we watch pattern "**/*_steps.go"
      And file ".git/hooks/pre-commit_steps.go" exists
      And file "src/test_steps.go" exists
      When discovery runs
      Then ".git/hooks/pre-commit_steps.go" should not be indexed
      And "src/test_steps.go" should be indexed

    Scenario: Hidden files are not indexed
      Given we watch pattern "**/*.go"
      And file ".hidden.go" exists
      And file "visible.go" exists
      When discovery runs
      Then ".hidden.go" should not be indexed
      And "visible.go" should be indexed

    Scenario: New hidden directory is not watched
      Given we watch pattern "**/*_steps.go"
      And discovery has run
      When directory ".cache" is created
      Then directory ".cache" should not be watched

    Scenario: New file in hidden directory is not indexed
      Given we watch pattern "**/*_steps.go"
      And directory ".hidden" exists
      And discovery has run
      When file ".hidden/test_steps.go" is created
      Then ".hidden/test_steps.go" should not be indexed

  Rule: Multiple patterns

    Scenario: Files matching any pattern are indexed
      Given we watch pattern "**/*_steps.go"
      And we watch pattern "**/*.feature"
      And file "test_steps.go" exists
      And file "test.feature" exists
      And file "other.go" exists
      When discovery runs
      Then "test_steps.go" should be indexed
      And "test.feature" should be indexed
      And "other.go" should not be indexed

    Scenario: Watch union of all pattern directories
      Given we watch pattern "tests/**/*.go"
      And we watch pattern "features/**/*.feature"
      And directory "tests" exists
      And directory "features" exists
      And directory "other" exists
      When discovery runs
      Then directory "tests" should be watched
      And directory "features" should be watched
      And directory "other" should not be watched
