# Contributing

Thanks for considering a contribution. This document covers how to get a
working development environment, the conventions the codebase follows, and
the pull request process.

## Code of Conduct

Participation in this project is governed by [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
By contributing you agree to abide by its terms.

## Reporting Security Issues

See [SECURITY.md](SECURITY.md). Do not open public issues for vulnerabilities.

## Development Setup

Prerequisites: Go 1.24+, Node.js 20+, Docker (recommended).

```bash
# Start the full stack (Postgres + MinIO + backend + frontend)
docker-compose up --build

# Or run components individually
cd backend  && go run cmd/server/main.go
cd frontend && npm install && npm run dev
```

Copy `.env.example` to `.env` and fill in values. Never commit `.env`.

See [README.md](README.md) and [CLAUDE.md](CLAUDE.md) for full architecture
and build details.

## Running Tests

```bash
cd backend  && go test ./...
cd frontend && npm test
```

End-to-end encryption tests in `frontend/src/shared/crypto/e2e.test.ts`
must pass for any change touching the crypto code. Backend equivalents:
`services/chat_service_scheme_test.go` and `services/user_service_vault_test.go`.

## Linting

Backend lint must be run inside Docker (the native Windows binary leaks
memory):

```bash
cd backend
docker run --rm --memory=5g -v "${PWD}:/app" -w /app \
  golangci/golangci-lint:v1.64.8 golangci-lint run --timeout=10m
```

Frontend:

```bash
cd frontend
npm run lint
npm run knip   # find unused exports / files / deps
```

## Pull Request Process

1. **Fork and branch** from `main`. Use a descriptive branch name, e.g.
   `fix/group-key-rotation` or `feat/typing-indicator`.
2. **Keep PRs focused.** One logical change per PR. Refactors and feature
   work should be separate.
3. **Write tests** for new behavior. Crypto, auth, and message-flow changes
   require tests — non-negotiable.
4. **Run lint and tests** locally before pushing.
5. **Update documentation** (`README.md`, `CLAUDE.md`, `ops/E2E.md`) if
   the change affects architecture, API surface, or crypto.
6. **Write a clear PR description**: what changes, why, and how to verify.
7. **No force-push** to a PR branch once review has started — push fixup
   commits instead. Squash is fine before merge.

## Commit Messages

Follow the existing style in `git log`:

```
<type>(<scope>): <short summary>

<optional body explaining motivation and trade-offs>
```

Types: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`, `ci`.
Scopes used so far include `e2e`, `nginx`, `auth`, `chats`, `calls`,
`groups`, `push`.

## Code Style

- **Go**: follow the Uber Go Style Guide. Wrap errors with context.
  Keep packages small and focused.
- **TypeScript / React**: ESLint `typescript-eslint/strict` config is the
  source of truth. Prefer function components and hooks. No `any` without
  a comment explaining why.
- **No dead code.** `npm run knip` should stay clean.
- **Comments explain WHY, not WHAT.** A well-named function does not need
  a docstring describing what it does.

## Architectural Boundaries

- Handlers must not access repositories directly — always go through services.
- The server must never see plaintext message content. All user-visible text
  flows through scheme=2 E2E encryption. See `ops/E2E.md`.
- New WebSocket actions go in `backend/internal/handlers/websocket_actions.go`
  and must include `chat_id` for authorization.

## License

By contributing, you agree that your contributions will be licensed under
the project's [AGPL-3.0-or-later](LICENSE) license.
