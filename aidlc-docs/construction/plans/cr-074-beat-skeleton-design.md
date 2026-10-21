# CR-074 · Beat sheet in two phases: skeleton, then content

## Problem

The beat sheet step asked the model to write the whole sheet in one call. A small local model returned two beats (both Hook) for a story whose chain has six steps, with a total length far below the target. The number of beats, their functions and their facts are all already implied by the approved story; only the wording and the length of each beat need a model.

## Decision

The step has two phases, separated by an approval of the skeleton.

1. **Skeleton (no AI).** Built by code from the approved story: one Hook, one beat per line of the `chain` field, one Payoff. Each beat carries a function (by position), the facts its chain line cites (the payoff line's facts for the Payoff), and the story line it was built from (`step`). A beat other than the Hook with no fact is flagged `no_fact` and the screen marks it with "!". The Creator reviews functions and facts, may add, delete or reorder beats, and approves the skeleton. Durations are not decided here.
2. **Content (AI, one beat at a time, in order).** After the skeleton is approved, "AI viết beat sheet" calls the model once per beat. Each call sees the approved skeleton, the beats written so far, and a time budget, and returns title, goal, content, next and an estimated `dur`. Function and facts always stay as approved. Fields the Creator wrote keep their value. Nothing is stored until every beat is written, so one failed beat stores nothing.

After the last beat, the total of the `dur` values is brought inside the target length by code: when it already fits the values stand; otherwise every beat is scaled by one factor, rounded to the 5-second step, the Hook kept at most 20 seconds, and the rest adjusted in 5-second steps. `rebalanced` on the gate records how many beats changed so the screen can say so. If no sheet can fit (every beat already at the minimum and still over, or the chain needs more beats than a sheet holds or than the target length allows) the run fails with the reason and stores nothing.

A reply that cannot be read is asked for once more for that beat, with what was wrong in the note.

## Data

- `beat.content` gains `step` (the story line the beat was built from; empty for a beat the Creator added).
- `beat_sheet.content` gains `skeleton_approved` and `rebalanced`.
- `GET /beats` gains `phase` (`empty` | `skeleton` | `content`), `unwritten`, `rebalanced`, and `no_fact` on each beat. In the skeleton phase `checks` holds only the skeleton checks (Hook first, Payoff last, must facts held).

## API

- `POST /v1/projects/{id}/beats/skeleton`: builds (or rebuilds) the skeleton; 409 once it is approved.
- `POST /v1/projects/{id}/beats/skeleton/approve`: approves it.
- `POST /v1/projects/{id}/beats/generate`: 409 until the skeleton is approved; then writes the content as above.

## Not in this CR

Writing beats in parallel (a later beat reads the earlier ones, so the run is sequential), reopening an approved skeleton, and the mockup update for step 5 (a separate CR).

## Overlap

`cr-072-u3e-script-writing` (uncommitted) also changes `beat_desk.go` and `beat_units.go`; the beat data shape the script step reads is unchanged apart from the added fields.
