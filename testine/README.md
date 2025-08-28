# testine

Testine provides helper APIs for seeding databases, capturing snapshots, and making assertions, helping reduce repetitive test setup.

## Features

* Load JSON fixture files from paths, glob patterns, or directories
* Optional document caching to avoid re-parsing fixtures in subtests
* Convenience methods for seeding, snapshotting, and assertions
* Integration with [`testequals`](https://github.com/calumari/testequals/) for rich diffs

## Usage

```go
func Test_Something(t *testing.T) {
    pt := poutine.New(mongodb.NewDriver(db))
    ti, err := testine.New(pt)
    if err != nil { t.Fatalf("failed to create test helper: %v", err) }
    ti.Cleanup(t) // cleanup after test

    // seed database from a fixture
    snap := ti.Seed(t, ti.LoadJSON(t, "testdata/seed.json"))

    // ... run code that modifies database ...
    snap.Assert(t) // assert no unintended mutations
    // or assert against a specific expected state
    ti.Assert(t, ti.LoadJSON(t, "testdata/after.json"))
}
```

## Document Caching

When the same fixtures are loaded in multiple subtests, caching avoids re-parsing:

```go
ti, _ := testine.New(pt, testine.WithDocumentCache())
```

## API

* **`Seed(t, doc) *Snapshot`** – Seed the database and capture the initial state for later comparison
* **`Assert(t, expectedDoc)`** – Capture a snapshot and compare against expected state
* **`Snapshot.Assert(t)`** – Compare the current database state against a previously captured snapshot
* **`Cleanup(t)`** – Register a test cleanup function
* **`LoadJSON(t, path)`** – Load JSON from a file, glob pattern, or directory, optionally using caching
