# poutine

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.25-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/calumari/poutine)](https://goreportcard.com/report/github.com/calumari/poutine)

A Go library for database integration testing using JSON fixtures and snapshot assertions.

## Packages

* **`poutine`** – Core library providing a database-agnostic testing interface.
* **`database/mongodb`** – MongoDB driver implementation.
* **`testine`** – Utilities for loading fixtures, capturing snapshots, and cleaning up.

## Overview

Database integration tests can be tricky - like trying to eat poutine with a spoon. This library helps you **put in** your test data cleanly:

* Version-controlled JSON fixtures for consistent test data
* Snapshot assertions to compare current database state with expected data  
* Handling of generated values like ObjectIDs and timestamps
* Clean teardown so you don't leave a mess behind
* Automatic cleanup of test data
* Support for multiple backends through a driver interface

The goal is to reduce boilerplate and make test code easier to read and maintain, without adding extra complexity.

## Install

```bash
go get github.com/calumari/poutine
# Optional MongoDB driver
go get github.com/calumari/poutine/database/mongodb
```

## Quick Start (MongoDB)

```go
package mytest

import (
	"testing"

	"github.com/calumari/poutine"
	"github.com/calumari/poutine/database/mongodb"
	"github.com/calumari/poutine/testine"
)

func Test_UserFlow(t *testing.T) {
	// Acquire a *mongo.Database instance
	pt := poutine.New(mongodb.NewDriver(db))
	ti, err := testine.New(pt)
	if err != nil { 
		t.Fatalf("failed to create test helper: %v", err) 
	}
	ti.Cleanup(t) // register teardown

	// Seed DB from a JSON fixture
	snap := ti.Seed(t, ti.LoadJSON(t, "testdata/seed.json"))

	// ... run code that modifies the database ...
	snap.Assert(t) // check expected state

	// Or compare against a specific expected state
	ti.Assert(t, ti.LoadJSON(t, "testdata/after.json"))
}
```

## JSON Fixture Format

JSON fixtures are objects with collection (or table) names as keys and arrays of documents as values:

```jsonc
{
	"users": [
		{"_id": {"$oid": true}, "email": "a@example.com"},
		{"_id": {"$oid": "507f1f77bcf86cd799439011"}, "email": "b@example.com"}
	],
	"pets": [
		{"name": "Fido", "ownerEmail": "a@example.com"}
	]
}
```

* `{"$oid": true}` – matches any ObjectID (wildcard)
* `{"$oid": "hex_string"}` – matches a specific ObjectID

## Using `testine`

`testine.T` wraps a `Poutine` instance and provides helper methods:

* `Seed(t, doc) *Snapshot` – seed DB and capture snapshot
* `Assert(t, expectedDoc)` – compare current DB state to expected
* `Snapshot.Assert(t)` – compare current state to previously captured snapshot
* `Cleanup(t)` – register teardown
* `LoadJSON(t, path|glob|dir)` – load JSON from file, directory, or glob; supports caching with `testine.WithDocumentCache()`

## Custom Drivers

Implement the `database.Driver` interface to support new databases:

```go
type Driver interface {
    Seed(ctx context.Context, root jwalk.Document) (jwalk.Document, error)
    Snapshot(ctx context.Context) (jwalk.Document, error)
    Teardown(ctx context.Context) error
}
```

See [`database/mongodb/driver.go`](database/mongodb/driver.go) for a reference implementation.