# Godogen Scenario Isolation Patterns

## Three-Tier Testing

Run same scenarios against different backends:

```go
// feature_test.go

func TestHandler_(t *testing.T) {
    suite := testsuite.NewHandlerSuite()  // Fast, in-memory
    runFeatures(t, suite)
}

func TestPostgres_(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration tests")
    }
    suite := testsuite.NewPostgresSuite(t)  // Slow, production-like
    runFeatures(t, suite)
}

func runFeatures(t *testing.T, s testsuite.Suite) {
    godog.TestSuite{
        TestSuiteInitializer: s.InitializeSuite,
        ScenarioInitializer:  s.InitializeScenario,
        Options: &godog.Options{
            Format:   "pretty",
            Paths:    []string{"../../features"},
            TestingT: t,
            Strict:   true,
        },
    }.Run()
}
```

## Suite vs Scenario Initialization

**InitializeSuite** - called once, expensive resources:

```go
func (s *PostgresSuite) InitializeSuite(tsc *godog.TestSuiteContext) {
    tsc.BeforeSuite(func() {
        // Create testcontainer (expensive, once per suite)
        s.pool, _ = sqltest.NewPgPool(context.Background())
        runMigrations(s.pool)
    })
    tsc.AfterSuite(func() {
        s.pool.Close()
    })
}
```

**InitializeScenario** - called per scenario, cheap isolation:

```go
func (s *PostgresSuite) InitializeScenario(sc *godog.ScenarioContext) {
    state := &ScenarioState{}

    // 1. Register steps FIRST (before hooks)
    ScenarioInitializer(sc, state)

    // 2. Setup hook - runs before each scenario
    sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
        // Create per-scenario isolation
        state.tx, _ = s.pool.Begin(ctx)
        state.app = NewApp(state.tx)
        return ctx, nil
    })

    // 3. Cleanup hook - runs after each scenario
    sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
        return ctx, state.tx.Rollback()
    })
}
```

**Important:** Register steps BEFORE hooks. Godog executes in registration order, and hooks registered before steps may run before the scenario state is properly initialized.

## Transaction Rollback Isolation

Shared database, rollback after each scenario:

```go
sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
    state.tx, _ = s.pool.Begin(ctx)
    return ctx, nil
})

sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
    return ctx, state.tx.Rollback()  // Discard all changes
})
```

## Unique Table Prefixes

Per-scenario tables for event stores:

```go
sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
    tableName := "events_" + strings.ReplaceAll(uuid.NewString(), "-", "_")
    state.store = sqlstore.NewPostgres(s.pool, codec, tableName)
    state.store.Migrate(ctx)
    return ctx, nil
})
```

## Message Queue Isolation

Suite-level server, per-scenario prefixes:

```go
func (s *IntegrationSuite) InitializeSuite(tsc *godog.TestSuiteContext) {
    tsc.BeforeSuite(func() {
        s.natsServer, _ = natstest.NewServer(context.Background())
    })
    tsc.AfterSuite(func() {
        s.natsServer.Close()
    })
}

func (s *IntegrationSuite) InitializeScenario(sc *godog.ScenarioContext) {
    sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
        prefix := fmt.Sprintf("test.%s.", uuid.New().String())
        state.bus, _ = eventbus.NewJetStream(ctx, s.natsServer.URL(),
            eventbus.WithPrefix(prefix))
        return ctx, nil
    })
}
```

## Eventual Consistency

Use `Eventually` helper for async projections:

```go
func Eventually(condition func() bool, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if condition() {
            return nil
        }
        time.Sleep(10 * time.Millisecond)
    }
    return fmt.Errorf("condition not met within timeout")
}

//godogen:then ^the item should be visible$
func (s *State) ItemVisible(ctx context.Context) error {
    return Eventually(func() bool {
        result, _ := s.app.GetItem(ctx, s.itemID)
        return result != nil
    }, 500*time.Millisecond)
}
```

## Before/After Hooks via Godogen

Define hooks on the state struct:

```go
//godogen:before
func (s *State) SetupContext(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
    ctx, s.cancel = context.WithCancel(ctx)
    return ctx, nil
}

//godogen:after
func (s *State) Cleanup(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
    if s.cancel != nil {
        s.cancel()
    }
    return ctx, nil
}
```
