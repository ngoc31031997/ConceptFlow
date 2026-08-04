# Code Generation - Detailed Steps

## Overview
This stage generates code for each unit of work through two integrated parts:
- **Part 1 - Planning**: Create detailed code generation plan with explicit steps
- **Part 2 - Generation**: Execute approved plan to generate code, tests, and artifacts

**Note**: For brownfield projects, "generate" means modify existing files when appropriate, not create duplicates.

## Prerequisites
- Unit Design Generation must be complete for the unit
- NFR Implementation (if executed) must be complete for the unit
- All unit design artifacts must be available
- Unit is ready for code generation

---

# PART 1: PLANNING

## Step 1: Analyze Unit Context
- [ ] Read unit design artifacts from Unit Design Generation
- [ ] Read unit story map to understand assigned stories
- [ ] Identify unit dependencies and interfaces
- [ ] Validate unit is ready for code generation

## Step 2: Create Detailed Unit Code Generation Plan
- [ ] Read workspace root and project type from `aidlc-docs/aidlc-state.md`
- [ ] Determine code location (see Critical Rules for structure patterns)
- [ ] **Brownfield only**: Review reverse engineering code-structure.md for existing files to modify
- [ ] Document exact paths (never aidlc-docs/)
- [ ] Create explicit steps for unit generation:
  - Project Structure Setup (greenfield only)
  - Business Logic Generation
  - Business Logic Unit Testing
  - Business Logic Summary
  - API Layer Generation
  - API Layer Unit Testing
  - API Layer Summary
  - Repository Layer Generation
  - Repository Layer Unit Testing
  - Repository Layer Summary
  - Frontend Components Generation (if applicable)
  - Frontend Components Unit Testing (if applicable)
  - Frontend Components Summary (if applicable)
  - Database Migration Scripts (if data models exist)
  - Documentation Generation (API docs, README updates — see README Requirements below)
  - Deployment Artifacts Generation
- [ ] Number each step sequentially
- [ ] Include story mapping references
- [ ] Add checkboxes [ ] for each step

## Step 3: Include Unit Generation Context
- [ ] For this unit, include:
  - Stories implemented by this unit
  - Dependencies on other units/services
  - Expected interfaces and contracts
  - Database entities owned by this unit
  - Service boundaries and responsibilities

## Step 3.5: Confirm Coding Standards (MANDATORY)
**DIRECTIVE**: Before code generation begins, coding standards must be explicit, not assumed. Check `aidlc-docs/construction/{unit-name}/low-level-design/module-structure.md` and `dependency-injection.md` (if present) for style/DI decisions already made in Low-Level Design, and reuse them here rather than asking again. If Low-Level Design was skipped, or details below are still undecided, ask the following using [Answer] tags in the unit plan document:

- **Language style guide**: Which naming convention applies for this language/stack? (e.g., TypeScript/JS: camelCase variables & functions, PascalCase classes/types; Python: snake_case functions & variables, PascalCase classes; Java/C#: camelCase members, PascalCase classes) — confirm the default or note a project-specific override
- **SOLID enforcement**: Confirm SOLID principles are mandatory for this unit's business logic and component code (see Coding Standards & SOLID Compliance in Critical Rules below) — ask only if the user may want exceptions (e.g., throwaway scripts, generated boilerplate)
- **Documentation style**: Which doc-comment convention applies? (e.g., JSDoc for JS/TS, docstrings for Python, Javadoc for Java, XML doc comments for C#) and expected coverage (public APIs only vs. all non-trivial methods)
- **Linting/formatting tooling**: Is there an existing linter/formatter config in the repo (ESLint/Prettier, Black/Flake8, Checkstyle, etc.) to conform to? If brownfield, detect and follow the existing config instead of asking.

Store answers in the unit plan document; do not proceed to generation with unresolved [Answer] tags.

## Step 4: Create Unit Plan Document
- [ ] Save complete plan as `aidlc-docs/construction/plans/{unit-name}-code-generation-plan.md`
- [ ] Include step numbering (Step 1, Step 2, etc.)
- [ ] Include unit context and dependencies
- [ ] Include story traceability
- [ ] Ensure plan is executable step-by-step
- [ ] Emphasize that this plan is the single source of truth for Code Generation

## Step 5: Summarize Unit Plan
- [ ] Provide summary of the unit code generation plan to the user
- [ ] Highlight unit generation approach
- [ ] Explain step sequence and story coverage
- [ ] Note total number of steps and estimated scope

## Step 6: Log Approval Prompt
- [ ] Before asking for approval, log the prompt with timestamp in `aidlc-docs/audit.md`
- [ ] Include reference to the complete unit code generation plan
- [ ] Use ISO 8601 timestamp format

## Step 7: Wait for Explicit Approval
- [ ] Do not proceed until the user explicitly approves the unit code generation plan
- [ ] Approval must cover the entire plan and generation sequence
- [ ] If user requests changes, update the plan and repeat approval process

## Step 8: Record Approval Response
- [ ] Log the user's approval response with timestamp in `aidlc-docs/audit.md`
- [ ] Include the exact user response text
- [ ] Mark the approval status clearly

## Step 9: Update Progress
- [ ] Mark Code Generation Part 1 (Planning) complete in `aidlc-state.md`
- [ ] Update the "Current Status" section
- [ ] Prepare for transition to Code Generation

---

# PART 2: GENERATION

## Step 10: Load Unit Code Generation Plan
- [ ] Read the complete plan from `aidlc-docs/construction/plans/{unit-name}-code-generation-plan.md`
- [ ] Identify the next uncompleted step (first [ ] checkbox)
- [ ] Load the context for that step (unit, dependencies, stories)

## Step 11: Execute Current Step
- [ ] Verify target directory from plan (never aidlc-docs/)
- [ ] **Brownfield only**: Check if target file exists
- [ ] Generate exactly what the current step describes:
  - **If file exists**: Modify it in-place (never create `ClassName_modified.java`, `ClassName_new.java`, etc.)
  - **If file doesn't exist**: Create new file
- [ ] Write to correct locations:
  - **Application Code**: Workspace root per project structure
  - **Documentation**: `aidlc-docs/construction/{unit-name}/code/` (markdown only)
  - **Build/Config Files**: Workspace root
- [ ] Follow unit story requirements
- [ ] Respect dependencies and interfaces
- [ ] Apply the Coding Standards & SOLID Compliance rules (see Critical Rules) to all generated code
- [ ] Apply the naming convention, documentation style, and DI mechanism confirmed in Step 3.5 / Low-Level Design
- [ ] If Low-Level Design defined an API versioning scheme or correlation/trace ID propagation for this unit's interfaces (`interface-contracts.md`), implement it in the generated API layer/client code — not as an afterthought

## Step 12: Update Progress
- [ ] Mark the completed step as [x] in the unit code generation plan
- [ ] Mark associated unit stories as [x] when their generation is finished
- [ ] Update `aidlc-docs/aidlc-state.md` current status
- [ ] **Brownfield only**: Verify no duplicate files created (e.g., no `ClassName_modified.java` alongside `ClassName.java`)
- [ ] Save all generated artifacts

## Step 13: Continue or Complete Generation
- [ ] If more steps remain, return to Step 10
- [ ] If all steps complete, proceed to present completion message

## Step 14: Present Completion Message
- Present completion message in this structure:
     1. **Completion Announcement** (mandatory): Always start with this:

```markdown
# 💻 Code Generation Complete - [unit-name]
```

     2. **AI Summary** (optional): Provide structured bullet-point summary
        - **Brownfield**: Distinguish modified vs created files (e.g., "• Modified: `src/services/user-service.ts`", "• Created: `src/services/auth-service.ts`")
        - **Greenfield**: List created files with paths (e.g., "• Created: `src/services/user-service.ts`")
        - List tests, documentation, deployment artifacts with paths
        - Keep factual, no workflow instructions
     3. **Formatted Workflow Message** (mandatory): Always end with this exact format:

```markdown
> **📋 <u>**REVIEW REQUIRED:**</u>**  
> Please examine the generated code at:
> - **Application Code**: `[actual-workspace-path]`
> - **Documentation**: `aidlc-docs/construction/[unit-name]/code/`



> **🚀 <u>**WHAT'S NEXT?**</u>**
>
> **You may:**
>
> 🔧 **Request Changes** - Ask for modifications to the generated code based on your review  
> ✅ **Continue to Next Stage** - Approve code generation and proceed to **[next-unit/Build & Test]**

---
```

## Step 15: Wait for Explicit Approval
- Do not proceed until the user explicitly approves the generated code
- Approval must be clear and unambiguous
- If user requests changes, update the code and repeat the approval process

## Step 16: Record Approval and Update Progress
- Log approval in audit.md with timestamp
- Record the user's approval response with timestamp
- Mark Code Generation stage as complete for this unit in aidlc-state.md

---

## Critical Rules

### Code Location Rules
- **Application code**: Workspace root only (NEVER aidlc-docs/)
- **Documentation**: aidlc-docs/ only (markdown summaries)
- **Read workspace root** from aidlc-state.md before generating code

**Structure patterns by project type**:
- **Brownfield**: Use existing structure (e.g., `src/main/java/`, `lib/`, `pkg/`)
- **Greenfield single unit**: `src/`, `tests/`, `config/` in workspace root
- **Greenfield multi-unit (microservices)**: `{unit-name}/src/`, `{unit-name}/tests/`
- **Greenfield multi-unit (monolith)**: `src/{unit-name}/`, `tests/{unit-name}/`

### Brownfield File Modification Rules
- Check if file exists before generating
- If exists: Modify in-place (never create copies like `ClassName_modified.java`)
- If doesn't exist: Create new file
- Verify no duplicate files after generation (Step 12)

### Planning Phase Rules
- Create explicit, numbered steps for all generation activities
- Include story traceability in the plan
- Document unit context and dependencies
- Get explicit user approval before generation

### Generation Phase Rules
- **NO HARDCODED LOGIC**: Only execute what's written in the unit plan
- **FOLLOW PLAN EXACTLY**: Do not deviate from the step sequence
- **UPDATE CHECKBOXES**: Mark [x] immediately after completing each step
- **STORY TRACEABILITY**: Mark unit stories [x] when functionality is implemented
- **RESPECT DEPENDENCIES**: Only implement when unit dependencies are satisfied

### Coding Standards & SOLID Compliance (MANDATORY)
Applies to all generated business logic, component, and service code (not generated boilerplate/config unless the user opted out in Step 3.5).

**SOLID principles** — verify each before marking a code-generation step [x]:
- **S — Single Responsibility**: Each class/module has exactly one reason to change; split classes that mix concerns (e.g., business logic + persistence + presentation)
- **O — Open/Closed**: Prefer extension points (interfaces, strategy/polymorphism) over editing existing logic with conditionals for new cases
- **L — Liskov Substitution**: Subtypes/implementations must be substitutable for their base type/interface without breaking callers' expectations
- **I — Interface Segregation**: Prefer several small, focused interfaces over one large interface clients are forced to depend on in full
- **D — Dependency Inversion**: High-level modules depend on abstractions (interfaces), not concrete low-level implementations; wire concrete implementations via the DI mechanism confirmed in Low-Level Design/Step 3.5

**General OOP hygiene**:
- Encapsulation: no public mutable fields where a method/property should mediate access; keep internal state private
- Composition over inheritance unless an "is-a" relationship with shared behavior genuinely applies
- Avoid god classes/functions — if a method exceeds ~40-50 lines or handles multiple unrelated steps, extract helper methods/classes

**Naming conventions**:
- Follow the convention confirmed in Step 3.5 (or the language's idiomatic default if unspecified: e.g., PascalCase for classes/types, camelCase for variables/functions in TS/Java/C#; snake_case for Python variables/functions)
- Names must be descriptive and unabbreviated except for well-known idioms (`i` in a tight loop, `id`, `ctx`) — no single-letter names for anything with meaningful scope
- Boolean names read as predicates (`isValid`, `hasPermission`, not `valid`, `flag`)

**Documentation**:
- Use the doc-comment convention confirmed in Step 3.5 (JSDoc/docstring/Javadoc/etc.) on all public classes, interfaces, and methods
- Document the WHY (non-obvious constraints, invariants) not the WHAT — do not restate what well-named code already shows
- Keep module-level docs in `aidlc-docs/construction/{unit-name}/code/` in sync with what was actually generated (per Code Location Rules)

**Verification before marking a step [x]**:
- [ ] Every new/modified class has a single, clearly stated responsibility
- [ ] Dependencies are injected via abstractions, not constructed/imported as concrete implementations inside business logic
- [ ] Naming and documentation follow the confirmed conventions
- [ ] No god classes/functions introduced

### README Requirements (MANDATORY)
When the code generation plan reaches "Documentation Generation" (per unit, or once at the workspace root for single-unit/greenfield projects), the project `README.md` at the workspace root MUST be created (greenfield) or updated (brownfield — merge in, never overwrite unrelated existing sections) with at minimum:

- **Project overview**: one-paragraph summary of what the project/unit does
- **Prerequisites**: required runtime/tooling versions (e.g., Node version, JDK version, Docker)
- **Installation**: exact commands to install dependencies (e.g., `npm install`, `mvn install`, `pip install -r requirements.txt`)
- **Configuration**: required environment variables / config files, with example values (never real secrets)
- **Running the project**: exact commands to run locally (dev server, main entrypoint) and, if applicable, via Docker/docker-compose
- **Running tests**: exact commands for unit tests, and integration/e2e tests if present (mirror `aidlc-docs/construction/build-and-test/*-instructions.md` once Build and Test has run; before that, use the commands from the unit's build/test setup)
- **Project structure**: brief map of top-level directories and their purpose
- **Link to CI/CD**: reference to `aidlc-docs/construction/build-and-test/ci-cd-integration-instructions.md` for pipeline, SonarQube, and OWASP scanning setup once that stage has run

For multi-unit projects, keep one root `README.md` with the sections above at the whole-system level, and per-unit `README.md` files (if the unit has its own deployable structure) covering the same sections scoped to that unit.

### Automation Friendly Code Rules
When generating UI code (web, mobile, desktop), ensure elements are automation-friendly:
- Add `data-testid` attributes to interactive elements (buttons, inputs, links, forms)
- Use consistent naming: `{component}-{element-role}` (e.g., `login-form-submit-button`, `user-list-search-input`)
- Avoid dynamic or auto-generated IDs that change between renders
- Keep `data-testid` values stable across code changes (only change when element purpose changes)

## Completion Criteria
- Complete unit code generation plan created and approved
- Coding standards confirmed (Step 3.5): naming convention, SOLID enforcement, documentation style, linting/formatting tooling
- All steps in unit code generation plan marked [x]
- All unit stories implemented according to plan
- Generated code passes the Coding Standards & SOLID Compliance verification checklist
- All code and tests generated (tests will be executed in Build & Test phase)
- Deployment artifacts generated
- Complete unit ready for build and verification
