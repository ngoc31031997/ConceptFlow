# NFR Design

## Prerequisites
- NFR Requirements must be complete for the unit
- NFR requirements artifacts must be available
- Execution plan must indicate NFR Design stage should execute

## Overview
Incorporate NFR requirements into unit design using patterns and logical components.

## Steps to Execute

### Step 1: Analyze NFR Requirements
- Read NFR requirements from `aidlc-docs/construction/{unit-name}/nfr-requirements/`
- Understand scalability, performance, availability, security needs

### Step 2: Create NFR Design Plan
- Generate plan with checkboxes [] for NFR design
- Focus on design patterns and logical components
- Each step should have a checkbox []

### Step 3: Generate Context-Appropriate Questions
**DIRECTIVE**: Thoroughly analyze the NFR requirements to identify ALL areas where clarification would improve NFR design quality. Be proactive in asking questions to ensure comprehensive non-functional design coverage.

**CRITICAL**: Default to asking questions when there is ANY ambiguity or missing detail that could affect NFR design quality. It's better to ask too many questions than to make incorrect assumptions about non-functional patterns.

**MANDATORY**: Evaluate ALL of the following categories by asking targeted questions about each. For each category, determine applicability based on evidence from the NFR requirements -- do not skip categories without explicit justification:

- EMBED questions using [Answer]: tag format
- Focus on ANY ambiguities, missing information, or areas needing clarification
- Generate questions wherever user input would improve pattern and component decisions
- **When in doubt, ask the question** - overconfidence leads to poor non-functional designs

**Question categories to evaluate** (consider ALL categories):
- **Resilience Patterns** - Ask about fault tolerance approach, retry strategies, and failure recovery expectations
- **Scalability Patterns** - Ask about scaling mechanisms, load boundaries, and growth projections
- **Performance Patterns** - Ask about optimization strategy, latency targets, and throughput requirements
- **Security Patterns** - Ask about security implementation approach, threat model, and compliance constraints
- **Logical Components** - Ask about infrastructure components (queues, caches, circuit breakers, etc.) and their integration patterns
- **Caching Strategy** - If NFR Requirements identified a caching need: ask about cache placement (in-process, distributed cache e.g. Redis/Memcached, CDN, HTTP caching headers), cache-aside vs. write-through vs. write-behind, key design, TTL/invalidation strategy, and cache stampede protection
- **Event-Driven Design** - If this unit publishes/consumes events: ask about the event schema/versioning approach, event broker/topic design, and whether events are domain events (business-meaningful) vs. integration events
- **Saga Pattern** - If this unit participates in a distributed transaction (per High-Level Design and NFR Requirements): ask which saga style applies (orchestration with a central coordinator, or choreography via events) and how compensating transactions are triggered and implemented for this unit's step
- **Inbox/Outbox Pattern** - If this unit publishes events as part of a database transaction, or must process incoming events exactly once: ask whether the Transactional Outbox pattern (write event to an outbox table in the same DB transaction as the business change, relay published by a separate poller/CDC process) and/or Inbox pattern (dedupe incoming messages via a processed-message-id table) should be applied, and confirm the relay mechanism (polling publisher vs. CDC/Debezium-style)
- **Idempotency** - Ask how this unit guarantees idempotent handling of retried requests/messages (idempotency keys, dedupe tables, natural idempotent operations)
- **Data Access Pattern (CRUD vs. CQRS)** - **MANDATORY, always ask explicitly, never default silently**: Ask whether this unit uses plain CRUD (single model for reads and writes) or CQRS (separate command/write model and query/read model). If CQRS:
  - How the read model is kept in sync with the write model: synchronous (same transaction/DB view) vs. asynchronous (event-driven projection, eventually consistent)
  - Whether the read model uses a different data store optimized for queries (e.g., write to Postgres, project to Elasticsearch/Redis/read-replica) or the same store with different schemas/views
  - Acceptable staleness window for the read model if asynchronous
  - Only recommend CQRS when justified by evidence (high read/write ratio skew, complex reporting/query needs, independent read scaling needs) — do not default to CQRS for simple CRUD units

### Step 4: Store Plan
- Save as `aidlc-docs/construction/plans/{unit-name}-nfr-design-plan.md`
- Include all [Answer]: tags for user input

### Step 5: Collect and Analyze Answers
- Wait for user to complete all [Answer]: tags
- Review for vague or ambiguous responses
- Add follow-up questions if needed

### Step 6: Generate NFR Design Artifacts
- Create `aidlc-docs/construction/{unit-name}/nfr-design/nfr-design-patterns.md`
- Create `aidlc-docs/construction/{unit-name}/nfr-design/logical-components.md`
- If caching applies: include cache placement, strategy (cache-aside/write-through/write-behind), key design, TTL/invalidation in `logical-components.md`
- If event-driven/messaging applies: create `aidlc-docs/construction/{unit-name}/nfr-design/messaging-design.md` with event schema/topic design, delivery guarantee, and Saga role (orchestrator/participant/N/A) with compensating actions
- If Inbox/Outbox pattern applies: include table schema (outbox: event id, aggregate id, payload, created_at, published_at; inbox: message id, processed_at) and relay mechanism in `messaging-design.md`
- Always record the CRUD vs. CQRS decision (with rationale) in `nfr-design-patterns.md`; if CQRS, detail the read/write model split and sync mechanism in `logical-components.md`
- Per `common/architecture-decision-records.md`: create an ADR for each significant pattern decision made in this stage (CRUD vs. CQRS, caching strategy, event-driven design, Saga style, Inbox/Outbox adoption). Reference the ADR number(s) from the relevant artifact above

### Step 7: Present Completion Message
- Present completion message in this structure:
     1. **Completion Announcement** (mandatory): Always start with this:

```markdown
# 🎨 NFR Design Complete - [unit-name]
```

     2. **AI Summary** (optional): Provide structured bullet-point summary of NFR design
        - Format: "NFR design has incorporated [description]:"
        - List key design patterns implemented (bullet points)
        - List logical components and infrastructure elements
        - Mention resilience, scalability, and performance patterns applied
        - DO NOT include workflow instructions ("please review", "let me know", "proceed to next phase", "before we proceed")
        - Keep factual and content-focused
     3. **Formatted Workflow Message** (mandatory): Always end with this exact format:

```markdown
> **📋 <u>**REVIEW REQUIRED:**</u>**  
> Please examine the NFR design at: `aidlc-docs/construction/[unit-name]/nfr-design/`



> **🚀 <u>**WHAT'S NEXT?**</u>**
>
> **You may:**
>
> 🔧 **Request Changes** - Ask for modifications to the NFR design based on your review  
> ✅ **Continue to Next Stage** - Approve NFR design and proceed to **[next-stage-name]**

---
```

### Step 8: Wait for Explicit Approval
- Do not proceed until the user explicitly approves the NFR design
- Approval must be clear and unambiguous
- If user requests changes, update the design and repeat the approval process

### Step 9: Record Approval and Update Progress
- Log approval in audit.md with timestamp
- Record the user's approval response with timestamp
- Mark NFR Design stage complete in aidlc-state.md
