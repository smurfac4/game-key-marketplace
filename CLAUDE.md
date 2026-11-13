# Project rules

- Keep commits small and focused: one coherent change per commit. Do not mix refactors and features.
- Add tests for business logic, including failure paths and boundary cases. Keep tests close to the code. A scaffold without business logic does not need placeholder tests.
- No hardcoded content in the frontend. Marketplace data must come from the REST API and PostgreSQL. Do not embed products, prices, sellers, or mock catalog arrays in components. Keep future UI copy in a dedicated localization/content layer, not inline in components.
- Explain non-obvious decisions in DECISIONS.md. Keep source free of prose comments; preserve required compiler and interpreter directives.
- Record every scope trade-off in DECISIONS.md, including the reason, consequences, and deferred work. Update the log in the same change.
- Keep web code in /web, Go REST code in /api, and sanitized AI assistance records in /ai-logs.
- Use React Router for routing, TanStack Query for server state, chi for HTTP routing, and pgx for PostgreSQL access.
- Keep secrets out of source control and AI logs. Document environment variables in .env.example.
- Run npm run build, go test ./..., and go vet ./... (Go commands from /api) before delivering code. Report any checks that could not run.
- Do not implement marketplace features until requested. Add schema migrations when the first persisted domain model is introduced.
