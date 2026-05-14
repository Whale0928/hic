# 집 모아

## 프로젝트 이름

- 한글명: `집 모아`
- 영문 풀네임: `House Information Collector`
- 약칭: `HIC`
- Go 모듈명: `hic`
- 기동 커맨드 경로: `./cmd/hic`

SH 인터넷청약/주택정보 사이트의 공개 주택 공고, 첨부 메타데이터, 첨부 미리보기 원문, XLSX/PDF 추출 후보 데이터를 PostgreSQL에 보존하는 Go/Echo 기반 수집 실험 프로젝트입니다.

## 빠른 실행

```bash
docker compose up -d postgres
DATABASE_URL='postgres://shdata:shdata@localhost:55432/shdata?sslmode=disable' go run ./cmd/hic migrate
DATABASE_URL='postgres://shdata:shdata@localhost:55432/shdata?sslmode=disable' go run ./cmd/hic collect-sh --pages 1 --store-previews=true --download-attachments=true
DATABASE_URL='postgres://shdata:shdata@localhost:55432/shdata?sslmode=disable' go run ./cmd/hic serve
```

특정 공고만 깊게 재수집할 때:

```bash
DATABASE_URL='postgres://shdata:shdata@localhost:55432/shdata?sslmode=disable' go run ./cmd/hic collect-sh --pages 20 --seq 296598,296353 --store-previews=false --download-attachments=true
```

API:

- `GET /health`
- `GET /notices?limit=100`
- `GET /units?limit=200`

## 설계 원칙

- SH 원문 HTML/JSON은 `raw_documents`에 먼저 저장합니다.
- 파싱된 공고는 `source_notices`, 첨부 메타는 `attachments`에 저장합니다.
- 첨부 미리보기/추출 결과는 `attachment_extractions`에 저장합니다.
- 첨부 원본은 `.data/downloads`에 저장하고, 저장 경로/해시/다운로드 URL은 `attachments`에 남깁니다.
- XLSX 첨부는 스트리밍으로 전체 행을 `attachment_extractions`에 저장하고, 호수 단위 후보 데이터는 `housing_units`에 저장하되 원본 행은 `raw_row` JSONB로 함께 보존합니다.
- PDF 첨부는 가능한 경우 plain text를 `attachment_extractions`에 저장합니다.
- HWP/HWPX는 현재 추출기가 없어 `unsupported` 상태로 기록합니다.
- LLM 정제는 후처리 단계로 붙이고, 원본과 결정론적 파서 결과를 항상 남겨 재처리 가능하게 둡니다.
