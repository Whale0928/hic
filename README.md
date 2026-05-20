# 집 모아 / House Information Collector

## Overview

집 모아(House Information Collector, HIC)는 공공주택 모집공고 데이터를 수집하고, 첨부 원본을 보존한 뒤, PDF/XLSX/HWP/HTML에서 주택 정보를 추출해 PostgreSQL에 정규화하는 시스템입니다.

현재 구현은 SH 게시판 수집과 LH/MyHome OpenAPI 수집을 분리된 경로로 실행하되, 최종 결과는 같은 `offerings` / `notice_schedules` / QA 모델로 수렴시킵니다.

핵심 데이터 단위는 `공급항목(Offering)`입니다. 개별 호실이 신청 단위일 수도 있고, 단지/면적/유형/성별/공급호수 조합이 신청 단위일 수도 있으므로 `unit_no`는 nullable입니다.

## What to do

수집 대상은 모집공고 계열만 허용합니다.

- 모집공고
- 정정공고
- 추가모집
- 잔여세대 모집

다음 게시글은 수집하지 않고 DB에도 저장하지 않습니다.

- 당첨자 발표
- 서류심사 대상자 발표
- 계약안내
- 청약 결과/경쟁률
- 시스템 공지
- 일반 알림

로컬 실행 예시는 다음과 같습니다.

```bash
docker compose up -d postgres
go run . migrate
go run . workflow collect-sh --board rental --pages 3 --dry-run=true
go run . workflow collect-sh --board rental --pages 3 --dry-run=false --preserve-attachments
go run . workflow collect-lh --kind rental --num-rows 200 --all-pages --dry-run=true --agency-filter LH
go run . workflow collect-lh --kind rental --num-rows 200 --all-pages --dry-run=false --agency-filter LH
go run . llm candidates --limit 20
go run . llm repair --artifact-id 123 --dry-run=false --max-attempts 1500
go run . qa promote-offerings
go run . serve
```

기본 포트는 프로젝트 내에서 `9551~9559` 범위를 사용합니다.

- PostgreSQL: `localhost:9551`
- HIC API: `localhost:9552`

## Other

기술 스택은 Go, Echo, Cobra, PostgreSQL, sqlc를 기준으로 합니다.

주요 패키지 방향은 다음과 같습니다.

```text
pkg/global
pkg/discovery
pkg/extraction
pkg/normalize
pkg/llm
pkg/workflow
pkg/qa
```

Redis와 MinIO는 선택 지원 인프라입니다. 핵심 도메인 로직은 Redis나 MinIO에 강하게 의존하지 않아야 하며, 기본 ObjectStore는 로컬 파일시스템입니다.

현재 POC에서 확인된 동작입니다.

- SH 모집공고 목록 discovery
- LH/MyHome 공공임대/공공분양 OpenAPI 수집
- 모집공고/비모집 게시글 분류
- 최근 게시일 기준 cutoff
- 이미 수집한 공고 skip
- 첨부 원본과 HTML 미리보기 보존
- PDF/HWPX/HTML 텍스트 추출
- HWP 외부 도구 기반 텍스트 추출, 도구 미설치 시 `hwp_unsupported` artifact 보존
- XLSX 주택목록 행 추출
- PDF 단일호실 공급표 일부 정규화
- LH/MyHome 신청기간, 공급기관, 공급유형, 주소, 단지명, 공급호수, 보증금, 월임대료 정규화
- source span은 serving 승인 기준에서 `object://` 또는 `myhome://`만 허용
- LLM 보정 후보 조회와 schema-constrained Offering 보정
- QA 승인 후 `/offerings` 기본 노출
- approved notice의 `/schedules` 기본 노출

## Reference

- [AI 작업 가이드](./AGENTS.md)
- [세션 핸드오프](./docs/AI_HANDOFF.md)
- [구현 백로그](./docs/IMPLEMENTATION_BACKLOG.md)
- [용어 사전](./docs/GLOSSARY.md)
- [아키텍처 재설계](./docs/architecture-redesign.md)
- [아키텍처 구성도](./docs/architecture-layers.svg)
- [PostgreSQL 스키마](./schema/schema.sql)
