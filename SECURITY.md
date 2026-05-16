# Security Policy

This project implements end-to-end encryption (X25519 ECDH + AES-256-GCM)
and handles user credentials, message contents, and file attachments.
Security reports are taken seriously.

## Reporting a Vulnerability

**Do not open public GitHub issues for security problems.**

Email: **fkulart@mail.ru**
Subject prefix: `[SECURITY] messenger:`

Please include:
- Affected component (backend / frontend / Android / ops)
- Affected version or commit SHA
- Steps to reproduce or a proof-of-concept
- Impact assessment (data exposure, auth bypass, RCE, etc.)
- Your suggested remediation, if any

If you prefer encrypted email, request a PGP key in your first message and
one will be provided.

## Response Targets

- Initial acknowledgement: within **72 hours**
- Triage and severity assessment: within **7 days**
- Fix or mitigation plan: within **30 days** for High/Critical issues
- Public disclosure: coordinated with reporter, normally after a patch ships

## Scope

In scope:
- Server (Go / Gin / GORM) and database access controls
- Authentication, JWT handling, session lifecycle
- End-to-end encryption design and implementation
  (`frontend/src/shared/crypto/`, `ops/E2E.md`)
- WebSocket message handling and authorization
- File upload / storage / presigned URL handling
- WebRTC signaling and call authorization
- Android client (Capacitor) build and FCM integration

Out of scope:
- Vulnerabilities requiring physical device access to a logged-in client
- Issues that require the user to install a malicious build
- Denial-of-service via unrestricted resource consumption on self-hosted
  instances (rate limits are best-effort)
- Findings against third-party services (Firebase, MinIO, PostgreSQL)
  themselves — report those upstream

## Safe Harbor

Good-faith security research is welcomed. We will not pursue legal action
against researchers who:
- Make a good-faith effort to avoid privacy violations, data destruction,
  and service disruption
- Only interact with accounts they own or have explicit permission to test
- Give us reasonable time to remediate before public disclosure
- Do not exfiltrate data beyond what is necessary to demonstrate the issue

## Cryptography

The E2E design is documented in `ops/E2E.md`, including the threat model
and known limitations. Review and feedback on the cryptographic design
is explicitly invited.
