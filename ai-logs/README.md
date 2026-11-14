# AI Engineering & Architecture Logs

> Owner-provided English submission record. This edited reconstruction is not a verbatim export; its quoted exchanges and timestamps have not been authenticated against the recorder. The [original recorded prompts and responses](https://github.com/smurfac4/game-key-marketplace/blob/7ffad4f/ai-logs/CHAT-LOG.md) remain available. Verification statements below are supplied summaries, not new test runs.

This directory contains the engineering logs, architectural implementation history, verification records, and owner-provided English collaboration record for the **KEYSPACE** game-key marketplace project.

## Overview of Files

| File | Description |
| --- | --- |
| [CHAT-LOG.md](CHAT-LOG.md) | English reconstruction of architectural prompts, feature specifications, code reviews, and assistant responses |
| [WORKLOG.md](WORKLOG.md) | Summary of backend and frontend engineering milestones, Verification & QA, and deployment |
| [VERIFICATION.md](VERIFICATION.md) | Executed test suites, database concurrency checks, query execution plans, and production smoke tests |

## Architectural & Workflow Highlights

The development workflow followed a structured, specification-driven approach:
1. **System Design & Monorepo Scaffolding**: Established a clear separation between `/web` (React, TypeScript, Vite, Tailwind CSS, TanStack Query) and `/api` (Go, `chi` router, `pgx` connection pool, PostgreSQL 17).
2. **Database Integrity & Concurrency**: Designed reversible `golang-migrate` schemas with strict foreign-key constraints, `pg_trgm` GIN indexes for fuzzy search, and transactional key allocation (`SELECT ... FOR UPDATE SKIP LOCKED`) to prevent double-selling under concurrent load.
3. **Modular Monolith Homepage**: Implemented independent backend packages, repositories, and endpoints (`/api/home/banners`, `/best-sellers`, `/deals`, `/new-releases`) so each frontend block fetches and fails in isolation without hardcoded content.
4. **Security & Session Management**: Built `bcrypt` password authentication, revocable `HttpOnly` JWT sessions, strict Origin/CSRF checks on mutating endpoints, and per-route rate limiting.
5. **Verification & Live Deployment**: Validated via Go race-detector suites, PostgreSQL integration tests, 15 Playwright + Axe accessibility scenarios, and deployed to Render with managed HTTPS and private database networking.

## Related Documentation

- [Main Repository README](../README.md) — Quick start and architecture overview
- [Architectural Decisions (DECISIONS.md)](../DECISIONS.md) — Trade-offs, schema rationale, and known scope boundaries
- [Deployment Guide](../docs/DEPLOYMENT.md) — Render Blueprint configuration and production environment notes
