# JSONType

JSONType is a CLI tool that analyzes one or more JSON files, infers their data types and structure, merges nested objects and arrays, and produces a clear, searchable description of the resulting JSON schema and type layout.

It is designed for exploration, debugging, reverse‑engineering unknown JSON, and documenting real‑world data formats that don’t come with schemas.

---

## Installation

```sh
go install github.com/4nd3r5on/jsontype/cmd/jsontype@latest
```

The binary will be installed as `jsontype` in your `$GOBIN`.

---

## Basic Usage

```sh
# Those three lines do the same
jsontype -file="./parseme.json"
jsontype -file ./parseme.json
```

Multiple files can be provided and will be merged into a single inferred structure:

```sh
# Those three lines do the same
go run ./cmd/jsontype -file="parse\ me.json parseme1.json parseme2.json"
go run ./cmd/jsontype -file="'parse me.json' parseme1.json parseme2.json"
go run ./cmd/jsontype -file="parse\ me.json,parseme1.json,parseme2.json"
```

Enable verbose diagnostics:

```sh
jsontype -file ./parseme.json -log-level debug
```

Write output to a file:

```sh
jsontype -file ./parseme.json -out schema.txt
```

---

## CLI Flags

```
-file value
    JSON file to parse (can be repeated or comma-separated)

-ignore-objects string
    Comma-separated JSON paths to ignore
    Example: "metadata,debug.info"

-parse-objects string
    Comma-separated JSON paths to explicitly parse
    Example: "users,data.items"

-max-depth int
    Maximum depth to parse (0 = unlimited)

-log-level string
    debug | info | warn | error (default: info)

-out string
    Output file (default: stdout)
```

---

## JSON Path Format

JSONType uses a simple, readable JSON path syntax to refer to specific locations in a document.

### Basic form

```
$.obj1.obj2.array[0].name
```

* `.` separates object keys
* `[0]` refers to a specific array index

### Wildcards

```
$.obj1.obj2.array[].id
```

`[]` means **any element** of:

* an array, or
* an object with integer keys

This is especially useful for describing homogeneous arrays or map-like objects.

### Root prefix

```
$
```

* `$.` is an optional root prefix
* Paths may be written with or without it

These are equivalent:

```
users[].id
$.users[].id
```

---

## Selective Parsing

### Parse only specific subtrees

```sh
jsontype -file data.json -parse-objects "users,events.items"
```

Only the listed paths will be analyzed; everything else is skipped.

### Ignore specific paths

```sh
jsontype -file data.json -ignore-objects "metadata,debug.info"
```

Ignored paths are completely excluded from inference and merging.

---

## Typical Use Cases

* Reverse‑engineering undocumented APIs
* Exploring logs or event streams
* Debugging unexpected JSON shape changes
* Creating documentation for real-world JSON formats
* Validating assumptions before writing parsers

---

## Example

```sh
jsontype -file response.json -ignore-objects "debug" -max-depth 5
```

Produces a compact, hierarchical description of the JSON structure, suitable for searching and comparison.

---

## License

MIT
