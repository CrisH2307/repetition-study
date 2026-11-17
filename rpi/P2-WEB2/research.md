# P2-WEB2 research
- Contract v4 (api/openapi.yaml): POST /v1/consent {text_version}, GET/POST /v1/feedback, GET /v1/me/export, DELETE /v1/me {confirm: "delete my account"}, POST /v1/courses/{id}/import (multipart file), GET /v1/imports/{id}.
- No endpoint says whether a user already accepted consent. The UI remembers it per user in localStorage (`sr:consent:<user id>`); a new browser asks again, and POST /v1/consent is idempotent. A `consented` field on GET /v1/me would fix this (contract change, Cris).
- On qa today the Go API serves none of the v4 routes (P2-API in parallel). loop.e2e.ts marks consent in localStorage so it runs on either API version.
