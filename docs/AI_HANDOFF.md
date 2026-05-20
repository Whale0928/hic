# House Information Collector AI Handoff

Last updated: 2026-05-20

This document is written for a future AI session that needs to continue the `집 모아 / House Information Collector` project without losing context.

## 1. Project Summary

House Information Collector, short name `HIC`, is a Go-based data collection and normalization system for public housing recruitment notices.

The active targets are SH and LH/MyHome. The user wants a system that can discover public housing recruitment notices, preserve original source files, extract data from large attachments, normalize application-selectable Offering records, and store the result in PostgreSQL.

The project is not primarily a generic crawler. It is a structured evidence-preserving data pipeline for public housing recruitment information.

## 2. User's Current Intent

The user wants to redesign and rebuild the project with clean architecture.

Important instruction from the user:

- Do not collect non-recruitment posts.
- Build Cobra-based domain subcommands.
- Keep `schema/schema.sql` in the project root area.
- Keep sample PDF/HWP/XLSX files under `data/`.
- Use MinIO, but only as optional S3-compatible supporting infrastructure.
- Use Redis, but only as optional queue/lock supporting infrastructure.
- Split the system by domain packages.
- Build strict domain-level verification commands.
- Treat this as a long-term system that can later collect from LH as well as SH.

## 3. Project Names

```text
Korean name: 집 모아
English name: House Information Collector
Short name: HIC
CLI command: hic
```

Use `HIC` for package/command identity, like an acronym.

## 4. Canonical Paths

Repository root:

```text
/Users/hgkim/workspace/hic
```

Obsidian/reference folder:

```text
/Users/hgkim/Documents/sync/프로젝트/House Information Collector
```

Old folder that should not be used:

```text
/Users/hgkim/Documents/New project
```

The old folder was intentionally removed earlier. Do not recreate it.

## 5. Existing Root Documents

Read these first:

```text
AGENTS.md
README.md
docs/GLOSSARY.md
docs/AI_HANDOFF.md
docs/IMPLEMENTATION_BACKLOG.md
docs/architecture-redesign.md
docs/architecture-layers.svg
docs/glossary-presentation.html
```

Archived historical documents are in:

```text
폐기/
```

Archived documents may contain useful evidence from exploration, but they are not current architecture.

## 6. Current Implementation Snapshot

Current active implementation is under `pkg/<domain>`, with `schema/schema.sql` as the PostgreSQL schema source.

Implemented and verified in the 2026-05-20 session:

- SH rental discovery rejects non-recruitment posts before persistence.
- SH attachment preservation stores originals under stable object keys and HTML previews under `hic-artifacts`.
- Extraction supports PDF text/table rows, XLSX rows, HTML preview text, HWPX text, and HWP external-tool extraction with `hwp_unsupported` fallback.
- SH artifact source spans for preserved objects use `object://...`; QA-approved serving records reject legacy local path spans such as `pdf://.data/...`.
- LH/MyHome collection is independent from SH and supports rental/sale endpoints:
  - `rsdtRcritNtcList`
  - `ltRsdtRcritNtcList`
- LH/MyHome fields are normalized to offerings and application schedules.
- `unit_no` remains nullable; grouped MyHome offerings and grouped LLM/PDF offerings can be approved when they have a valid application label/source evidence and supply count or another application unit discriminator.
- LLM repair has:
  - JSON schema constrained response parsing.
  - prompt/schema/model/input hash attempt records.
  - global max-attempt guard capped at 1500.
  - `llm candidates` default policy that excludes notices already having QA-approved offerings.
  - successful repair output upserted into pending `offerings`, then promoted by QA.
- Serving/API defaults:
  - `/offerings` returns QA-approved offerings only unless `qa_status` is explicitly provided.
  - `/schedules` returns schedules only for notices with approved offerings.

Fresh verification snapshot:

```text
go test ./...: pass
llm_candidates default: 0
offerings: 452 total
approved serving offerings: 356
myhome approved/rejected: 336 / 56
sh approved/rejected: 20 / 40
approved legacy local path source_span: 0
llm_repair_attempts: 2
```

Important CLI examples:

```bash
go run . workflow collect-sh --board rental --pages 1 --dry-run=false --preserve-attachments --max-age-days 0 --skip-existing=false --object-root .data/objects-check
go run . workflow collect-lh --kind rental --num-rows 200 --all-pages --dry-run=false --agency-filter LH
go run . workflow collect-lh --kind sale --num-rows 200 --all-pages --dry-run=false --agency-filter LH
go run . llm candidates --limit 20
go run . llm candidates --limit 20 --include-approved-notices
go run . llm repair --artifact-id 48 --dry-run=false --max-input-chars 20000 --max-attempts 1500
go run . qa promote-offerings
go run . serve
```

## 7. Historical Prototype Code

Existing code is exploratory and should be treated as prototype evidence, not the final target.

Current prototype capabilities include:

- `main.go`
  - `migrate`
  - `collect-sh`
  - `serve`
- `internal/sh`
  - SH HTTP client
  - rental list/detail parsing
  - attachment metadata parsing
  - notice classification
- `internal/collect`
  - SH rental collection workflow
  - attachment download
  - XLSX/PDF extraction trigger
- `internal/extract`
  - PDF text extraction
  - XLSX streaming row extraction
  - early unit candidate inference
- `internal/store`
  - embedded schema string
  - PostgreSQL persistence
- `internal/api`
  - simple Echo endpoints

This prototype proves feasibility, but it violates the target architecture in several ways:

- schema is embedded in Go instead of root `schema/schema.sql`
- prototype code has been archived under `폐기/prototype-internal`; active implementation is under `pkg/<domain>`
- collection stores too much raw/prototype data
- recruitment-only gating is not strict enough
- normalized unit model is incomplete
- Redis/MinIO optional-port strategy is not implemented

## 8. Known SH Site Findings

These findings came from prior direct exploration and prototype code. Re-verify them before making production-grade assumptions because public websites can change.

Base:

```text
https://www.i-sh.co.kr
```

Known SH rental list path:

```text
/app/lay2/program/S48T1581C563/www/brd/m_247/list.do?multi_itm_seq=2
```

Known SH rental detail path:

```text
/app/lay2/program/S48T1581C563/www/brd/m_247/view.do
```

Known in-progress notice JSON path:

```text
/houseinfo/map/selectNoticeInProgressList.do
```

Known attachment download pattern:

```text
/app/com/file/innoFD.do?brdId={brdId}&seq={seq}&fileTp={fileTp}&fileSeq={fileSeq}
```

Known attachment preview pattern:

```text
/app/com/util/htmlConverter.do?brd_id={brdId}&seq={seq}&data_tp=A&file_seq={fileSeq}
```

Known list/detail behavior:

- Rental list HTML is table-based.
- Detail sequence can appear in JavaScript such as `getDetailView('304295')`.
- Detail page is more stable as a form-style POST.
- Attachment metadata appears in a JavaScript array such as `initParam.downList`.

Known SH house information JSON candidates from old research:

```text
/houseinfo/map/selectHouseRentWithFilter.do
/houseinfo/map/selectHouseBscDetail.do
/houseinfo/map/selectSplyTyInfo.do
```

The SH sale board was discussed as important, but the final stable endpoint still needs to be re-verified in a fresh site analysis pass.

## 9. Evidence From Prior Collection Experiment

Historical experiment results from archived docs:

First SH rental page scan:

```text
list rows: 10
detail notices: 10
attachment metadata: 38
attachment downloads: 38
attachment extensions: PDF 37, HWP 1
XLSX rows: 0
unit candidates: 0
```

Twenty-page scan without downloads:

```text
pages=20
list_rows=200
details=200
attachments=728
previews=0
downloads=0
xlsx_rows=0
offerings=0
errors=0
```

Focused collection for known XLSX-heavy notices:

```text
pages=20
list_rows=200
details=2
attachments=17
previews=0
downloads=17
xlsx_rows=2702
offerings=776
pdf_texts=8
pdf_failed=0
unsupported=3
errors=0
```

Known notice examples:

```text
296598: 2025년 하반기 청년 매입임대주택 우선공급
296353: 2025년 1차 청년 매입임대주택
304295: test/example seq used in parser fixture for 2026 두레주택 모집공고
```

Known XLSX unit candidates:

```text
296353 / file_seq 57 / 주택목록: 751 offerings
296598 / file_seq 2 / 주택목록: 25 offerings
296353 / file_seq 54 / 당첨자 명단: 0 offerings
296353 / file_seq 55 / 예비자 명단: 0 offerings
296353 / file_seq 56 / 커트라인: 0 offerings
```

Important lesson:

Applicant/winner spreadsheets can contain personal-information headers such as `접수번호`, `성명`, `생월일`. These must not become housing-unit records.

## 10. Data That Must Be Normalized

Do not leave the following only in raw JSON.

Notice:

```text
agency
board_kind
source_seq
title
posted_at
notice_type
notice_subtype
source_url
```

Housing unit:

```text
supply_category
list_no
district
address
legal_dong
housing_name
building_name
unit_no
floor_no
unit_type
structure_type
elevator_installed
```

Area and price:

```text
exclusive_area_m2
area_pyeong
deposit_krw
monthly_rent_krw
```

Rent conversion:

```text
rent_to_deposit_allowed
rent_to_deposit_max_ratio
rent_to_deposit_annual_rate
rent_unit_krw
deposit_per_rent_unit_krw
deposit_to_rent_allowed
application_cycle
reapply_limit
application_method
max_convertible_rent_krw
estimated_additional_deposit_krw
estimated_min_monthly_rent_krw
```

Schedules:

```text
schedule_type
label
starts_at
ends_at
date_text
channel
note
source_text
```

Evidence:

```text
source_span
source_artifact_id
source_sheet
source_row
source_cell
source_page
confidence
```

## 11. Target Architecture

Layer order:

```text
External Infrastructure
Internal Infrastructure
Discovery
Discovery QA
Extraction
Extraction QA
Normalize
Normalize QA
LLM Assist
LLM QA
Serving Promotion / API
```

External infrastructure:

```text
SH/LH public sites
LLM API
optional MinIO
optional Redis
```

Internal infrastructure:

```text
PostgreSQL
ObjectStore Port
Queue Port
Lock Port
Config
Logger
Clock
Retry Policy
Site Registry
```

Redis and MinIO should never be direct dependencies of discovery, extraction, normalize, LLM, or QA domain logic.

## 12. Target Commands

Expected Cobra command direction:

```bash
hic discovery sh --board rental --pages 10
hic discovery sh --board sale --pages 10
hic extract attachment --id 123
hic extract pdf --file data/samples/sh/recruitment_pdf/sample.pdf
hic normalize notice --id 123
hic normalize schedules --notice-id 123
hic normalize conversion --notice-id 123
hic llm repair --artifact-id 123
hic workflow collect-sh --board rental --pages 20
hic qa sample --case youth-rent-2025
```

Each domain command should validate only its own layer responsibility.

## 13. LLM Strategy

LLM normalization is useful but should be constrained.

Use LLM for:

- low-confidence PDF/HWP table extraction
- ambiguous section headings
- schedule text normalization when deterministic parsing fails
- rent conversion rule extraction when text layout is unstable
- mapping varied Korean field labels to canonical schema fields

Do not use LLM for:

- basic XLSX row streaming
- obvious numeric parsing
- deterministic date formats that can be parsed locally
- replacing source evidence

Every LLM result must be validated against JSON Schema and must cite source spans.

## 14. Strong QA Rules

Discovery QA:

- Only recruitment-family posts pass.
- Non-recruitment posts do not reach attachment download.
- Re-running the same input does not create duplicates.

Extraction QA:

- Every source attachment has object key, content type, sha256, and extractor status.
- Extraction failure is isolated to the attachment/artifact.
- HWP may be `unsupported` until extractor support exists.

Normalize QA:

- Housing rows must have address, district, housing name, unit number, exclusive area, deposit, and monthly rent when available in source.
- Rent conversion rules are notice-level records.
- Per-unit conversion estimates are calculated records with source references.
- Schedules must be stored in schedule tables, not only body text.

LLM QA:

- JSON schema must pass.
- Source span is mandatory.
- Prompt version, schema version, model, and input hash are mandatory.
- LLM results conflicting with deterministic parser results require quarantine or review.

Serving QA:

- Only QA-approved normalized records should be visible in API/dashboard serving models.

## 15. Recommended First Move In A New Session

If asked to continue implementation, do this order:

1. Read `AGENTS.md`, `docs/AI_HANDOFF.md`, `docs/GLOSSARY.md`, `docs/architecture-redesign.md`.
2. Run `git status --short`.
3. Decide whether to keep or replace prototype code.
4. Create root directories if missing:

```text
schema/
data/samples/
data/expected/
pkg/
```

5. Move schema out of `internal/store/schema.go` into `schema/schema.sql`.
6. Add Cobra command skeleton.
7. Add ports/interfaces before Redis/MinIO adapters.
8. Build domain commands with tests/fixtures one layer at a time.

## 16. Do Not Forget

- The user was unhappy with raw JSON being used as the main data shape.
- Individual unit-level data in attachments is the core business value.
- Address and rent conversion fields are mandatory, not nice-to-have.
- Supply schedules must be normalized.
- SH sale board is important and should be added after rental board is stable.
- LH support should be a new site adapter, not a forked pipeline.
- Avoid hardcoded one-off parsing. Use registry/profile/fallback design.
