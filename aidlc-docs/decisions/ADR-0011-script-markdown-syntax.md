# ADR-0011: Script Syntax — Markdown with Scene Delimiters

## Status
Accepted

## Date
2026-08-07

## Stage
Low-Level Design (Unit 4: Script Processing Service)

## Context
FR2.1/FR2.2 and Story A2 require the system to parse a Creator-authored "script/markdown" into a structured scene list (`narration_text`, `illustration_hint`, `code_snippet?`), but no concrete grammar was defined at Inception (Application Design's `component-methods.md` only fixed the output scene schema, not the input syntax). Low-Level Design needed to pick one concrete, parseable format.

## Options Considered
### Option A: Markdown with scene delimiters (Chosen)
- What it is: Each scene is a `## Scene N` heading; a `> ` blockquote line under it is `illustration_hint`; plain text is `narration_text`; an optional fenced code block is `code_snippet`.
- Strengths: Matches Story A1's own description of the input as "script/markdown"; natural for a Creator to write by hand as flowing text (not a form); Python has simple parsing options for this narrow grammar (regex or a lightweight Markdown library).
- Trade-offs: Requires defining clear syntax-error messages (missing heading, empty narration, unclosed code fence) — addressed by `ScriptSyntaxError(line_number, reason)` in Question 8.

### Option B: YAML/JSON explicit structure
- What it is: `{ scenes: [{narration_text, illustration_hint, code_snippet}] }`, parsed with a standard library parser.
- Strengths: Trivial to parse — no custom grammar/parser to write or maintain.
- Trade-offs: Unnatural for authoring long narration text as flowing prose (Story A1 describes "soạn script" as writing text, not filling structured fields) — worse authoring experience, and doesn't match Story A1's own "script/markdown" framing.

## Decision
Option A — Markdown with `## Scene N` headings, `>` blockquote for illustration hints, and fenced code blocks for code snippets.

## Rationale
Story A1 already frames the input as "script/markdown" that Creators write directly in the GUI editor; a Markdown-based grammar keeps that authoring experience natural (flowing prose) while still being unambiguous enough to parse deterministically and produce precise syntax-error locations for Story A2's acceptance criteria.

## Consequences
- **Positive**: Natural authoring experience; deterministic parsing; clear error locations (line number + reason).
- **Negative / Accepted Trade-offs**: The grammar is bespoke (not a Markdown standard), so it must be documented for Creators (e.g., in GUI help text) — accepted since it stays simple (3 constructs: heading, blockquote, code fence).
- **Follow-ups**: If future FRs need richer per-scene metadata, the grammar will need extension (e.g., additional blockquote-prefixed lines) without breaking existing scripts (additive-only).

## Related
- Design artifact: `aidlc-docs/construction/script-processing-service/low-level-design/interface-contracts.md`
- Related ADRs: None
