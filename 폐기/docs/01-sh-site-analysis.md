# SH 데이터 수집 대상 분석

작성일: 2026-05-13

## 대상 사이트

- SH 인터넷청약시스템: `https://www.i-sh.co.kr/app/index.do`
- 주택임대 공고 목록: `/app/lay2/program/S48T1581C563/www/brd/m_247/list.do?multi_itm_seq=2`
- 주택임대 상세: `/app/lay2/program/S48T1581C563/www/brd/m_247/view.do`
- 진행 중 공고 JSON: `/houseinfo/map/selectNoticeInProgressList.do`
- 단지/주택정보 JSON: `/houseinfo/map/selectHouseRentWithFilter.do`, `/houseinfo/map/selectHouseBscDetail.do`, `/houseinfo/map/selectSplyTyInfo.do`

## 확인한 수집 방식

주택임대 목록은 공개 HTML 테이블로 제공된다. 목록 HTML의 실제 구조는 `div#listTb table tbody tr`이며, 상세 공고 번호는 `href`가 아니라 `onclick="getDetailView('304295')"` 형태에 들어간다.

상세 페이지는 일반 폼 POST 방식이다. 필요한 주요 파라미터는 `page`, `seq`, `multi_itm_seq`, `multi_itm_seqsStr`, `srchTp`, `srchWord`다. `view.do`는 AJAX 요청이 아니라 브라우저 폼 제출에 가깝게 호출하는 편이 안정적이었다.

첨부 메타데이터는 상세 HTML의 `initParam.downList` JavaScript 배열에 들어간다. 실제 키는 `brdId`, `seq`, `fileSeq`, `fileSize`, `oriFileNm`, `fileTp`다.

첨부 원본 다운로드 URL은 SH `sh_innorix.js` 분석 결과 다음 패턴으로 확인됐다.

```text
/app/com/file/innoFD.do?brdId={brdId}&seq={seq}&fileTp={fileTp}&fileSeq={fileSeq}
```

첨부 미리보기 URL은 상세 HTML에 다음 형태로 제공된다.

```text
/app/com/util/htmlConverter.do?brd_id={brdId}&seq={seq}&data_tp=A&file_seq={fileSeq}
```

## 실제 수집 검증

2026-05-13 기준 주택임대 1페이지 수집 결과:

- 목록 행: 10건
- 상세 공고: 10건
- 첨부 메타: 38건
- 첨부 원본 다운로드: 38건
- 첨부 확장자: PDF 37건, HWP 1건
- XLSX 행 추출: 0건
- 호수 후보: 0건

이번 1페이지에는 XLSX 첨부가 없어서 호수 단위 데이터는 생성되지 않았다. 다만 XLSX가 발견되면 스트리밍으로 전체 행을 저장하고, 헤더 기반으로 `housing_units` 후보를 생성하도록 구현했다.

## 장기 확장 포인트

LH까지 확장할 때는 사이트별 어댑터만 추가하고, 원본 저장소와 정규화 저장소는 공유한다. SH/LH 모두 첨부 문서 구조가 핵심이므로, 공고 HTML보다 첨부 수집/추출 파이프라인을 공통 계층으로 유지하는 편이 좋다.
