# House Information Collector Implementation Backlog

This backlog is for future sessions. It reflects current user decisions and architecture direction as of 2026-05-14.

## Phase 0. Repository Reset And Guardrails

Goal: make the repository safe for clean implementation.

Tasks:

- Add `schema/`, `data/samples/`, `data/expected/`, and `pkg/` directories.
- Move the current embedded PostgreSQL schema into `schema/schema.sql`.
- Decide which prototype code is kept as reference and which is replaced.
- Keep old/obsolete architecture notes under `폐기/`.
- Add project-level tests that prevent non-recruitment posts from being persisted.

Acceptance checks:

```bash
test -f schema/schema.sql
test -d data/samples
test -d data/expected
test -d pkg
go test ./...
```

## Phase 1. CLI And Global Layer

Goal: create a stable command surface and shared infrastructure contracts.

Tasks:

- Replace ad-hoc `flag` command parsing with Cobra.
- Add root `hic` command.
- Add domain subcommands:
  - `hic discovery ...`
  - `hic extract ...`
  - `hic normalize ...`
  - `hic llm ...`
  - `hic workflow ...`
  - `hic qa ...`
- Add `pkg/global` with config, logger, clock, retry policy, and site registry.
- Add infrastructure ports:
  - `ObjectStore`
  - `Queue`
  - `Lock`

Acceptance checks:

```bash
go run . --help
go run . discovery --help
go test ./...
```

## Phase 2. Schema And Persistence

Goal: model normalized data explicitly.

Tasks:

- Define SQL schema in `schema/schema.sql`.
- Use sqlc for core typed queries.
- Decide ORM scope separately; do not use ORM as a substitute for clear SQL in reporting/serving queries.
- Add tables for:
  - collection runs
  - source boards
  - notices
  - attachments
  - stored objects
  - extracted artifacts
  - housing units
  - rent conversion rules
  - housing unit conversion estimates
  - notice schedules
  - LLM repair attempts
  - QA decisions
- Add uniqueness constraints for idempotency:
  - agency + board + source sequence
  - attachment source identifiers
  - housing unit per notice/attachment/source row/unit number

Acceptance checks:

```bash
sqlc generate
go test ./...
```

## Phase 3. Discovery Layer

Goal: find only recruitment-notice family posts.

Tasks:

- Build `pkg/discovery`.
- Add board registry for SH rental.
- Re-verify SH sale board endpoint and add it to registry.
- Preserve selector profiles per board.
- Add recruitment classifier with include/exclude rules.
- Use detail-page inspection when title is ambiguous.
- Return rejected posts only in run reports, not persistence.

Acceptance checks:

```bash
hic discovery sh --board rental --pages 1 --dry-run
hic discovery sh --board sale --pages 1 --dry-run
go test ./pkg/discovery/...
```

## Phase 4. Object Storage And Extraction

Goal: preserve source files and extract mechanical artifacts.

Tasks:

- Build `pkg/extraction`.
- Add local filesystem `ObjectStore` implementation as the default.
- Add MinIO adapter as optional extension.
- Store object key, sha256, content type, size, and original filename.
- Add XLSX streaming extractor.
- Add PDF text/table candidate extractor.
- Add HWP unsupported handling first, then HWP extractor later.
- Add attachment type classifier:
  - notice PDF
  - housing unit list XLSX
  - schedule PDF
  - winner/applicant file
  - application form
  - unsupported

Acceptance checks:

```bash
hic extract attachment --id 123
hic extract pdf --file data/samples/sh/recruitment_pdf/sample.pdf
hic extract xlsx --file data/samples/sh/unit_list/sample.xlsx
go test ./pkg/extraction/...
```

## Phase 5. Normalize Layer

Goal: convert artifacts into canonical domain records.

Tasks:

- Build `pkg/normalize`.
- Normalize notice metadata.
- Normalize housing units.
- Normalize address, district, legal dong, housing name, building name, unit number, floor.
- Normalize exclusive area and pyeong.
- Normalize deposit and monthly rent as numeric KRW.
- Normalize rent conversion rules from notice text/PDF.
- Calculate per-unit conversion estimates.
- Normalize schedules into structured schedule records.
- Keep source spans for every derived field.

Acceptance checks:

```bash
hic normalize notice --id 123
hic normalize units --attachment-id 123
hic normalize schedules --notice-id 123
hic normalize conversion --notice-id 123
go test ./pkg/normalize/...
```

## Phase 6. LLM Assist Layer

Goal: use LLM safely where deterministic parsing is weak.

Tasks:

- Build `pkg/llm`.
- Define JSON schemas for repair outputs.
- Store prompt version, schema version, model, input hash, and output hash.
- Require source spans.
- Use LLM only for failed/low-confidence segments.
- Add conflict detection against deterministic parser output.

Acceptance checks:

```bash
hic llm repair --artifact-id 123 --dry-run
go test ./pkg/llm/...
```

## Phase 7. QA And Workflow

Goal: orchestrate the full pipeline while protecting layer boundaries.

Tasks:

- Build `pkg/qa`.
- Build `pkg/workflow`.
- Add checkpointing per stage.
- Add retry by failed attachment/artifact.
- Add QA decision records.
- Add serving promotion only after QA pass.
- Add sample regression cases under `data/expected`.

Acceptance checks:

```bash
hic workflow collect-sh --board rental --pages 1
hic qa sample --case sh-rental-basic
go test ./pkg/workflow/... ./pkg/qa/...
```

## Phase 8. API And Dashboard Support

Goal: expose normalized records, not raw blobs.

Tasks:

- Keep Echo for API.
- Expose notices, housing units, conversion rules, schedules, and source evidence.
- Ensure API defaults to QA-approved serving data.
- Keep raw artifacts available only for audit/debug endpoints.

Acceptance checks:

```bash
hic serve
curl http://localhost:9552/health
curl http://localhost:9552/notices
curl http://localhost:9552/units
```

## First Implementation Recommendation

Start with Phase 0 through Phase 2 before touching live SH crawling again.

Reason:

- The user wants clean architecture and normalized data.
- Existing prototype already proved that SH collection is feasible.
- The bigger risk is storing incomplete raw-shaped data again.
