# CI/CD Integration Instructions

## Pipeline Stages Overview
[Checkout] → [Install Dependencies] → [Build] → [Unit Tests] → [SonarQube Analysis] → [OWASP Dependency Check] → [Integration/E2E Tests] → [Package/Artifact] → [Deploy]

## 1. CI/CD Platform
- **Platform**: GitHub Actions (targeted by default). No existing `.github/workflows/`, `.gitlab-ci.yml`, `Jenkinsfile`, or `azure-pipelines.yml` was found in the repo (brownfield check: none exist — this is a greenfield CI setup). `git remote -v` confirms the origin is `https://github.com/ngoc31031997/ConceptFlow.git`, i.e. a GitHub-hosted repo, which is why GitHub Actions is the natural default rather than an ambiguous guess. **This is a decision, not a hard requirement** — confirm with the team before relying on it if another platform is preferred.
- **Pipeline config file location**: `.github/workflows/ci.yml` (does not exist yet — to be created).

## 2. Proposed Workflow Structure
Given this is a polyglot monorepo (6 Python/FastAPI services + 1 Go service + 1 infra-only service), use a matrix job for the Python services plus a dedicated Go job, both gated behind path filters so unrelated service changes don't retrigger unrelated builds:

```yaml
name: CI
on:
  pull_request:
  push:
    branches: [main]

jobs:
  python-services:
    strategy:
      matrix:
        service: [content-plugin, script-processing, tts, rendering, video-assembly, publisher]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: '3.12'
      - name: System deps (rendering only needs cairo/pango/ffmpeg)
        if: matrix.service == 'rendering'
        run: sudo apt-get update && sudo apt-get install -y pkg-config libcairo2-dev libpango1.0-dev ffmpeg
      - run: pip install -r services/${{ matrix.service }}/requirements-dev.txt
      - run: pytest services/${{ matrix.service }}/tests -q --junitxml=reports/${{ matrix.service }}-junit.xml

  orchestrator:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go build ./... 
        working-directory: services/orchestrator
      - run: go vet ./...
        working-directory: services/orchestrator
      - run: go test ./... -v
        working-directory: services/orchestrator

  compose-validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: cp .env.example .env
      - run: docker compose config
      - run: docker compose build

  sonarqube:
    needs: [python-services, orchestrator]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: sonarsource/sonarqube-scan-action@v3
        env:
          SONAR_TOKEN: ${{ secrets.SONAR_TOKEN }}
          SONAR_HOST_URL: ${{ secrets.SONAR_HOST_URL }}

  owasp-dependency-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: dependency-check/Dependency-Check_Action@main
        with:
          project: 'ConceptFlow'
          path: '.'
          format: 'HTML,JSON'
        env:
          JAVA_HOME: /opt/jdk
```

## 3. SonarQube Integration
**Purpose**: Static code analysis for code quality, maintainability, bugs, code smells, and security hotspots across both the Python services and the Go service.

- **Prerequisites**:
  - SonarQube server URL (self-hosted or SonarCloud) and auth token stored as CI secrets `SONAR_HOST_URL`, `SONAR_TOKEN` (never hardcoded).
  - `sonar-scanner` CLI (used via `sonarsource/sonarqube-scan-action`), which auto-detects Python and Go sources.
- **Configuration file**: `sonar-project.properties` at repo root (to be created):
  ```properties
  sonar.projectKey=conceptflow
  sonar.sources=services
  sonar.tests=services
  sonar.test.inclusions=**/tests/**,**/*_test.go
  sonar.exclusions=**/.venv/**,**/__pycache__/**
  sonar.python.coverage.reportPaths=reports/coverage-*.xml
  sonar.go.coverage.reportPaths=services/orchestrator/coverage.out
  ```
  Note: Python coverage reports don't currently exist (no `pytest-cov` configured — see unit-test-instructions.md); add `pytest-cov` and `--cov --cov-report=xml:reports/coverage-<service>.xml` to the pytest invocation before wiring coverage into Sonar. For Go, add `go test ./... -coverprofile=coverage.out`.
- **Quality Gate**: Configure the pipeline to fail the build if the SonarQube Quality Gate fails (block merge on new bugs/vulnerabilities/coverage regression).

## 4. OWASP Security Scanning
**Purpose**: Identify known vulnerabilities in dependencies and (optionally) runtime vulnerabilities.

- **OWASP Dependency-Check** (dependency/SCA scanning — always include):
  - Scans each service's `requirements.txt` and the Go `go.sum` against the NVD.
  - Pipeline step (example):
    ```bash
    dependency-check --project "ConceptFlow" --scan . --format HTML --format JSON --out reports/dependency-check
    ```
  - Fail the build on CVSS score >= 7 (High/Critical), per standard project risk tolerance — confirm the actual threshold with the team since no explicit threshold was specified in this project's NFR docs.
- **OWASP ZAP** (DAST — include since 6 of the 8 units expose a running FastAPI HTTP service):
  - Baseline scan against each FastAPI service once deployed to a staging/test environment (not run in CI against ephemeral containers without real data/secrets — schedule as a nightly or pre-release job instead).
  - Pipeline step (example, GitHub Actions):
    ```yaml
    - uses: zaproxy/action-baseline@v0.x
      with:
        target: 'https://staging.example.com/content-plugin'
    ```
  - Review and triage findings; block deploy on high-severity alerts per project risk tolerance.

## 5. Secrets Management
Required CI secrets:
- `SONAR_TOKEN`, `SONAR_HOST_URL`
- Container registry credentials (if pushing built images, e.g. `GHCR_TOKEN` for GitHub Container Registry)
- `GOOGLE_OAUTH_CLIENT_ID`, `GOOGLE_OAUTH_CLIENT_SECRET` — only needed for integration/E2E jobs that actually exercise the publisher's YouTube upload; do NOT expose these to PR-triggered jobs from forks.
- `RABBITMQ_USER`/`PASS`, `POSTGRES_USER`/`PASS` — for any CI job that spins up `docker compose up` for integration testing.

Never commit secrets; reference the CI platform's secret store (GitHub Actions repo/environment secrets).

## 6. Pipeline Trigger Rules
- **On Pull Request**: `python-services` matrix, `orchestrator` job, `compose-validate`, SonarQube analysis, OWASP Dependency-Check (fast feedback, no deploy, no real secrets exposed to fork PRs).
- **On Merge to Main**: full pipeline including the scenarios in `integration-test-instructions.md` (once an automated integration harness exists), OWASP ZAP against a staging deploy, image build/push, deploy to target environment.

## 7. Local Reproduction
Before pushing, reproduce CI locally:
```bash
# Python services
cd services/<service> && source .venv/bin/activate && pytest tests/ -q

# Go service
cd services/orchestrator && go build ./... && go vet ./... && go test ./...

# Compose validation
docker compose config && docker compose build

# SonarQube (requires local server or SonarCloud token)
sonar-scanner -Dsonar.projectKey=conceptflow -Dsonar.sources=services -Dsonar.host.url=$SONAR_HOST_URL -Dsonar.token=$SONAR_TOKEN

# OWASP Dependency-Check
dependency-check --project "ConceptFlow" --scan . --format HTML --out reports/dependency-check
```
