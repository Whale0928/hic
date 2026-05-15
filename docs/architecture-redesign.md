# 집 모아 아키텍처 재설계

작성일: 2026-05-14

## 1. 설계 목표

집 모아(House Information Collector)는 SH/LH 등 공공주택 사이트에서 **모집공고 계열만** 수집하고, 첨부자료에서 공급항목(Offering) 정보를 정규화하는 시스템이다.

이번 재설계의 핵심은 다음이다.

- 모집공고가 아닌 게시글은 DB에도 저장하지 않는다.
- Raw 데이터는 조회 모델이 아니라 감사/재처리용 증거로만 둔다.
- 주소, 호실, 모집호수, 면적, 보증금, 월임대료, 전세금액, 상호전환, 공급일정은 모두 정규화 필드가 되어야 한다.
- 각 레이어는 자신의 책임만 수행하고, 다음 레이어로 넘기기 전 검증 게이트를 통과해야 한다.
- 하드코딩된 사이트별 로직은 레지스트리와 전략 객체 뒤에 숨기고, 단계별 fallback을 명시한다.

## 2. 레이어 구조

```text
global
  PostgreSQL, optional Redis, optional S3-compatible object storage, config, logger, clock, retry policy

discovery
  게시판 탐색, 모집공고 계열 판정, 첨부 메타 발견

extraction
  원본 파일 보존, PDF/XLSX/HWP/HTML 기계적 추출

normalize
  추출 산출물을 도메인 필드로 정규화

llm
  낮은 confidence 또는 구조화 실패 구간 보정

workflow
  단계 오케스트레이션, 재시도, 체크포인트, idempotency 관리

qa
  샘플 회귀 검증, 필수 필드 누락률, source trace, 중복 검증
```

## 3. 레이어별 입력과 출력

| 레이어 | 입력 | 출력 | 통과 조건 |
|---|---|---|---|
| Discovery | board registry, page range | recruitment notice candidate, attachment metadata | 모집공고 계열만 남아야 함 |
| Extraction | recruitment notice, attachment metadata | stored object, extracted artifact | checksum, content type, extractor status가 있어야 함 |
| Normalize | extracted artifact | normalized notice, offering, schedules, conversion rules | 필수 필드와 source trace가 있어야 함 |
| LLM | low-confidence artifact | structured candidate JSON | JSON schema와 source span이 있어야 함 |
| QA | normalized records | serving promotion decision | 누락률, 중복, 계산 검증을 통과해야 함 |
| Workflow | job request | run report | 실패 단위가 격리되고 재실행 가능해야 함 |

## 4. 수집 정책

포함 대상:

- 모집공고
- 정정공고
- 추가모집
- 잔여세대 모집

제외 대상:

- 당첨자 발표
- 예비자 발표
- 계약안내
- 시스템 공지
- 일반 알림

제외 대상은 raw archive에도 저장하지 않는다. 운영 중 오탐 검증이 필요하면 `--dry-run` 또는 QA fixture로만 다룬다.

## 5. Fallback 정책

### Discovery

1. Board Registry의 정적 URL과 파라미터를 사용한다.
2. 목록 HTML 파서가 실패하면 board별 selector profile을 교체한다.
3. 제목만으로 모집공고 판정이 애매하면 상세 본문을 확인한다.
4. 상세 확인 후에도 애매하면 저장하지 않고 `rejected_unknown`으로 run report에만 기록한다.

금지:

- 제외 키워드가 강하게 매칭된 게시글을 첨부 다운로드 단계로 넘기지 않는다.
- 임시로 모든 게시글을 저장한 뒤 나중에 걸러내지 않는다.

### Extraction

1. 첨부 원본을 object storage 인터페이스에 저장한다.
2. 기본 구현은 local filesystem이고, MinIO는 S3 호환 확장 구현이다.
3. content type과 sha256을 검증한다.
4. 타입별 extractor를 실행한다.
5. extractor가 없으면 `unsupported` artifact를 만든다.
6. 추출 실패는 attachment 단위로 격리한다.

금지:

- object storage 인터페이스를 우회해 로컬 절대경로나 특정 인프라 구현만 DB에 저장하지 않는다.
- PDF/HWP 추출 실패 때문에 공고 전체를 실패시키지 않는다.

### Normalize

1. deterministic parser로 명확한 셀/텍스트 값을 정규화한다.
2. 필수 필드가 raw에만 있고 정규화 컬럼에 없으면 QA 실패다.
3. 파싱 confidence가 낮으면 LLM 후보 생성으로 넘긴다.
4. LLM 결과도 schema/source span 검증을 통과해야 저장한다.

금지:

- `source_row`, `raw_text`, `content_json`을 API 조회 모델로 사용하지 않는다.
- 주소, 가격, 상호전환, 공급일정을 JSONB 안에만 묻어두지 않는다.

### LLM

1. deterministic parser가 실패한 구간만 입력한다.
2. prompt version, schema version, model, input hash를 저장한다.
3. source span 없는 LLM 결과는 폐기한다.
4. deterministic 결과와 충돌하면 QA가 승인하기 전까지 serving으로 승격하지 않는다.

## 6. 필수 정규화 도메인

### 공고

- `agency`
- `board_kind`
- `source_seq`
- `title`
- `posted_at`
- `notice_type`
- `notice_subtype`
- `source_url`

### 공급항목

- `application_unit_label`
- `supply_method`
- `application_category`
- `supply_category`
- `list_no`
- `district`
- `address`
- `legal_dong`
- `housing_name`
- `building_name`
- `unit_no`
- `supply_count`
- `floor_no`
- `unit_type`
- `structure_type`
- `elevator_installed`
- `reserved_count`
- `gender_requirement`
- `occupancy_type`
- `capacity_persons`
- `heating_method`
- `move_in_start_text`

### 면적/가격

- `exclusive_area_m2`
- `area_pyeong`
- `deposit_krw`
- `monthly_rent_krw`
- `jeonse_deposit_krw`
- `contract_deposit_krw`
- `balance_payment_krw`
- `dormitory_fee_krw`

공급항목(Offering)은 신청 가능 단위다. `unit_no`는 동호수가 원문에 공개된 경우에만 채우고, 장기전세/기숙사처럼 단지·면적·신청유형·모집호수 조합으로 신청하는 공고에서는 비워둘 수 있다.
QA는 `application_unit_label`, `housing_name`/`complex_name`, `unit_no`, `supply_count`, 면적/거주유형, 금액 필드 중 하나 이상을 조합해 승인 여부를 판단한다.

### 상호전환

- `rent_to_deposit_allowed`
- `rent_to_deposit_max_ratio`
- `rent_to_deposit_annual_rate`
- `rent_unit_krw`
- `deposit_per_rent_unit_krw`
- `deposit_to_rent_allowed`
- `application_cycle`
- `reapply_limit`
- `application_method`
- `max_convertible_rent_krw`
- `estimated_additional_deposit_krw`
- `estimated_min_monthly_rent_krw`

### 공급일정

- `schedule_type`
- `label`
- `starts_at`
- `ends_at`
- `date_text`
- `channel`
- `note`
- `source_text`

## 7. 검증 체계

### Discovery QA

- SH 주택임대/주택분양에서 모집공고 계열만 통과한다.
- 당첨자/계약/공지 키워드는 DB 저장까지 도달하지 않는다.
- 같은 공고 재실행 시 중복되지 않는다.

### Extraction QA

- 모든 첨부 원본은 stored object key와 sha256을 가진다.
- MinIO 없이도 local filesystem object storage로 동일 검증을 수행할 수 있어야 한다.
- PDF는 page/text artifact를 만든다.
- XLSX는 sheet/row artifact를 만든다.
- HWP는 extractor 준비 전까지 `unsupported` artifact로 남긴다.

### Normalize QA

- 공급항목은 신청 가능 단위로 검증한다.
- 동호수가 공개된 공급항목은 `unit_no`와 source span을 우선 근거로 삼는다.
- 동호수가 공개되지 않은 공급항목은 `application_unit_label`, 단지명/주택명, 전용면적 또는 거주유형, 모집호수, 보증금/월임대료/전세금액/기숙사비 중 적용 가능한 금액 필드를 조합해 검증한다.
- 상호전환 규칙은 공고 단위로 저장되고, 공급항목별 계산값이 생성되어야 한다.
- 공급일정은 본문 텍스트가 아니라 일정 테이블에 저장되어야 한다.

### LLM QA

- 결과 JSON은 schema validation을 통과해야 한다.
- 모든 필드에 source span이 있어야 한다.
- prompt/schema/model/input hash가 저장되어야 한다.

### Workflow QA

- discovery, extraction, normalize, llm, qa 단계별 checkpoint가 있어야 한다.
- 실패한 attachment 또는 artifact만 재실행 가능해야 한다.
- QA gate 실패 데이터는 serving/API로 승격하지 않는다.

## 8. 패키지와 커맨드

패키지:

```text
pkg/global
pkg/discovery
pkg/extraction
pkg/normalize
pkg/llm
pkg/workflow
pkg/qa
```

Cobra 커맨드:

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

## 9. 인프라

Docker Compose 대상:

- PostgreSQL: 필수 영속 저장소
- Redis: 선택 지원 인프라, queue/lock adapter 후보
- MinIO: 선택 지원 인프라, S3-compatible object store adapter 후보

의존성 원칙:

- 핵심 도메인과 workflow는 `ObjectStore`, `Queue`, `Lock` 포트에만 의존한다.
- local 개발과 테스트는 filesystem object storage, in-process queue/lock fallback으로 실행 가능해야 한다.
- Redis/MinIO 장애나 미기동은 discovery, extraction, normalize 설계를 막는 강한 의존성이 되지 않는다.

Object bucket/key 네이밍 후보:

- `hic-originals`
- `hic-artifacts`
- `hic-samples`

루트 디렉터리:

```text
schema/schema.sql
data/samples
data/expected
docs
폐기
```

## 10. 감사 체크리스트

- 모집공고가 아닌 게시글이 DB에 저장되었는가?
- 핵심 필드가 raw/source JSON에만 있고 정규화 컬럼에 없는가?
- 원본 파일 저장이 object storage 포트를 우회하거나 특정 인프라 구현에 강결합되어 있는가?
- 상호전환 계산값이 원문 근거 없이 저장되었는가?
- 공급일정이 구조화 테이블이 아니라 본문 텍스트에만 있는가?
- source span 없는 LLM 결과가 저장되었는가?
- 같은 입력 재실행으로 중복 공고/첨부/주택이 생기는가?

위 항목 중 하나라도 참이면 QA gate를 통과할 수 없다.
