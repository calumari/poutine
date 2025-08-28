# poutine MongoDB Driver

MongoDB driver for the poutine testing library.

## Features

* Bulk insert operations for test data seeding
* Database snapshot capture as JSON documents
* ObjectID handling with `$oid` directives (wildcard or exact match)
* Built on the official [MongoDB v2 Go driver](https://github.com/mongodb/mongo-go-driver)

## Install

```bash
go get github.com/calumari/poutine/database/mongodb
```

## Usage

Get started with familiar MongoDB patterns:

```go
import (
    "testing"

    "github.com/calumari/poutine"
    "github.com/calumari/poutine/database/mongodb"
    "github.com/calumari/poutine/testine"
)

func Test_Something(t *testing.T) {
    // db is your *mongo.Database instance
    pt := poutine.New(mongodb.NewDriver(db))
    ti, err := testine.New(pt)
    if err != nil { t.Fatalf("failed to create test helper: %v", err) }
    ti.Cleanup(t)
    // ... mutate DB ...
    ti.Assert(t, ti.LoadJSON(t, "testdata/expected.json"))
}
```

## ObjectID Handling

Use `$oid` in JSON documents to represent MongoDB ObjectIDs:

```json
{
  "users": [
    {"_id": {"$oid": true}, "email": "a@example.com"},           // wildcard: any ObjectID
    {"_id": {"$oid": "507f1f77bcf86cd799439011"}, "email": "b@example.com"}  // exact match
  ]
}
```

* `{"$oid": true}` – matches any valid ObjectID
* `{"$oid": "hex_string"}` – matches a specific ObjectID
