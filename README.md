# 집 모아 / House Information Collector

## Overview

집 모아(House Information Collector, HIC)는 공공주택 모집공고 데이터를 수집하고, 첨부 원본을 보존한 뒤, PDF/XLSX/HWP/HTML에서 주택 정보를 추출해 PostgreSQL에 정규화하는 시스템입니다.

현재 POC는 SH 공공주택 모집공고 게시판을 우선 대상으로 합니다. 장기적으로 LH 등 다른 기관은 같은 파이프라인에 site adapter를 추가하는 방식으로 확장합니다.

핵심 데이터 단위인 `주택`은 건물 자체가 아니라 건물 안의 개별 호실 또는 유닛입니다.

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
- 모집공고/비모집 게시글 분류
- 최근 게시일 기준 cutoff
- 이미 수집한 공고 skip
- 첨부 원본 보존
- PDF 텍스트 추출
- XLSX 주택목록 행 추출
- PDF 단일호실 공급표 일부 정규화
- QA 승인 후 `/offerings` 기본 노출

## Reference

- [AI 작업 가이드](./AGENTS.md)
- [세션 핸드오프](./docs/AI_HANDOFF.md)
- [구현 백로그](./docs/IMPLEMENTATION_BACKLOG.md)
- [용어 사전](./docs/GLOSSARY.md)
- [아키텍처 재설계](./docs/architecture-redesign.md)
- [아키텍처 구성도](./docs/architecture-layers.svg)
- [PostgreSQL 스키마](./schema/schema.sql)
