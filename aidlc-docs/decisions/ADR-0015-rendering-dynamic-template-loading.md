# ADR-0015: Dynamic Plugin Loading for Rendering Service Animation Templates

## Status
Accepted

## Date
2026-08-07

## Stage
Low-Level Design (Unit 5: Rendering Service)

## Context
FR3.1 requires reusable Manim scene/component patterns for common programming-education illustrations: code with syntax highlight, step-by-step algorithm animation, data structure diagrams. Content Plugin Service (Unit 2) already assigns each scene an `animation_template_id` (`algorithm_visualization` or `concept_illustration` at MVP scope). Low-Level Design needed to decide how Rendering Service resolves that ID to an actual Manim `Scene` subclass to run.

## Options Considered
### Option A: Static mapping
- What it is: A plain Python dict (`{"algorithm_visualization": AlgorithmVisualizationScene, ...}`), mirroring TTS Service's `voice_registry.py`.
- Strengths: Simplest possible implementation; no discovery machinery.
- Trade-offs: Adding a new template requires editing the mapping/import list directly — acceptable at 2-template MVP scope, but less consistent with how the system already handles the analogous problem (Content Plugin Service's content-type plugins).

### Option B: Dynamic plugin loading (Chosen)
- What it is: Each animation template is its own plugin module under `adapters/rendering/templates/`, auto-discovered at startup — the same mechanism Content Plugin Service uses for content-type plugins (ADR-0006).
- Strengths: Consistent extensibility pattern across the two places in the system that need "pick an implementation by an ID coming from upstream classification" (Content Plugin's content-type plugins, and now Rendering's animation templates); adding a new template (e.g., a future "data_structure_diagram" category) means dropping in a new module, not editing a central mapping/import list.
- Trade-offs: More machinery than a static dict for only 2 templates today — accepted because the pattern is already proven and tested in Unit 2, and templates are expected to grow as more content-plugin categories are added (each new Content Plugin category will likely need a matching Rendering template).

## Decision
Dynamic plugin loading for Rendering Service's animation templates, mirroring Content Plugin Service's registry/discovery mechanism (ADR-0006).

## Rationale
The user chose consistency with the already-proven Content Plugin Service pattern over the smaller upfront simplicity of a static dict — since both problems are structurally the same ("resolve an ID from an upstream classification step to a concrete implementation"), reusing one mechanism keeps the codebase's extensibility story uniform rather than having two different answers to the same kind of problem.

## Consequences
- **Positive**: Adding new animation templates (as new Content Plugin categories are added) requires no changes to Rendering Service's core dispatch code — just a new template module.
- **Negative / Accepted Trade-offs**: More moving parts than a static dict for the current 2-template scope; template registry needs its own discovery/registration code (mirroring Unit 2's `ContentPluginRegistry`).
- **Follow-ups**: `code_snippet` handling (Manim's `Code` mobject for syntax highlight, Story B3) is orthogonal to template selection — every template renders it when present, regardless of which template plugin is chosen.

## Related
- Design artifact: `aidlc-docs/construction/rendering-service/low-level-design/module-structure.md`
- Related ADRs: Mirrors ADR-0006 (Dynamic Plugin Loading for Content Plugin Service)
