# Infrastructure Design

## Prerequisites
- Functional Design must be complete for the unit
- NFR Design recommended (provides logical components to map)
- Execution plan must indicate Infrastructure Design stage should execute

## Overview
Map logical software components to actual infrastructure choices for deployment environments.

## Steps to Execute

### Step 1: Analyze Design Artifacts
- Read functional design from `aidlc-docs/construction/{unit-name}/functional-design/`
- Read NFR design from `aidlc-docs/construction/{unit-name}/nfr-design/` (if exists)
- Identify logical components needing infrastructure

### Step 2: Create Infrastructure Design Plan
- Generate plan with checkboxes [] for infrastructure design
- Focus on mapping to actual services (AWS, Azure, GCP, on-premise)
- Each step should have a checkbox []

### Step 3: Generate Context-Appropriate Questions
**DIRECTIVE**: Thoroughly analyze the functional and NFR design to identify ALL areas where clarification would improve infrastructure decisions. Be proactive in asking questions to ensure comprehensive infrastructure coverage.

**CRITICAL**: Default to asking questions when there is ANY ambiguity or missing detail that could affect infrastructure quality. It's better to ask too many questions than to make incorrect infrastructure assumptions.

**MANDATORY**: Evaluate ALL of the following categories by asking targeted questions about each. For each category, determine applicability based on evidence from the functional and NFR design artifacts -- do not skip categories without explicit justification:

- EMBED questions using [Answer]: tag format
- Focus on ANY ambiguities, missing information, or areas needing clarification
- Generate questions wherever user input would improve infrastructure decisions
- **When in doubt, ask the question** - overconfidence leads to poor infrastructure choices

**Question categories to evaluate** (consider ALL categories):
- **Deployment Environment** - Ask about cloud provider preferences, environment setup, and deployment targets
- **Compute Infrastructure** - Ask about compute service choices, sizing, and scaling requirements (e.g., horizontal auto-scaling triggers/thresholds, min/max instance counts)
- **Storage Infrastructure** - Ask about database selection, storage patterns, and data lifecycle needs
- **Database Read/Write Splitting** - **MANDATORY if this unit has non-trivial read load, always ask explicitly**: Ask whether the database uses a single primary for both reads and writes, or a primary-replica setup with read replicas for read traffic. If replicas: ask about replication lag tolerance, which queries are routed to replicas vs. primary (must-be-fresh reads go to primary), and how the app/ORM routes read vs. write queries. If NFR Design selected CQRS for this unit (`aidlc-docs/construction/{unit-name}/nfr-design/`), map the read/write model split to concrete infrastructure here (e.g., write store = Postgres primary, read store = Elasticsearch/Redis/read replica)
- **Database Sharding/Partitioning** - **MANDATORY if NFR Requirements' scalability targets exceed what a single primary + read replicas can sustain for writes (very high write throughput, dataset larger than a single instance can hold)**: Ask whether the database should be sharded/partitioned, and if so:
  - Sharding key selection — what it is: the value used to route a row to a specific shard (e.g., tenant ID, user ID, geographic region, hash of a natural key)
    - ✅ Strengths of a well-chosen key: even data distribution, most queries hit a single shard
    - ⚠️ Trade-offs: a poorly chosen key causes hot shards; cross-shard queries/joins become expensive or impossible without a fan-out/aggregation layer
  - Sharding approach: range-based (ordered ranges of the key), hash-based (hash of key mod N), or directory-based (lookup table mapping key → shard) — each with its own rebalancing cost when adding shards
  - Native DB partitioning (e.g., Postgres declarative partitioning, MySQL partitioning) vs. application-level sharding (routing logic in the app/ORM) vs. a managed sharded service (e.g., Vitess, DynamoDB/Cosmos DB's built-in partitioning, Citus)
  - Cross-shard query strategy: how (or whether) queries that span multiple shards are supported, and the performance/consistency cost of doing so
  - Resharding/rebalancing plan: how shard count grows as data grows without a full-system outage
  - If NFR Requirements' targets do NOT require sharding, explicitly record "sharding not required at current scale" with the read/write numbers that justified that decision, rather than silently omitting the question
- **Messaging Infrastructure** - Ask about messaging/queuing services, event-driven patterns, and async processing
- **Networking Infrastructure** - Ask about network topology, VPC/subnet design, and TLS termination
- **Load Balancer** - **MANDATORY if this unit has more than one running instance, always ask explicitly**: Ask about load balancer type (L4/L7, cloud-native e.g. ALB/NLB/Azure LB/GCP LB, or self-managed e.g. nginx/HAProxy), routing algorithm (round-robin, least-connections, IP hash), health check configuration, and session affinity/stickiness needs
- **API Gateway** - Confirm and implement the API Gateway decision made in High-Level Design (`aidlc-docs/inception/high-level-design/integration-boundaries.md`, if present). If HLD decided a gateway fronts services: ask about the specific product (AWS API Gateway, Kong, Apigee, nginx, Azure API Management, etc.), routing rules, auth/rate-limiting enforcement point, and how it relates to the load balancer (gateway in front of LB, or LB in front of gateway instances). If no HLD decision exists (e.g., single-unit project), ask directly
- **Monitoring Infrastructure** - Ask about observability tooling, alerting strategy, and logging requirements
- **Shared Infrastructure** - Ask about infrastructure sharing strategy, multi-tenancy, and resource isolation

### Step 4: Store Plan
- Save as `aidlc-docs/construction/plans/{unit-name}-infrastructure-design-plan.md`
- Include all [Answer]: tags for user input

### Step 5: Collect and Analyze Answers
- Wait for user to complete all [Answer]: tags
- Review for vague or ambiguous responses
- Add follow-up questions if needed

### Step 6: Generate Infrastructure Design Artifacts
- Create `aidlc-docs/construction/{unit-name}/infrastructure-design/infrastructure-design.md`
- Create `aidlc-docs/construction/{unit-name}/infrastructure-design/deployment-architecture.md` including:
  - Database topology: single primary, primary + read replicas (with replica count and read/write routing rules), and/or sharding strategy (key, approach, cross-shard query handling, resharding plan) if applicable
  - Load balancer configuration: type, algorithm, health checks (if unit has multiple instances)
  - API Gateway configuration or explicit "no gateway" note (consistent with the HLD decision, if HLD ran)
  - Scaling configuration: auto-scaling triggers, min/max instances
- If shared infrastructure: Create `aidlc-docs/construction/shared-infrastructure.md` (API Gateway and cross-unit load balancing, if shared across units, belong here rather than duplicated per-unit)
- Per `common/architecture-decision-records.md`: create an ADR for each significant infrastructure decision made in this stage (database read/write splitting, load balancer choice, API Gateway product, cloud provider/service selection). Reference the ADR number(s) from `deployment-architecture.md`

### Step 7: Present Completion Message
- Present completion message in this structure:
     1. **Completion Announcement** (mandatory): Always start with this:

```markdown
# 🏢 Infrastructure Design Complete - [unit-name]
```

     2. **AI Summary** (optional): Provide structured bullet-point summary of infrastructure design
        - Format: "Infrastructure design has mapped [description]:"
        - List key infrastructure services and components (bullet points)
        - List deployment architecture decisions and rationale
        - Mention cloud provider choices and service mappings
        - DO NOT include workflow instructions ("please review", "let me know", "proceed to next phase", "before we proceed")
        - Keep factual and content-focused
     3. **Formatted Workflow Message** (mandatory): Always end with this exact format:

```markdown
> **📋 <u>**REVIEW REQUIRED:**</u>**  
> Please examine the infrastructure design at: `aidlc-docs/construction/[unit-name]/infrastructure-design/`



> **🚀 <u>**WHAT'S NEXT?**</u>**
>
> **You may:**
>
> 🔧 **Request Changes** - Ask for modifications to the infrastructure design based on your review  
> ✅ **Continue to Next Stage** - Approve infrastructure design and proceed to **Code Generation**

---
```

### Step 8: Wait for Explicit Approval
- Do not proceed until the user explicitly approves the infrastructure design
- Approval must be clear and unambiguous
- If user requests changes, update the design and repeat the approval process

### Step 9: Record Approval and Update Progress
- Log approval in audit.md with timestamp
- Record the user's approval response with timestamp
- Mark Infrastructure Design stage complete in aidlc-state.md
