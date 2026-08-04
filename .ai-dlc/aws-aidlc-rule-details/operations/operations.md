# Operations - Detailed Steps

## Purpose
**Deployment strategy, observability, resilience, and production-readiness for the built system**

Operations focuses on:
- How the system gets deployed and rolled back safely
- How autoscaling is configured and verified against the scalability requirements from NFR Requirements
- How the system is observed in production (metrics, logs, traces)
- How the team responds when things go wrong
- Whether the system can survive the failure scenarios its NFRs claimed it could (disaster recovery)

**Note**: This stage does NOT re-decide architecture (that's HLD/NFR Design) or infrastructure component choices (that's Infrastructure Design). It operationalizes what was already designed.

## Prerequisites
- Build and Test must be complete
- Infrastructure Design recommended (provides the infrastructure components to operationalize)
- Execution plan must indicate Operations stage should execute

## Execute IF
- System is intended for a real deployment target (not a local-only prototype/spike)
- Multiple environments (staging/production) or any production deployment is planned
- NFR Requirements identified availability, scalability, or disaster-recovery expectations

## Skip IF
- Purely local/prototype work with no deployment target
- User explicitly defers Operations to a separate, later effort

## Step-by-Step Execution

### Step 1: Analyze Prior Artifacts
- Read `aidlc-docs/construction/{unit-name}/nfr-requirements/nfr-requirements.md` for availability/scalability targets per unit
- Read `aidlc-docs/construction/{unit-name}/infrastructure-design/deployment-architecture.md` for infrastructure choices
- Read `aidlc-docs/construction/build-and-test/ci-cd-integration-instructions.md` for the existing pipeline

### Step 2: Create Operations Plan
- Generate plan with checkboxes [] for operations setup
- Each step should have a checkbox []

### Step 3: Generate Context-Appropriate Questions
**DIRECTIVE**: Thoroughly analyze the NFR and infrastructure artifacts to identify ALL areas where clarification would improve production readiness. Follow `common/question-format-guide.md` — every technology/mechanism option must explain what it is, its strengths, and its trade-offs, and proactively surface options the user may not have considered.

**Question categories to evaluate** (consider ALL categories; mark N/A with justification if a category doesn't apply):
- **Deployment Strategy** - Ask about rollout mechanism:
  - Rolling deployment — replace instances gradually; simple, but a bad version is live for some users during rollout
  - Blue-Green — full parallel environment swap; instant rollback, but doubles infra cost during cutover
  - Canary — route a small % of traffic to the new version first; catches issues with minimal blast radius, but needs traffic-splitting infra and longer rollout time
  - 💡 Suggested (if applicable): Feature flags for decoupling deploy from release, if the unit has risky user-facing changes
- **Rollback Plan** - Ask about rollback trigger criteria (error rate threshold, manual trigger) and rollback mechanism (redeploy previous artifact, traffic shift back, DB migration reversibility)
- **Autoscaling Verification** - Ask to confirm the autoscaling triggers/thresholds set in Infrastructure Design actually match the scalability targets from NFR Requirements (e.g., "NFR said 10x traffic spikes" — does the autoscaling policy's max instance count and scale-up speed actually handle that?)
- **Observability - Metrics** - Ask which metrics platform (Prometheus/Grafana, CloudWatch, Datadog, etc.) and what golden signals are tracked (latency, traffic, errors, saturation)
- **Observability - Distributed Tracing** - **MANDATORY if more than one service/unit calls another**: Ask whether distributed tracing is implemented (e.g., OpenTelemetry with a backend like Jaeger/Tempo/X-Ray), and how correlation/trace IDs propagate across service boundaries (HTTP headers, message metadata for async calls). Without this, debugging a slow/failing request that crosses services is effectively guesswork.
- **Observability - Logging** - Ask about structured logging format, log aggregation platform, and whether trace/correlation IDs are included in every log line for cross-service correlation
- **SLO/Error Budget** - Ask whether Service Level Objectives are defined (e.g., 99.9% availability, p99 latency < 500ms) and how error budget burn triggers action
- **Incident Response** - Ask about on-call rotation existence, alerting routing (PagerDuty/Opsgenie/etc.), and runbook expectations for common failure modes
- **Disaster Recovery** - Ask about RPO (Recovery Point Objective) and RTO (Recovery Time Objective) targets, backup strategy and frequency, and whether multi-region/multi-AZ failover is required — cross-check against the availability NFR to catch mismatches (e.g., NFR said "99.99% availability" but no multi-AZ plan exists)
- **Capacity Planning** - Ask about expected growth rate and when infrastructure (DB shard count, instance limits, quota) needs review

### Step 4: Store Plan
- Save as `aidlc-docs/operations/plans/operations-plan.md`
- Include all [Answer]: tags for user input

### Step 5: Collect and Analyze Answers
- Wait for user to complete all [Answer]: tags
- Review for vague/ambiguous responses ("depends", "standard", "typical") and add follow-up questions before proceeding
- **Cross-check against NFRs**: flag any mismatch between what NFR Requirements promised (availability %, scale targets) and what the operations answers actually configure — do not silently let these diverge

### Step 6: Generate Operations Artifacts
- Create `aidlc-docs/operations/deployment-strategy.md` with rollout mechanism, rollback plan and triggers
- Create `aidlc-docs/operations/autoscaling-verification.md` cross-checking configured autoscaling against NFR scalability targets
- Create `aidlc-docs/operations/observability.md` with metrics platform, distributed tracing setup and correlation ID propagation approach, logging strategy, SLOs/error budgets
- Create `aidlc-docs/operations/incident-response.md` with on-call/alerting setup and runbook links/stubs for known failure modes (e.g., "database primary unreachable", "downstream service timeout")
- Create `aidlc-docs/operations/disaster-recovery.md` with RPO/RTO targets, backup strategy, failover plan
- Create `aidlc-docs/operations/capacity-plan.md` with growth assumptions and infra review triggers
- Per `common/architecture-decision-records.md`: create an ADR for the deployment strategy and for the tracing/observability platform choice (both are costly-to-reverse operational decisions)

### Step 7: Present Completion Message

```markdown
# 🚀 Operations Setup Complete

[AI-generated summary of operations artifacts created in bullet points]

> **📋 <u>**REVIEW REQUIRED:**</u>**  
> Please examine the operations artifacts at: `aidlc-docs/operations/`

> **🚀 <u>**WHAT'S NEXT?**</u>**
>
> **You may:**
>
> 🔧 **Request Changes** - Ask for modifications to the operations setup if required
> ✅ **Approve & Complete** - Approve operations setup and mark the workflow **Complete**
```

### Step 8: Wait for Explicit Approval
- Do not proceed until the user explicitly approves the operations setup

### Step 9: Record Approval and Update Progress
- Log approval in audit.md with timestamp
- Mark Operations stage complete in `aidlc-docs/aidlc-state.md`

## Completion Criteria
- Deployment strategy and rollback plan defined
- Autoscaling configuration verified against NFR scalability targets (mismatches resolved, not just noted)
- Observability (metrics, tracing, logging, SLOs) defined
- Incident response and disaster recovery plans defined with concrete RPO/RTO
- Capacity plan defined
- Relevant ADRs created
