# Code standards and documentation comments

## 1. Comments document the code, not its history

A comment tells the reader what the code does and why it is built the way it is today. It never tells the story of how it got there.

- Write: what a module, type or function does; its contract (arguments, return value, errors raised, side effects); invariants; the technical reason behind a non-obvious choice ("keeps capitals inside a word so `conNguoi` matches its PALETTE key").
- Do not write: change request, requirement, ticket, review or unit numbers (`CR-012`, `FR-3`, `review C3`, `Unit 2`); what the code used to do, why it was changed, who asked for it; version changelogs; "fixed", "no longer", "used to", "previously". That history lives in git, `aidlc-docs/` and the ADRs.
- Allowed: one short `See ADR-00NN.` next to the code that carries out an architecture decision recorded there.
- No commented-out code. No `TODO` without the concrete work it stands for (see CLAUDE.md "No fake code").
- Keep comments in step with the code: a change that makes a comment wrong updates the comment in the same change.

Example:

- Bad: `// Hình minh hoạ (CR-046: promoted from a sub-state of Code; runs BEFORE Code)`
- Good: `// Hình minh hoạ: runs before Code, which reads its output.`

## 2. Documentation comments per language

Every public unit carries a documentation comment that says what it does. Describe arguments, return value and errors when they are not obvious from the names and types.

- Python: a docstring on every module, class and public function or method (PEP 257). The first line is a one-sentence summary; details follow after a blank line.
- Go: a doc comment on every package and every exported identifier, starting with the identifier's name (`// RenderScript renders ...`).
- TypeScript / React: a JSDoc `/** ... */` block on every exported function, component, hook and type.
- SQL in an embedded schema: a comment says what a table or a non-obvious column is for.

## 3. Basic code standards

- Format and lint clean before reporting work done: Python with the service's `ruff` config; Go with `gofmt` and `go vet`; TypeScript with `eslint`, `prettier` and `tsc --noEmit`.
- Names say what a thing is, in the language's convention (Python `snake_case`, Go `MixedCaps`, TypeScript `camelCase` / `PascalCase` for components and types). No cryptic abbreviations.
- A function does one thing. Prefer early returns over deep nesting.
- Never swallow an error: return it, raise it, or log it with context and handle it explicitly.
- No dead code, unused imports or repeated magic numbers and strings; give a repeated literal a named constant.
- Respect the hexagonal layout of each service: `domain` does not import `adapters`; `application` talks to adapters through ports.

## 4. Tests

- Test file names and test names describe the behaviour under test, not the change request that added it (`code_segment_repository_test.go`, not `code_segment_cr050_test.go`).
- Assertion messages describe what went wrong, not the requirement number.

## 5. Runtime strings

Log messages, error messages and UI text never contain change request, requirement or ticket numbers.
