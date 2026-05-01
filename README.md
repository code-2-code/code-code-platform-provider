# code-code-platform-provider

Provider catalog, connection, observability, and orchestration services for
Code Code.

This repository owns:

- `packages/platform-k8s/internal/providerservice`: provider catalog,
  provider runtime registry, observability, template, and provider transport
  behavior.
- `packages/platform-k8s/internal/providerconnect`: provider connection
  sessions and credential/OAuth connection flows.
- `packages/platform-k8s/internal/providerorchestration`: provider
  orchestration workflows, activities, probes, and schedules.
- `packages/platform-k8s/cmd/platform-provider-service`: provider service
  entrypoint.
- `packages/platform-k8s/cmd/platform-provider-orchestration-service`:
  provider orchestration worker/service entrypoint.
- `packages/agent-runtime-contract`: local provider runtime interfaces used by
  provider implementations in this split.
- `code-code-contracts`: generated shared contracts as a Git submodule.

Useful checks:

```bash
cd packages/agent-runtime-contract && go test ./credential/... ./provider/...
cd packages/platform-k8s && go test ./internal/providerservice/... ./internal/providerconnect/... ./internal/providerorchestration/... ./cmd/platform-provider-service ./cmd/platform-provider-orchestration-service
```
