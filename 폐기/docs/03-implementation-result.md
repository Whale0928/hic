# 1차 구현 및 검증 결과

작성일: 2026-05-13

## 구현 범위

- Go 모듈 `shdata` 생성
- SH 주택임대 목록 HTML 파서
- SH 상세 HTML 파서
- 첨부 `initParam.downList` 파서
- 공고/결과/시스템 알림 분류기
- XLSX 스트리밍 행 추출기
- XLSX 헤더 기반 호수 후보 추정기
- PostgreSQL 스키마 및 저장소
- SH 실제 수집 CLI
- Echo API `GET /health`, `GET /notices`, `GET /units`
- Docker Compose PostgreSQL

## 검증 명령

```bash
go test ./...
docker compose up -d postgres
DATABASE_URL='postgres://shdata:shdata@localhost:55432/shdata?sslmode=disable' go run ./cmd/hic migrate
DATABASE_URL='postgres://shdata:shdata@localhost:55432/shdata?sslmode=disable' go run ./cmd/hic collect-sh --pages 1 --store-previews=false --download-attachments=true
```

## 검증 결과

`go test ./...` 결과:

- `internal/extract`: 통과
- `internal/sh`: 통과
- 나머지 패키지: 컴파일 통과

실제 SH 1페이지 수집 결과:

```text
pages=1 list_rows=10 details=10 attachments=38 previews=0 downloads=38 xlsx_rows=0 units=0 errors=0
```

DB 저장 건수:

- `source_notices`: 10건
- `attachments`: 38건
- `attachment_extractions`: 38건
- `raw_documents`: 54건 이상
- `housing_units`: 0건

첨부 원본은 `.data/downloads/SH/{seq}/...` 경로에 저장된다.

## 확인된 한계

이후 20페이지 스캔에서 XLSX 첨부 공고를 찾아 실제 호수 후보 776건을 `housing_units`에 저장했다. 상세 결과는 `04-xlsx-pdf-extraction-result.md`에 기록했다.

PDF 원본은 다운로드 후 plain text 추출까지 저장한다. HWP/HWPX는 현재 추출기가 없어 `unsupported` 상태로 기록한다. PDF/HWP 표를 직접 구조화하는 영역은 LLM 또는 별도 표 추출기를 붙이는 다음 작업 단위로 분리하는 것이 좋다.
