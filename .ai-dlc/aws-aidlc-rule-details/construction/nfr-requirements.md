# NFR Requirements

## Prerequisites
- Functional Design must be complete for the unit
- Unit functional design artifacts must be available
- Execution plan must indicate NFR Requirements stage should execute

## Overview
Determine non-functional requirements for the unit and make tech stack choices.

## Steps to Execute

### Step 1: Analyze Functional Design
- Read functional design artifacts from `aidlc-docs/construction/{unit-name}/functional-design/`
- Understand business logic complexity and requirements

### Step 2: Create NFR Requirements Plan
- Generate plan with checkboxes [] for NFR assessment
- Focus on scalability, performance, availability, security
- Each step should have a checkbox []

### Step 3: Generate Context-Appropriate Questions
**DIRECTIVE**: Thoroughly analyze the functional design to identify ALL areas where NFR clarification would improve system quality and architecture decisions. Be proactive in asking questions to ensure comprehensive NFR coverage.

**CRITICAL**: Default to asking questions when there is ANY ambiguity or missing detail that could affect system quality. It's better to ask too many questions than to make incorrect NFR assumptions.

- EMBED questions using [Answer]: tag format
- Focus on ANY ambiguities, missing information, or areas needing clarification
- Generate questions wherever user input would improve NFR and tech stack decisions
- **When in doubt, ask the question** - overconfidence leads to poor system quality

**Question categories to evaluate** (consider ALL categories):
- **Scalability Requirements** - Ask about expected load, growth patterns, scaling triggers, and capacity planning
- **Performance Requirements** - Ask about response times, throughput, latency, and performance benchmarks
- **Availability Requirements** - Ask about uptime expectations, disaster recovery, failover, and business continuity
- **Security Requirements** - Ask about data protection, compliance, authentication, authorization, and threat models
- **Tech Stack Selection** - **MANDATORY, always ask explicitly with full trade-off options (per `common/question-format-guide.md`)**, covering:
  - **Consistency with system-wide direction**: If High-Level Design produced `aidlc-docs/inception/high-level-design/technology-direction.md`, present the system-level direction as the default option and ask the user to confirm or justify a deviation for this unit (polyglot should be a deliberate choice, not an accident)
  - **Language/runtime selection**: Present realistic language/runtime candidates for this unit's workload, each with what it is, strengths, and trade-offs
  - **Framework selection**: Present realistic framework candidates within the chosen language, covering:
    - Ecosystem/library maturity for this unit's needs (e.g., ORM support, async support, testing tooling)
    - Performance characteristics relevant to this unit's NFRs (throughput, latency, memory footprint)
    - Team familiarity / existing skillset — ask directly rather than assuming; an unfamiliar framework has a real ramp-up cost even if technically superior
    - Long-term maintenance: community size/activity, release cadence, LTS support, hiring pool
    - Licensing and cost implications (especially for commercial/enterprise frameworks or SaaS-dependent libraries)
  - Never default to "whatever HLD said" or "whatever's popular" without presenting the trade-offs and getting explicit confirmation
- **Reliability Requirements** - Ask about error handling, fault tolerance, monitoring, and alerting needs
- **Maintainability Requirements** - Ask about code quality, documentation, testing, and operational requirements
- **Usability Requirements** - Ask about user experience, accessibility, and interface requirements
- **Caching Requirements** - Ask whether this unit needs caching (read-heavy endpoints, expensive computations, external API calls), what should be cached, cache invalidation triggers, and acceptable staleness
- **Messaging & Event Participation** - If High-Level Design selected event-driven/async communication (`aidlc-docs/inception/high-level-design/integration-boundaries.md`), ask whether this unit publishes events, consumes events, or both, and what delivery guarantee is required (at-least-once, exactly-once, at-most-once)
- **Distributed Transaction Participation** - If this unit takes part in a cross-service business operation, ask whether it participates in a Saga as an orchestrator, a participant/choreography step, or is out of scope; ask about compensating action requirements for this unit's steps

### Step 4: Store Plan
- Save as `aidlc-docs/construction/plans/{unit-name}-nfr-requirements-plan.md`
- Include all [Answer]: tags for user input

### Step 5: Collect and Analyze Answers
- Wait for user to complete all [Answer]: tags
- **MANDATORY**: Carefully review ALL responses for vague or ambiguous answers
- **CRITICAL**: Add follow-up questions for ANY unclear responses - do not proceed with ambiguity
- Look for responses like "depends", "maybe", "not sure", "mix of", "somewhere between", "standard", "typical"
- Create clarification questions file if ANY ambiguities are detected
- **Do not proceed until ALL ambiguities are resolved**

### Step 6: Generate NFR Requirements Artifacts
- Create `aidlc-docs/construction/{unit-name}/nfr-requirements/nfr-requirements.md`
- Create `aidlc-docs/construction/{unit-name}/nfr-requirements/tech-stack-decisions.md` with the chosen language, runtime, and framework(s), plus the rationale (ecosystem, performance, team familiarity, maintenance, licensing) from Step 3
- Per `common/architecture-decision-records.md`: create an ADR for the language/framework selection (this is a costly-to-reverse decision by definition) and for any other significant NFR trade-off decided in this stage. Reference the ADR number(s) from `tech-stack-decisions.md`

### Step 7: Present Completion Message
- Present completion message in this structure:
     1. **Completion Announcement** (mandatory): Always start with this:

```markdown
# 📊 NFR Requirements Complete - [unit-name]
```

     2. **AI Summary** (optional): Provide structured bullet-point summary of NFR requirements
        - Format: "NFR requirements assessment has identified [description]:"
        - List key scalability, performance, availability requirements (bullet points)
        - List security and compliance requirements identified
        - Mention tech stack decisions and rationale
        - DO NOT include workflow instructions ("please review", "let me know", "proceed to next phase", "before we proceed")
        - Keep factual and content-focused
     3. **Formatted Workflow Message** (mandatory): Always end with this exact format:

```markdown
> **📋 <u>**REVIEW REQUIRED:**</u>**  
> Please examine the NFR requirements at: `aidlc-docs/construction/[unit-name]/nfr-requirements/`



> **🚀 <u>**WHAT'S NEXT?**</u>**
>
> **You may:**
>
> 🔧 **Request Changes** - Ask for modifications to the NFR requirements based on your review  
> ✅ **Continue to Next Stage** - Approve NFR requirements and proceed to **[next-stage-name]**

---
```

### Step 8: Wait for Explicit Approval
- Do not proceed until the user explicitly approves the NFR requirements
- Approval must be clear and unambiguous
- If user requests changes, update the requirements and repeat the approval process

### Step 9: Record Approval and Update Progress
- Log approval in audit.md with timestamp
- Record the user's approval response with timestamp
- Mark NFR Requirements stage complete in aidlc-state.md
