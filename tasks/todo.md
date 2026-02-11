# Tasks

## Project status
- Fixture suite is fully vendored and green.
- Parser refactor is complete and stdlib-first.
- Remaining work is maintenance and future feature parity checks if new fixtures are added.

## Refactor opportunities
- [ ] Unify inline attribute fragment scanning between block and link contexts.
- [ ] Consolidate quote handling into a shared classifier to avoid drift.
- [ ] Centralize dash/en-dash glyph conversion logic.
- [ ] Share URL sanitization policy logic across link/image helpers.
- [ ] Factor HTML tag scanning for block detection and wrapper parsing.
- [ ] Reuse table cell parsing logic with standard inline parsing.
- [ ] Extract note/footnote anchor construction helper.
- [x] Extract restricted-comment escaping helper.

## Completed milestones
### 1) Vendor test suite
- [x] Add php-textile fixtures under `test/fixtures/`.
- [x] Include referenced assets (e.g., `test/10x10.gif`).
- [x] Document source/version in `test/fixtures/README.md`.

### 2) Repository organization
- [x] Stabilize layout (`internal/`, `parser/`, `document/`, `test/`).
- [x] Move code with updated imports and module path.
- [x] Preserve the public API or document changes.

### 3) Test harness
- [x] Parse YAML fixtures with subtests.
- [x] Support fixture setup options (lite/restricted/etc.).
- [x] Provide clear expected vs. actual diffs.

### 4) Parser refactor (stdlib-first)
- [x] Replace regex-heavy parsing with stdlib scanning.
- [x] Tokenize block/inline structures without regex reliance.
- [x] Preserve Textile-specific constructs.

### 5) Feature parity
- [x] Block parsing parity (headers, lists, tables, blockquotes, code/pre).
- [x] Inline parsing parity (emphasis, strong, code, links, images, glyphs).
- [x] Attributes, classes/IDs, styles, alignment, dimensions.
- [x] Raw HTML passthrough and restricted/lite modes.
- [x] Link/image policies, rel handling, prefixes.
- [x] Line wrapping and paragraph handling.

### 6) Verification
- [x] Run full fixture suite locally.
- [x] Confirm all fixtures are green.
