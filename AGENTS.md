# Agent Rules

- This repository owns provider catalog, provider connection, provider
  observability, and provider orchestration behavior.
- Do not edit protobuf source or generated contract bindings here.
- If a public contract must change, make that change in `code-code-contracts`
  first, then update this repository to the released contract version.
- Keep provider state and lifecycle transitions inside the provider domain.
- Do not move auth, egress, profile, catalog, notification, UI, or deployment
  behavior into this repository.
- Keep changes narrow to one provider use case at a time.
