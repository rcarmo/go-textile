# Plan: Vendor php-textile tests + extend Go renderer

## Goals
- Vendor the php-textile test suite into this repo.
- Build a Go test harness to run those fixtures against our renderer.
- Extend the Go Textile parser to match php-textile behavior.

## Checklist

### 1) Vendor test suite
- [ ] Add php-textile fixtures under a local path (e.g., `test/fixtures/`).
- [ ] Include any referenced assets (e.g., images) needed by tests.
- [ ] Document the source/version of the vendored tests.

### 2) Repository organization
- [ ] Define a stable layout (e.g., `cmd/`, `internal/`, `pkg/`, `test/`).
- [ ] Move current code into the new structure with updated imports.
- [ ] Keep public API surface stable (or document any changes).

### 3) Test harness
- [ ] Parse YAML fixtures and run subtests for each case.
- [ ] Support fixture setup options (e.g., `setLite`, `setRestricted`, etc.).
- [ ] Provide clear failure diffs for expected vs. actual output.

### 4) Feature implementation (iterative)
- [ ] Block parsing parity (headers, lists, tables, blockquotes, code/pre, etc.).
- [ ] Phrase/inline parsing parity (emphasis, strong, code, links, images, glyphs).
- [ ] Attributes, classes/IDs, styles, alignment, and dimension handling.
- [ ] Raw HTML passthrough and restricted/lite modes.
- [ ] Link/image policies, rel handling, and prefixes.
- [ ] Line wrapping behavior and paragraph handling.

### 5) Verification
- [ ] Run full fixture suite locally.
- [ ] Track passing/failing fixtures and iterate until green.

## Notes
- Implementation will proceed by bringing up the fixture harness first, then tackling failures in batches.
