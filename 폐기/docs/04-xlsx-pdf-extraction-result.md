# XLSX/PDF 추출 검증 결과

작성일: 2026-05-13

## 목적

SH 공고 첨부에서 개별 호수 단위 데이터를 실제로 추출해 `housing_units`에 저장할 수 있는지 검증했다. 특히 사용자가 강조한 “개별 호수는 첨부파일에 있고, 엑셀이 클 수 있다”는 요구를 기준으로 XLSX 스트리밍 처리를 확인했다.

## 수행 내용

먼저 주택임대 공고 20페이지를 다운로드 없이 스캔했다.

```text
pages=20 list_rows=200 details=200 attachments=728 previews=0 downloads=0 xlsx_rows=0 units=0 errors=0
```

그 결과 XLSX 첨부 5건을 찾았다.

- `296598`, `2025년 하반기 청년 매입임대주택 우선공급`, 주택목록 XLSX 1건
- `296353`, `2025년 1차 청년 매입임대주택`, 당첨자/예비자/커트라인/주택목록 XLSX 4건

이후 두 공고만 대상으로 첨부 원본 다운로드와 추출을 수행했다.

```text
pages=20 list_rows=200 details=2 attachments=17 previews=0 downloads=17 xlsx_rows=2702 units=776 pdf_texts=8 pdf_failed=0 unsupported=3 errors=0
```

## 저장 결과

DB 카운트:

- `source_notices`: 200건
- `attachments`: 728건
- `attachment_extractions`: 2751건
- `housing_units`: 776건
- `raw_documents`: 472건

호수 후보 품질:

- 전체 호수 후보: 776건
- 호수 번호 존재: 776건
- 단지명 존재: 776건
- 전용면적 존재: 776건
- 보증금 존재: 776건
- 월임대료 존재: 776건

XLSX별 호수 후보:

- `296353 / file_seq 57 / 주택목록`: 751건
- `296598 / file_seq 2 / 주택목록`: 25건
- `296353 / file_seq 54 / 당첨자 명단`: 0건
- `296353 / file_seq 55 / 예비자 명단`: 0건
- `296353 / file_seq 56 / 커트라인`: 0건

당첨자 명단은 `접수번호`, `성명`, `생월일` 같은 개인정보성 헤더를 감지해 `housing_units` 생성 대상에서 제외했다.

## API 검증

Echo API 검증:

- `GET /health`: `{"status":"ok"}`
- `GET /units?limit=2`: 단지명, 호수, 주택형, 전용면적, 보증금, 월임대료, 원본 행 JSON 응답 확인
- `GET /notices?limit=1`: 공고 본문/분류 응답 확인

## 한계와 다음 작업

PDF plain text 추출은 8건 성공했다. HWP 3건은 현재 로컬 추출 도구가 없어 `unsupported` 상태로 기록했다.

다음 작업은 PDF/HWP 표를 더 정교하게 구조화하는 것이다. 이 단계에서는 LLM을 “원본 텍스트/표 후보를 JSON 스키마로 보정하는 후처리기”로 붙이는 방식이 적합하다.
