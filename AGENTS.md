# AI Working Guide for House Information Collector

This file is the first reference for any AI agent working in this repository.

## Language and Collaboration

- Respond to the user in Korean unless the user explicitly asks otherwise.
- Technical reference documents may be written in English.
- This repository is currently in architecture/design transition. Do not assume the existing prototype code is the final target structure.
- Do not start broad implementation unless the user explicitly asks for implementation. The current priority is clear architecture, collection policy, data model, and verification design.

## Project Identity

- Korean product name: `집 모아`
- English full name: `House Information Collector`
- Short command/package identity: `HIC`
- Purpose: collect public housing recruitment notices from SH first, later LH, preserve source artifacts, extract attachment data, normalize individual housing-unit records, and expose only QA-approved normalized data.

## Non-Negotiable Product Policy

- Collect only recruitment-notice family posts.
- Include:
  - recruitment notices
  - correction notices
  - additional recruitment notices
  - remaining-unit recruitment notices
- Exclude completely:
  - winner announcements
  - applicant result announcements
  - contract guides
  - system notices
  - generic alerts
- Excluded posts must not be stored in the normal database archive. They may appear only in dry-run reports or QA fixtures.

## Domain Definition

The user now defines the normalized end-user grain as `공급항목(Offering)`.
It represents one application-selectable unit. `unit_no` is nullable because some notices accept applications by complex, area, category, gender, count, and price before assigning exact units later.

Core hierarchy:

```text
Agency -> Board -> Notice -> Attachment -> Extracted Artifact -> Offering
```

`Offering` is the core end-user data grain. Application-selectable information often lives inside XLSX/PDF/HWP attachments, not in the notice HTML.

## Target Stack

- Language: Go
- HTTP framework: Echo
- CLI: Cobra subcommands
- Database: PostgreSQL
- Query layer: sqlc must be used for important typed SQL.
- ORM: use an ORM only where it adds value; do not hide important reporting/query logic behind opaque ORM behavior.
- Optional support infrastructure:
  - Redis as optional queue/lock adapter
  - MinIO as optional S3-compatible object-store adapter

Redis and MinIO are supporting extension features only. Core discovery, extraction, normalization, and QA design must not strongly depend on them.

## Target Package Direction

Prefer domain packages under `pkg/`, not a monolithic `internal/` layout:

```text
pkg/global
pkg/discovery
pkg/extraction
pkg/normalize
pkg/llm
pkg/workflow
pkg/qa
```

Expected root directories:

```text
pkg/
schema/schema.sql
data/samples
data/expected
docs/
폐기/
```

## Layer Boundaries

Discovery:

- Finds board rows, candidate notice details, and attachment metadata.
- Decides whether a post belongs to the recruitment-notice family.
- Must reject non-recruitment posts before attachment download and DB persistence.

Extraction:

- Preserves original attachments through an `ObjectStore` interface.
- Default object storage implementation should be local filesystem.
- MinIO is only an optional S3-compatible adapter.
- Extracts mechanical artifacts from PDF, XLSX, HWP, and HTML.

Normalize:

- Converts extracted artifacts into domain fields.
- Address, unit number, area, deposit, monthly rent, conversion rules, and schedules must be structured columns/tables, not only raw JSON.

LLM:

- Optional assist layer.
- Use only for low-confidence or failed deterministic parsing segments.
- LLM outputs must include schema version, prompt version, model, input hash, confidence, and source span.

Workflow:

- Orchestrates stages, retries, checkpoints, and idempotency.
- Failed attachment/artifact offering records must be independently retryable.

QA:

- Guards promotion to serving/API.
- Data that fails QA must not be promoted to end-user query models.

## Infrastructure Rule

Use ports/interfaces:

```text
ObjectStore -> local filesystem by default, MinIO adapter optionally
Queue      -> in-process default, Redis adapter optionally
Lock       -> in-process/PostgreSQL advisory fallback, Redis adapter optionally
```

Do not let domain packages import MinIO/Redis clients directly.

## Current Repository State

The current Go code is a prototype from the first exploration phase. It can be used as reference for:

- SH rental list/detail request shape
- attachment metadata parsing
- attachment download pattern
- XLSX streaming extraction proof
- simple PDF text extraction proof
- early database table ideas

It should not be treated as final architecture.

Important existing documents:

- `README.md`: current project summary
- `docs/GLOSSARY.md`: canonical terminology
- `docs/AI_HANDOFF.md`: next-session context snapshot
- `docs/IMPLEMENTATION_BACKLOG.md`: recommended work units
- `docs/architecture-redesign.md`: target architecture
- `docs/architecture-layers.svg`: visual architecture
- `폐기/`: archived/deprecated documents, useful only as historical evidence

## Verification Expectations

Before claiming completion:

- Run relevant tests or validation commands.
- Read outputs.
- State what was verified.

For documentation-only changes, at minimum verify:

```bash
rg -n "..." .
test -f <new-file>
```

For SVG changes:

```bash
python3 -m xml.etree.ElementTree docs/architecture-layers.svg
```
