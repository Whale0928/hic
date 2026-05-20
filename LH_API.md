# LH 모집공고 데이터 - 마이홈 OpenAPI 활용 가이드

## 개요

LH(한국토지주택공사) 및 지방공기업(인천iH, 부산도시공사, 경북개발 등)의 공공임대주택 모집공고는 **국토교통부 마이홈포털 공공주택 모집공고 조회 서비스 OpenAPI**로 조회한다.

- 데이터셋: data.go.kr/data/**15108420**
- 제공기관: 국토교통부 청년주거정책과
- 갱신주기: **실시간**
- 비용: 무료
- 자동승인 (개발 + 운영)
- 트래픽 한도: 개발계정 **1,000건/일**

**SH 데이터는 포함되지 않는다.** SH 모집공고는 i-sh.co.kr 게시판 스크래핑이 별도로 필요하다 (HIC의 기존 discovery 경로 유지).

---

## 활용신청

1. https://www.data.go.kr 로그인
2. https://www.data.go.kr/data/15108420/openapi.do 진입
3. `활용신청` 클릭 → 활용목적 입력 → 신청 (자동승인, 즉시 처리)
4. 마이페이지 → 데이터활용 → Open API → 개발계정 → 상세보기에서 **일반 인증키** 확인 (hex 64자)

---

## 인증키 보관

`.env` 파일에 추가한다.

```
MYHOME_API_KEY=e76c7e27dc19ad1c1b33d8006dda77a7a3147d3af6606161970295f269b452f8
```

마이페이지의 hex 키를 그대로 사용. URL 인코딩 불필요.

활용기간: 2026-05-19 ~ 2028-05-19 (개발계정, 자동승인)

> 처음에 표시되는 Encoding/Decoding 형식의 base64 키와 다른 키이므로 혼동 주의. 호출 결과 401이 떨어지면 hex 키를 다시 확인할 것.

---

## 호출 형식

**Base URL**: `https://apis.data.go.kr/1613000/HWSPR02`

### 엔드포인트

| Path | 설명 |
|---|---|
| `GET /rsdtRcritNtcList` | 공공임대주택 모집공고 (주력) |
| `GET /ltRsdtRcritNtcList` | 공공분양주택 모집공고 |

### 파라미터

| 이름 | 필수 | 값 |
|---|---|---|
| `serviceKey` | 필수 | 일반 인증키 hex |
| `pageNo` | 필수 | 1부터 |
| `numOfRows` | 권장 | 1~200 (perPage는 무시됨, GW 표준은 numOfRows) |

선택 파라미터(광역시도/시군구/공급유형/주택유형/전세형여부/월임대료/모집공고월): 코드는 활용신청 페이지 첨부 `붙임1. 요청 파라미터 코드(공공주택 모집공고)_260331.xlsx` 참고. **단, 광역시도 코드 필터는 현재 무시되고 전국이 반환됨** — 클라이언트에서 `brtcNm` 필드로 필터링해야 한다.

### 예시 호출

```bash
curl "https://apis.data.go.kr/1613000/HWSPR02/rsdtRcritNtcList?serviceKey=${MYHOME_API_KEY}&pageNo=1&numOfRows=200"
```

---

## 응답 구조

```json
{
  "response": {
    "header": { "resultCode": "00", "resultMsg": "NORMAL SERVICE" },
    "body": {
      "totalCount": "425",
      "numOfRows": "200",
      "pageNo": "1",
      "item": [
        {
          "pblancId": "20364",
          "houseSn": 1,
          "sttusNm": "일반공고",
          "pblancNm": "함평기산 통합공공임대주택 입주자모집(2026.05.19.공고)",
          "suplyInsttNm": "LH",
          "houseTyNm": "아파트",
          "suplyTyNm": "통합공공임대",
          "beforePblancId": "",
          "rcritPblancDe": "20260519",
          "przwnerPresnatnDe": "20260908",
          "suplyHoCo": "",
          "refrnc": "LH 콜센터 : 1600-1004",
          "url": "https://apply.lh.or.kr/lhapply/apply/wt/wrtanc/selectWrtancInfo.do?...",
          "pcUrl": "https://www.myhome.go.kr/hws/portal/sch/selectRsdtRcritNtcDetailView.do?pblancId=20364&houseSn=1",
          "mobileUrl": "https://m.myhome.go.kr/hws/portal/sch/selectRsdtRcritNtcDetailView.do?pblancId=20364&houseSn=1",
          "hsmpNm": "함평기산",
          "brtcNm": "전라남도",
          "signguNm": "함평군",
          "fullAdres": "전라남도 함평군 함평읍 기각리 775-1",
          "refrnLegaldongNm": "함평읍",
          "pnu": "4686025022107750001",
          "heatMthdNm": "개별난방",
          "totHshldCo": "",
          "sumSuplyCo": 60,
          "rentGtn": 7303000,
          "enty": 365150,
          "prtpay": 0,
          "surlus": 6937850,
          "mtRntchrg": 109640,
          "beginDe": "20260605",
          "endDe": "20260609"
        }
      ]
    }
  }
}
```

### HIC 스키마 매핑 (제안)

| 마이홈 필드 | HIC 매핑 |
|---|---|
| `pblancId` | `source_notices.seq` (PK) |
| `pblancNm` | `source_notices.title` |
| `suplyInsttNm` | `source_notices.agency` ("LH", "인천도시공사" 등) |
| `suplyTyNm` | `offerings.supply_category` (통합공공임대/국민임대/행복주택/전세임대/매입임대) |
| `houseTyNm` | `offerings.unit_type` |
| `rcritPblancDe` | `source_notices.posted_at` (YYYYMMDD 파싱) |
| `beginDe` / `endDe` | `notice_schedules.starts_at` / `ends_at` (schedule_type='application') |
| `brtcNm` / `signguNm` | `offerings.district` |
| `fullAdres` | `offerings.address` |
| `pnu` | (참고) 부동산 PNU |
| `hsmpNm` | `offerings.housing_name` |
| `sumSuplyCo` | `offerings.supply_count` |
| `rentGtn` | `offerings.deposit_krw` |
| `mtRntchrg` | `offerings.monthly_rent_krw` |
| `url` | `source_notices.source_url` (원 기관 사이트) |
| `pcUrl` | `source_notices.detail_url` (마이홈 자체 상세) |

**핵심 이점**: `beginDe`/`endDe`가 직접 필드로 제공되므로 **PDF 파싱 없이 신청기간 확정 가능**. SH 게시판 스크래핑처럼 `notice_schedules` placeholder 문제가 없다.

---

## 2026-05-20 HIC dry-run 검증 통계

마이홈 API는 실시간 갱신이므로 아래 수치는 실행 시점에 따라 변한다. 현재 HIC 수집기는 `suplyInsttNm` 기준으로 LH만 클라이언트 필터링한다.

```bash
go run . workflow collect-lh --kind rental --num-rows 200 --all-pages --dry-run=true --agency-filter LH
```

- 공공임대 모집공고: `pages=3`, `total_count=418`, `raw_items=418`, `LH items=397`

```bash
go run . workflow collect-lh --kind sale --num-rows 200 --all-pages --dry-run=true --agency-filter LH
```

- 공공분양 모집공고: `pages=1`, `total_count=52`, `raw_items=52`, `LH items=51`

SH 매입임대·사회주택·잔여세대·든든주택은 마이홈 응답에 포함되지 않으므로 SH 게시판 스크래핑 경로를 계속 유지한다.

---

## 알려진 한계

1. **SH 미포함** - i-sh.co.kr 스크래핑 병행 필수
2. **광역시도 코드 필터 무시** - 클라이언트 측에서 `brtcNm` 필드로 필터링
3. **`perPage` 무시** - GW 표준 `numOfRows` 사용
4. 응답이 표준 data.go.kr XML 에러 코드가 아닌 평문 401/403일 때가 있음 (게이트웨이 분산 환경의 일시적 비일관성). 재시도로 회복

---

## 보조 OpenAPI

| 데이터셋 | URL | 용도 |
|---|---|---|
| 15110581 | https://www.data.go.kr/data/15110581/openapi.do | 마이홈 공공임대주택 단지정보 (마스터, 우선 보조 경로) |
| 15108378 | https://www.data.go.kr/data/15108378/openapi.do | 마이홈 예비입주자 대기현황 |
| 15058476 | https://www.data.go.kr/data/15058476/openapi.do | LH 공공임대주택 단지정보 (기존/보조. 공공데이터포털 안내상 인증키 오류 개선 후 15110581로 제공 중) |

---

## 참고

- 마이홈 OpenAPI 페이지: https://www.data.go.kr/data/15108420/openapi.do
- 활용신청 페이지 첨부: `붙임1. 요청 파라미터 코드(공공주택 모집공고)_260331.xlsx` (광역시도/시군구/공급유형/주택유형 코드 정의)
- data.go.kr 게이트웨이 호출 가이드: https://www.data.go.kr/images/biz/swagger-guide/gw/gateway_swagger_guide.pdf
- 첨부 PDF 본문은 응답에 포함되지 않으며, 필요 시 `url` 또는 `pcUrl` 에서 별도 fetch 후 PDF 추출
