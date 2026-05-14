# 집 모아

House Information Collector(HIC)는 공공주택 **모집공고 계열**만 발견하고, 첨부 원본을 보존한 뒤, 개별 주택 정보를 정규화하는 수집/정제 시스템입니다.

## 현재 기준

- 수집 대상은 모집공고, 정정공고, 추가모집, 잔여세대 모집뿐입니다.
- 비모집 게시글은 DB 저장 대상이 아닙니다.
- 구현 중심은 루트 `main.go`와 `pkg/` 도메인 패키지입니다.
- PostgreSQL 스키마 산출물은 `schema/schema.sql`입니다.
- CLI는 루트에서 `go run .`로 실행합니다.
- 로컬 기본 포트는 `9551~9559` 범위만 사용합니다.
  - PostgreSQL: `localhost:9551`
  - HIC API: `localhost:9552`

## 핵심 문서

- [AI 작업 가이드](./AGENTS.md)
- [다음 세션 핸드오프](./docs/AI_HANDOFF.md)
- [구현 백로그](./docs/IMPLEMENTATION_BACKLOG.md)
- [용어 사전](./docs/GLOSSARY.md)
- [아키텍처 재설계](./docs/architecture-redesign.md)
- [아키텍처 구성도](./docs/architecture-layers.svg)

## 예정 레이어

```text
pkg/global
pkg/discovery
pkg/extraction
pkg/normalize
pkg/llm
pkg/workflow
pkg/qa
```
