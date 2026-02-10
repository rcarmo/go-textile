# Tasks

## Plan: Vendor php-textile tests + extend Go renderer

### 1) Vendor test suite
- [x] Add php-textile fixtures under a local path (e.g., `test/fixtures/`).
- [x] Include any referenced assets (e.g., images) needed by tests.
- [x] Document the source/version of the vendored tests.

### 2) Repository organization
- [x] Define a stable layout (e.g., `cmd/`, `internal/`, `pkg/`, `test/`).
- [x] Move current code into the new structure with updated imports.
- [x] Keep public API surface stable (or document any changes).

### 3) Test harness
- [x] Parse YAML fixtures and run subtests for each case.
- [x] Support fixture setup options (e.g., `setLite`, `setRestricted`, etc.).
- [x] Provide clear failure diffs for expected vs. actual output.

### 4) Parser refactor (stdlib-first)
- [ ] Replace regex-heavy parsing with stdlib tools (bufio.Scanner, text/scanner, strings.Reader).
- [ ] Introduce tokenization (block/inline) using stdlib scanning.
- [ ] Ensure parser structure supports Textile-specific constructs without regex reliance.

### 5) Feature implementation (iterative)
- [ ] Block parsing parity (headers, lists, tables, blockquotes, code/pre, etc.).
- [ ] Phrase/inline parsing parity (emphasis, strong, code, links, images, glyphs).
- [ ] Attributes, classes/IDs, styles, alignment, and dimension handling.
- [ ] Raw HTML passthrough and restricted/lite modes.
- [ ] Link/image policies, rel handling, and prefixes.
- [ ] Line wrapping behavior and paragraph handling.

### 6) Verification
- [ ] Run full fixture suite locally.
- [ ] Track passing/failing fixtures and iterate until green.
