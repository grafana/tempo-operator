---
name: deploy
description: Build and deploy the Tempo operator to the connected cluster (kind, Kubernetes, or OpenShift). Use when the user asks to deploy the operator.
---

# Deploy Tempo Operator

Build and deploy the operator to the connected cluster (kind, Kubernetes, or OpenShift).

## Steps

1. Run the deploy script from the repository root:

```bash
.claude/skills/deploy/deploy.sh
```

2. Check if the OpenTelemetry operator is already installed (`kubectl get crd opentelemetrycollectors.opentelemetry.io`). If not installed, ask the user if they want to deploy it (default: no). If yes, run:

```bash
.claude/skills/deploy/deploy-otel.sh
```

3. Report the result to the user.
