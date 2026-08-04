# Question Format Guide

## MANDATORY: All Questions Must Use This Format

### Rule: Never Ask Questions in Chat
**CRITICAL**: You must NEVER ask questions directly in the chat. ALL questions must be placed in dedicated question files.

### Question File Format

#### File Naming Convention
- Use descriptive names: `{phase-name}-questions.md`
- Examples:
  - `classification-questions.md`
  - `requirements-questions.md`
  - `story-planning-questions.md`
  - `design-questions.md`

#### Question Structure
Every question must include meaningful options plus "Other" as the last option:

```markdown
## Question [Number]
[Clear, specific question text]

A) [Option name] — [1-2 sentence: what it is, in plain terms]
   - ✅ Strengths: [key advantages]
   - ⚠️ Trade-offs: [key downsides/costs/risks]

B) [Option name] — [1-2 sentence: what it is, in plain terms]
   - ✅ Strengths: [key advantages]
   - ⚠️ Trade-offs: [key downsides/costs/risks]

[...additional options as needed...]

X) Other (please describe after [Answer]: tag below)

[Answer]: 
```

**MANDATORY — Option Explanations & Trade-offs**: For any question offering a technology, pattern, or mechanism choice (not simple factual/scope questions like "greenfield or brownfield"), every option MUST include:
1. **What it is** — a short, plain-language explanation. Never assume the user already knows the term; a user who did know it loses nothing by having it restated briefly, but a user who didn't know it is unblocked.
2. **Strengths** — why you'd pick it.
3. **Trade-offs** — cost, complexity, operational burden, or scenarios where it's the wrong choice.

This lets the user make an informed trade-off decision instead of picking a label blindly.

**MANDATORY — Proactively Surface Options the User May Not Know About**: Don't limit options to what the user already mentioned or to the most textbook-obvious choices. Before finalizing a question's option list, actively consider whether there is a relevant technology, pattern, or mechanism the user hasn't brought up but that fits the context (e.g., if the user is deciding on caching and their read pattern is highly skewed toward a few hot keys, surface a note about hot-key mitigation, not just "add a cache"). When you do this:
- Add it as its own lettered option with the same What-it-is / Strengths / Trade-offs structure
- Make clear it's a suggestion the user may not have considered, e.g. prefix with "💡 Suggested:" in the option name
- Don't overload the question — 1-2 well-chosen proactive suggestions beat a long list of unlikely options
- If genuinely nothing beyond the obvious choices applies, don't force a suggestion just to have one

**CRITICAL**:
- "Other" is MANDATORY as the LAST option for every question
- Only include meaningful options - don't make up options to fill slots
- Use as many or as few options as make sense (minimum 2 + Other)
- **Each option must be separated by a blank line** so strict CommonMark renderers (IntelliJ, PyCharm, etc.) display them on separate lines instead of collapsing them into a single paragraph

### Complete Example

```markdown
# Requirements Clarification Questions

Please answer the following questions to help clarify the requirements.

## Question 1
What is the primary user authentication method?

A) Username and password

B) Social media login (Google, Facebook)

C) Single Sign-On (SSO)

D) Multi-factor authentication

E) Other (please describe after [Answer]: tag below)

[Answer]: 

## Question 2
Will this be a web or mobile application?

A) Web application

B) Mobile application

C) Both web and mobile

D) Other (please describe after [Answer]: tag below)

[Answer]: 

## Question 3
Is this a new project or existing codebase?

A) New project (greenfield)

B) Existing codebase (brownfield)

C) Other (please describe after [Answer]: tag below)

[Answer]: 
```

### User Response Format
Users will answer by filling in the letter choice after [Answer]: tag:

```markdown
## Question 1
What is the primary user authentication method?

A) Username and password

B) Social media login (Google, Facebook)

C) Single Sign-On (SSO)

D) Multi-factor authentication

[Answer]: C
```

### Reading User Responses
After user confirms completion:
1. Read the question file
2. Extract answers after [Answer]: tags
3. Validate all questions are answered
4. Proceed with analysis based on responses

### Multiple Choice Guidelines

#### Option Count
- Minimum: 2 meaningful options + "Other" (A, B, C)
- Typical: 3-4 meaningful options + "Other" (A, B, C, D, E)
- Maximum: 5 meaningful options + "Other" (A, B, C, D, E, F)
- **CRITICAL**: Don't make up options just to fill slots - only include meaningful choices

#### Option Quality
- Make options mutually exclusive
- Cover the most common scenarios
- Only include meaningful, realistic options
- **ALWAYS include "Other" as the LAST option** (MANDATORY)
- Be specific and clear
- **Don't make up options to fill A, B, C, D slots**

#### Good Example:
```markdown
## Question 5
What database technology will be used?

A) Relational (PostgreSQL, MySQL) — structured tables with fixed schema and strong consistency guarantees (ACID transactions)
   - ✅ Strengths: mature tooling, strong consistency, powerful joins/queries, well-understood operationally
   - ⚠️ Trade-offs: schema changes require migrations; horizontal write-scaling is harder than NoSQL

B) NoSQL Document (MongoDB, DynamoDB) — schema-flexible documents (JSON-like), typically eventually consistent by default
   - ✅ Strengths: flexible schema evolution, easy horizontal scaling, good fit for nested/variable data
   - ⚠️ Trade-offs: weaker consistency defaults, joins across documents are awkward/expensive

C) NoSQL Key-Value (Redis, Memcached) — simple key → value lookups, usually in-memory
   - ✅ Strengths: very low latency, simple mental model, great for caching/session storage
   - ⚠️ Trade-offs: no query flexibility beyond the key; not meant as a system of record for complex data

D) Graph Database (Neo4j, Neptune) — nodes and edges optimized for traversing relationships
   - ✅ Strengths: excellent for relationship-heavy queries (recommendations, fraud detection, social graphs)
   - ⚠️ Trade-offs: niche use case, added operational complexity if the rest of the system doesn't need graph queries

E) Other (please describe after [Answer]: tag below)

[Answer]: 
```

#### Bad Example (Avoid):
```markdown
## Question 5
What database will you use?

A) Yes

B) No

C) Maybe

[Answer]: 
```

### Workflow Integration

#### Step 1: Create Question File
```markdown
Create aidlc-docs/{phase-name}-questions.md with all questions
```

#### Step 2: Inform User
```
"I've created {phase-name}-questions.md with [X] questions. 
Please answer each question by filling in the letter choice after the [Answer]: tag. 
If none of the options match your needs, choose the last option (Other) and describe your preference. Let me know when you're done."
```

#### Step 3: Wait for Confirmation
Wait for user to say "done", "completed", "finished", or similar.

#### Step 4: Read and Analyze
```
Read aidlc-docs/{phase-name}-questions.md
Extract all answers
Validate completeness
Proceed with analysis
```

### Error Handling

#### Missing Answers
If any [Answer]: tag is empty:
```
"I noticed Question [X] is not answered. Please provide an answer using one of the letter choices 
for all questions before proceeding."
```

#### Invalid Answers
If answer is not a valid letter choice:
```
"Question [X] has an invalid answer '[answer]'. 
Please use only the letter choices provided in the question."
```

#### Ambiguous Answers
If user provides explanation instead of letter:
```
"For Question [X], please provide the letter choice that best matches your answer. 
If none match, choose 'Other' and add your description after the [Answer]: tag."
```

### Contradiction and Ambiguity Detection

**MANDATORY**: After reading user responses, you MUST check for contradictions and ambiguities.

#### Detecting Contradictions
Look for logically inconsistent answers:
- Scope mismatch: "Bug fix" but "Entire codebase affected"
- Risk mismatch: "Low risk" but "Breaking changes"
- Timeline mismatch: "Quick fix" but "Multiple subsystems"
- Impact mismatch: "Single component" but "Significant architecture changes"

#### Detecting Ambiguities
Look for unclear or borderline responses:
- Answers that could fit multiple classifications
- Responses that lack specificity
- Conflicting indicators across multiple questions

#### Creating Clarification Questions
If contradictions or ambiguities detected:

1. **Create clarification file**: `{phase-name}-clarification-questions.md`
2. **Explain the issue**: Clearly state what contradiction/ambiguity was detected
3. **Ask targeted questions**: Use multiple choice format to resolve the issue
4. **Reference original questions**: Show which questions had conflicting answers

**Example**:
```markdown
# [Phase Name] Clarification Questions

I detected contradictions in your responses that need clarification:

## Contradiction 1: [Brief Description]
You indicated "[Answer A]" (Q[X]:[Letter]) but also "[Answer B]" (Q[Y]:[Letter]).
These responses are contradictory because [explanation].

### Clarification Question 1
[Specific question to resolve contradiction]

A) [Option that resolves toward first answer]

B) [Option that resolves toward second answer]

C) [Option that provides middle ground]

D) [Option that reframes the question]

[Answer]: 

## Ambiguity 1: [Brief Description]
Your response to Q[X] ("[Answer]") is ambiguous because [explanation].

### Clarification Question 2
[Specific question to clarify ambiguity]

A) [Clear option 1]

B) [Clear option 2]

C) [Clear option 3]

D) [Clear option 4]

[Answer]: 
```

#### Workflow for Clarifications

1. **Detect**: Analyze all responses for contradictions/ambiguities
2. **Create**: Generate clarification question file if issues found
3. **Inform**: Tell user about the issues and clarification file
4. **Wait**: Do not proceed until user provides clarifications
5. **Re-validate**: After clarifications, check again for consistency
6. **Proceed**: Only move forward when all contradictions are resolved

#### Example User Message
```
"I detected 2 contradictions in your responses:

1. Bug fix scope vs. codebase impact (Q1 vs Q2)
2. Low risk vs. breaking changes (Q7 vs Q4)

I've created classification-clarification-questions.md with 2 questions to resolve these.
Please answer these clarifying questions before I can proceed with classification."
```

### Best Practices

1. **Be Specific**: Questions should be clear and unambiguous
2. **Be Comprehensive**: Cover all necessary information
3. **Be Concise**: Keep questions focused on one topic
4. **Be Practical**: Options should be realistic and actionable
5. **Be Consistent**: Use same format throughout all question files

### Phase-Specific Examples

#### Example with 2 meaningful options:
```markdown
## Question 1
Is this a new project or existing codebase?

A) New project (greenfield)

B) Existing codebase (brownfield)

C) Other (please describe after [Answer]: tag below)

[Answer]: 
```

#### Example with 3 meaningful options:
```markdown
## Question 2
What is the deployment target?

A) Cloud (AWS, Azure, GCP)

B) On-premises servers

C) Hybrid (both cloud and on-premises)

D) Other (please describe after [Answer]: tag below)

[Answer]: 
```

#### Example with 4 meaningful options:
```markdown
## Question 3
What architectural pattern should be used?

A) Monolithic architecture

B) Microservices architecture

C) Serverless architecture

D) Event-driven architecture

E) Other (please describe after [Answer]: tag below)

[Answer]: 
```

## Summary

**Remember**: 
- ✅ Always create question files
- ✅ Always use multiple choice format
- ✅ **Always include "Other" as the LAST option (MANDATORY)**
- ✅ Only include meaningful options - don't make up options to fill slots
- ✅ For technology/pattern/mechanism choices: explain what each option IS, plus its strengths and trade-offs
- ✅ Proactively surface relevant options the user hasn't mentioned, marked "💡 Suggested:"
- ✅ Always use [Answer]: tags
- ✅ Always wait for user completion
- ✅ Always validate responses for contradictions
- ✅ Always create clarification files if needed
- ✅ Always resolve contradictions before proceeding
- ❌ Never ask questions in chat
- ❌ Never make up options just to have A, B, C, D
- ❌ Never proceed without answers
- ❌ Never proceed with unresolved contradictions
- ❌ Never make assumptions about ambiguous responses
