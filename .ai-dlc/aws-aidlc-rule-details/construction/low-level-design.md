# Low-Level Design (LLD) - Detailed Steps

## Purpose
**Unit-level detailed design blueprint, produced before business logic, NFR, and infrastructure design**

Low-Level Design focuses on:
- Module/class/file structure for the unit
- Interface and API contracts (method signatures, request/response shapes) between internal parts of the unit
- Sequence/interaction diagrams for the unit's key operations
- Data flow within the unit at implementation-relevant detail

**Note**: This builds upon the macro architecture from High-Level Design and Application Design (INCEPTION phase). Business logic and domain rules are detailed afterward in Functional Design; NFR patterns in NFR Design; infrastructure mapping in Infrastructure Design (all per-unit, CONSTRUCTION phase).

## Prerequisites
- Units Generation must be complete
- Unit of work artifacts must be available
- High-Level Design / Application Design recommended (provides macro component structure)
- Execution plan must indicate Low-Level Design stage should execute

## Overview
Design the internal structure of the unit — modules, interfaces, and interaction sequences — so that Functional Design, NFR Design, and Infrastructure Design have a concrete skeleton to fill in.

## Steps to Execute

### Step 1: Analyze Unit Context
- Read unit definition from `aidlc-docs/inception/application-design/unit-of-work.md`
- Read `aidlc-docs/inception/high-level-design/architecture-overview.md` (if present)
- Understand unit responsibilities and boundaries

### Step 2: Create Low-Level Design Plan
- Generate plan with checkboxes [] for low-level design
- Focus on module structure, interface contracts, and sequence flows
- Each step should have a checkbox []

### Step 3: Generate Context-Appropriate Questions
**DIRECTIVE**: Thoroughly analyze the unit definition and high-level/application design artifacts to identify ALL areas where clarification would improve the low-level design. Be proactive in asking questions to ensure comprehensive understanding.

**CRITICAL**: Default to asking questions when there is ANY ambiguity or missing detail that could affect low-level design quality.

- EMBED questions using [Answer]: tag format
- Focus on ANY ambiguities, missing information, or areas needing clarification
- **When in doubt, ask the question** - overconfidence leads to poor designs

**Question categories to consider** (evaluate ALL categories):
- **Module/Class Structure** - Ask about internal module boundaries, class responsibilities, and file organization, consistent with the architectural style chosen in High-Level Design (`aidlc-docs/inception/high-level-design/architectural-style.md`, if present)
- **Layering & Dependency Direction** - **MANDATORY, always ask explicitly**: Confirm how the unit's internal layers/modules map to the chosen style (e.g., for Clean Architecture: domain/entities → use-cases → interface-adapters → frameworks-and-drivers) and which direction dependencies are allowed to point. If no system-level style was chosen in High-Level Design, ask the user to pick one for this unit (Layered, Clean/Onion, Hexagonal, DDD-style bounded model, or simple/none) rather than defaulting silently.
- **Dependency Injection** - **MANDATORY, always ask explicitly**: Ask whether the unit uses Dependency Injection, and if so:
  - Injection mechanism (constructor injection, setter/property injection, DI container/framework, manual factory/service locator)
  - What gets injected vs. constructed directly (e.g., inject repositories/external clients, construct value objects directly)
  - Whether interfaces/abstractions are injected (for testability/inversion of control) or concrete implementations
- **Interface Contracts** - Ask about method signatures, request/response shapes, and error contracts between internal parts
- **API Versioning** - **MANDATORY if this unit exposes an API consumed by other units/external clients**: Ask how breaking changes to the API are managed:
  - URI versioning (`/v1/...`, `/v2/...`) — explicit and cache-friendly, but requires clients to migrate paths
  - Header-based versioning (e.g., `Accept: application/vnd.api+json;version=2`) — keeps URIs stable, but less visible/discoverable
  - Payload/field-level versioning with backward-compatible additive changes only (never remove/rename fields, only add) — avoids version proliferation, but constrains schema evolution
  - Ask about the deprecation policy: how long old versions stay supported, and how consumers are notified before a version is retired
- **Distributed Tracing & Correlation** - **MANDATORY if this unit calls or is called by another unit/service**: Confirm how trace/correlation IDs propagate into and out of this unit (inbound header/message-metadata extraction, outbound propagation on every downstream call, inclusion in every log line). This must be designed at the interface level so it's not an afterthought bolted on during Code Generation.
- **Sequence Flows** - Ask about the order of operations for key use cases within the unit
- **Data Flow** - Ask about how data moves and transforms between internal modules
- **State Management** - Ask about where and how state is held within the unit (if applicable)

### Step 4: Store Plan
- Save as `aidlc-docs/construction/plans/{unit-name}-low-level-design-plan.md`
- Include all [Answer]: tags for user input

### Step 5: Collect and Analyze Answers
- Wait for user to complete all [Answer]: tags
- **MANDATORY**: Carefully review ALL responses for vague or ambiguous answers
- **CRITICAL**: Add follow-up questions for ANY unclear responses - do not proceed with ambiguity

### Step 6: Generate Low-Level Design Artifacts
- Create `aidlc-docs/construction/{unit-name}/low-level-design/module-structure.md` with:
  - Module/class/file breakdown and responsibilities
  - How modules map to the chosen architectural style's layers (domain, use-case/application, adapters/infrastructure, etc.)
  - Allowed dependency direction between layers
- Create `aidlc-docs/construction/{unit-name}/low-level-design/dependency-injection.md` with:
  - Whether DI is used, and the mechanism (constructor injection, DI container/framework, manual factory)
  - What is injected (interfaces/abstractions) vs. constructed directly
  - Composition root / wiring location for the unit
- Create `aidlc-docs/construction/{unit-name}/low-level-design/interface-contracts.md` with:
  - Method signatures and request/response shapes for internal interfaces
  - API versioning scheme for externally/cross-unit consumed endpoints, and deprecation policy
  - Correlation/trace ID propagation points (inbound extraction, outbound injection) for calls crossing unit boundaries
- Create `aidlc-docs/construction/{unit-name}/low-level-design/sequence-flows.md` with sequence diagrams (Mermaid, validated per content-validation.md) for the unit's key operations
- Per `common/architecture-decision-records.md`: create an ADR if multiple viable layering/dependency-injection approaches were considered and presented to the user. Reference the ADR number from `module-structure.md` or `dependency-injection.md`

### Step 7: Present Completion Message
- Present completion message in this structure:
     1. **Completion Announcement** (mandatory): Always start with this:

```markdown
# 🧩 Low-Level Design Complete - [unit-name]
```

     2. **AI Summary** (optional): Provide structured bullet-point summary of low-level design
        - Format: "Low-level design has created [description]:"
        - List key modules/classes and their responsibilities
        - List interface contracts defined
        - Mention key sequence flows covered
        - DO NOT include workflow instructions ("please review", "let me know", "proceed to next phase", "before we proceed")
        - Keep factual and content-focused
     3. **Formatted Workflow Message** (mandatory): Always end with this exact format:

```markdown
> **📋 <u>**REVIEW REQUIRED:**</u>**  
> Please examine the low-level design artifacts at: `aidlc-docs/construction/[unit-name]/low-level-design/`



> **🚀 <u>**WHAT'S NEXT?**</u>**
>
> **You may:**
>
> 🔧 **Request Changes** - Ask for modifications to the low-level design based on your review  
> ✅ **Continue to Next Stage** - Approve low-level design and proceed to **[next-stage-name]**

---
```

### Step 8: Wait for Explicit Approval
- Do not proceed until the user explicitly approves the low-level design
- Approval must be clear and unambiguous
- If user requests changes, update the design and repeat the approval process

### Step 9: Record Approval and Update Progress
- Log approval in audit.md with timestamp
- Record the user's approval response with timestamp
- Mark Low-Level Design stage complete in aidlc-state.md
