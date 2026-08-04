# High-Level Design (HLD) - Detailed Steps

## Purpose
**System-wide macro architecture, defined before any component-level detail**

High-Level Design focuses on:
- System context: external actors, external systems, and boundaries
- Major components/services and their responsibilities at a macro level
- Technology stack direction (languages, frameworks, platforms) at the system level
- Integration points and data flow between major components
- Deployment topology at a conceptual level (not infrastructure specifics)

**Note**: Detailed component interfaces, methods, and business rules happen later in Application Design and Functional Design. Infrastructure specifics happen later in Infrastructure Design (per-unit, CONSTRUCTION phase).

## Prerequisites
- Workspace Detection must be complete
- Requirements Analysis recommended (provides functional context)
- User Stories recommended (user stories guide architectural decisions)
- Execution plan must indicate High-Level Design stage should execute

## Step-by-Step Execution

### 1. Analyze Context
- Read `aidlc-docs/inception/requirements/requirements.md` and `aidlc-docs/inception/user-stories/stories.md` (if present)
- Read reverse engineering artifacts (if brownfield)
- Identify system boundaries, external actors, and external systems
- Determine design scope and complexity

### 2. Create High-Level Design Plan
- Generate plan with checkboxes [] for high-level design
- Focus on system context, major components, technology direction, and integration boundaries
- Each step and sub-step should have a checkbox []

### 3. Include Mandatory Design Artifacts in Plan
- **ALWAYS** include these mandatory artifacts in the design plan:
  - [ ] Generate system-context.md with actors, external systems, and system boundary
  - [ ] Generate architecture-overview.md with major component/service map and responsibilities
  - [ ] Generate technology-direction.md with technology stack direction and rationale
  - [ ] Generate integration-boundaries.md with integration points and macro data flow
  - [ ] Generate architectural-style.md with the chosen system-organization style (e.g., Layered/N-tier, Clean Architecture, Hexagonal/Ports & Adapters, Domain-Driven Design, Microservices) and rationale
  - [ ] Validate design completeness and consistency

### 4. Generate Context-Appropriate Questions
**DIRECTIVE**: Analyze the requirements and stories to generate questions relevant to THIS specific system's architecture. Use the categories below as guidance. Evaluate each category and, when in doubt about applicability, ask the question rather than skipping it — overconfidence leads to poor outcomes (see overconfidence-prevention.md).

- EMBED questions using [Answer]: tag format
- Focus on ANY ambiguities, missing information, or areas needing clarification
- Generate questions wherever user input would improve architectural decisions
- **When in doubt, ask the question** - overconfidence leads to poor designs

**Question categories to evaluate** (consider ALL categories):
- **System Context** - Ask about external actors, external systems, and system boundary
- **Major Components** - Ask about how the system should be macro-decomposed (e.g., monolith vs. services, layering strategy)
- **Architectural Style** - **MANDATORY, always ask explicitly**: Ask the user to choose (or confirm) the system-organization style, e.g.:
  - Layered / N-tier (Presentation → Business/Service → Data Access)
  - Clean Architecture / Onion Architecture (dependency rule: outer layers depend on inner layers only)
  - Hexagonal / Ports & Adapters
  - Domain-Driven Design (bounded contexts, aggregates, domain model isolated from infrastructure)
  - Microservices (per bounded context/capability)
  - Other / hybrid — describe
  - Never assume a default; if the user has no preference, propose one based on requirements/complexity and ask for confirmation
- **Technology Direction** - Ask about language/framework/platform preferences and constraints
- **Integration Boundaries** - Ask about synchronous vs. asynchronous integration, external API dependencies
- **Distributed System & Inter-Service Communication** - **MANDATORY if more than one component/service is identified, always ask explicitly**:
  - Communication style between components: synchronous request/response (REST/gRPC) vs. asynchronous event-driven (message broker/event bus) vs. hybrid
  - If asynchronous/event-driven: choreography (services react to events independently) vs. orchestration (central coordinator/workflow engine)
  - Distributed transaction/consistency approach for multi-service business operations: 2PC (rare/avoid), Saga pattern (orchestration or choreography-based), or eventual consistency with compensating actions
  - Data ownership: confirm each data entity is owned by exactly one service/component (no shared database across service boundaries) — flag and resolve any ambiguity
  - Consistency expectation per cross-service operation: strong consistency required, or eventual consistency acceptable (and what staleness window is tolerable)
  - Never assume request/response + shared database as a default for a multi-service system; ask explicitly and record the rationale in `integration-boundaries.md`
  - **API Gateway** - If more than one externally-callable service/component exists: ask whether a single API Gateway (unified entry point, routing, auth, rate limiting) fronts all services, or each service is called directly. Never assume one silently — record the decision even if the answer is "no gateway, direct calls"
- **Non-Functional Drivers** - Ask about scale, availability, or compliance drivers that shape the architecture (detailed NFR design happens later, per-unit)
- **Deployment Topology** - Ask about conceptual deployment model (single region, multi-region, on-prem, cloud, hybrid)

### 5. Store High-Level Design Plan
- Save as `aidlc-docs/inception/plans/high-level-design-plan.md`
- Include all [Answer]: tags for user input

### 6. Request User Input
- Ask user to fill [Answer]: tags directly in the plan document
- Emphasize importance of architectural decisions at this stage
- Provide clear instructions on completing the [Answer]: tags

### 7. Collect Answers
- Wait for user to provide answers to all questions using [Answer]: tags in the document
- Do not proceed until ALL [Answer]: tags are completed
- Review the document to ensure no [Answer]: tags are left blank

### 8. ANALYZE ANSWERS (MANDATORY)
Before proceeding, you MUST carefully review all user answers for:
- **Vague or ambiguous responses**: "mix of", "somewhere between", "not sure", "depends"
- **Undefined criteria or terms**: References to concepts without clear definitions
- **Contradictory answers**: Responses that conflict with each other
- **Missing design details**: Answers that lack specific guidance
- **Answers that combine options**: Responses that merge different approaches without clear decision rules

### 9. MANDATORY Follow-up Questions
If the analysis in step 8 reveals ANY ambiguous answers, you MUST:
- Add specific follow-up questions to the plan document using [Answer]: tags
- DO NOT proceed to approval until all ambiguities are resolved

### 10. Generate High-Level Design Artifacts
- Execute the approved plan to generate design artifacts
- Create `aidlc-docs/inception/high-level-design/system-context.md` with:
  - External actors and external systems
  - System boundary description
  - Context diagram (Mermaid, validated per content-validation.md)
- Create `aidlc-docs/inception/high-level-design/architecture-overview.md` with:
  - Major component/service map
  - Responsibility of each major component/service (macro level, not method-level)
  - High-level architecture diagram (Mermaid, validated per content-validation.md)
- Create `aidlc-docs/inception/high-level-design/technology-direction.md` with:
  - Selected technology stack direction and rationale
  - Key constraints or standards driving the choice
- Create `aidlc-docs/inception/high-level-design/integration-boundaries.md` with:
  - Integration points between major components and with external systems
  - Macro-level data flow diagram
  - Communication style per integration (sync/async), and for async: choreography vs. orchestration
  - Distributed consistency approach for cross-service business operations (Saga pattern with orchestration/choreography, eventual consistency + compensation, or N/A for single-service systems) and rationale
  - Data ownership map: which service/component owns each key data entity
- Create `aidlc-docs/inception/high-level-design/architectural-style.md` with:
  - The chosen system-organization style (Layered/N-tier, Clean Architecture, Hexagonal, DDD, Microservices, or hybrid) and rationale
  - The dependency rule for the style (e.g., "outer layers depend on inner layers only" for Clean Architecture)
  - Whether Dependency Injection is used at the system level and the general mechanism (constructor injection, DI container/framework, service locator) — detailed per-unit DI wiring is defined later in Low-Level Design
- Create `aidlc-docs/inception/high-level-design/high-level-design.md` that consolidates the above docs into a single overview
- Per `common/architecture-decision-records.md`: create an ADR for each significant trade-off decision made in this stage (architectural style, distributed communication pattern, API Gateway decision, technology direction). Reference the ADR number(s) from the relevant artifact above

### 11. Log Approval
- Log approval prompt with timestamp in `aidlc-docs/audit.md`
- Include complete approval prompt text
- Use ISO 8601 timestamp format

### 12. Present Completion Message

```markdown
# 🏛️ High-Level Design Complete

[AI-generated summary of high-level design artifacts created in bullet points]

> **📋 <u>**REVIEW REQUIRED:**</u>**  
> Please examine the high-level design artifacts at: `aidlc-docs/inception/high-level-design/`

> **🚀 <u>**WHAT'S NEXT?**</u>**
>
> **You may:**
>
> 🔧 **Request Changes** - Ask for modifications to the high-level design if required
> ✅ **Approve & Continue** - Approve design and proceed to **Application Design**
```

### 13. Wait for Explicit Approval
- Do not proceed until the user explicitly approves the high-level design
- Approval must be clear and unambiguous
- If user requests changes, update the design and repeat the approval process

### 14. Record Approval Response
- Log the user's approval response with timestamp in `aidlc-docs/audit.md`
- Include the exact user response text
- Mark the approval status clearly

### 15. Update Progress
- Mark High-Level Design stage complete in `aidlc-docs/aidlc-state.md`
- Update the "Current Status" section
- Prepare for transition to Application Design
