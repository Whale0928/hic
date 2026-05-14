# House Information Collector 용어 사전

| 한글 | 영어 | 축약어/변수명 | 설명 |
|---|---|---|---|
| 집 모아 | House Information Collector | HIC | 프로젝트 전체 이름. 공공주택 모집공고와 첨부자료를 발견, 추출, 정규화하는 시스템. |
| 기관 | Agency | `agency` | SH, LH처럼 공고를 제공하는 주체. |
| 게시판 | Board | `board` | 기관 사이트 안의 공고 목록 단위. 예: SH 주택임대, SH 주택분양. |
| 게시판 종류 | Board Kind | `board_kind` | 게시판을 코드로 구분하는 값. 예: `rental`, `sale`. |
| 공고 | Notice | `notice` | 기관 게시판에 올라온 게시글. HIC에서는 모집공고 계열만 저장 대상으로 삼는다. |
| 모집공고 | Recruitment Notice | `recruitment_notice`, `notice_type=recruitment` | 실제 신청/청약/입주자 모집과 연결되는 공고. 기본 수집 대상. |
| 정정공고 | Correction Notice | `correction_notice`, `notice_subtype=correction` | 기존 모집공고 내용을 수정하거나 보완하는 공고. 모집공고 계열에 포함한다. |
| 추가모집 | Additional Recruitment | `additional_recruitment` | 기존 공급 이후 추가로 신청자를 모집하는 공고. 모집공고 계열에 포함한다. |
| 잔여세대 모집 | Remaining Unit Recruitment | `remaining_unit_recruitment` | 남은 주택/세대를 대상으로 모집하는 공고. 모집공고 계열에 포함한다. |
| 비모집 게시글 | Non-Recruitment Post | `non_recruitment_post` | 당첨자 발표, 계약안내, 시스템 공지, 일반 알림 등. HIC 기본 정책에서는 수집하지 않는다. |
| 공고번호 | Source Sequence | `source_seq` | 원본 사이트에서 사용하는 게시글 식별자. SH 예: `296598`. |
| 원본 URL | Source URL | `source_url` | 공고 목록, 상세, 첨부 다운로드 등 원본 사이트 URL. |
| 첨부파일 | Attachment | `attachment` | 공고에 연결된 PDF, XLSX, HWP, HTML 미리보기 등 파일. |
| 첨부 순번 | File Sequence | `file_seq` | 원본 사이트에서 첨부파일을 구분하는 번호. |
| 원본 객체 | Stored Object | `stored_object`, `object` | object storage 인터페이스에 저장한 원본 파일 또는 추출 산출물. 기본 구현은 로컬 저장소, 확장 구현은 MinIO 같은 S3 호환 저장소다. |
| 객체 키 | Object Key | `object_key` | 저장소 구현과 무관하게 원본 파일 또는 산출물을 찾는 논리 경로. |
| 해시 | SHA-256 Checksum | `sha256` | 파일 중복 방지와 무결성 검증을 위한 해시값. |
| 발견 | Discovery | `discovery` | 게시판에서 모집공고와 첨부 메타를 식별하는 단계. 비모집 게시글은 이 단계에서 폐기한다. |
| 추출 | Extraction | `extraction` | PDF/XLSX/HWP/HTML에서 텍스트, 페이지, 행, 표 후보를 기계적으로 뽑는 단계. |
| 추출 산출물 | Extracted Artifact | `artifact` | 추출 단계가 만든 텍스트, 행 JSON, 페이지 텍스트, 표 후보 등. |
| 정규화 | Normalize | `normalize`, `normalization` | 추출 산출물을 주소, 주택, 가격, 일정, 상호전환 등 도메인 필드로 변환하는 단계. |
| LLM 보정 | LLM Assist | `llm_assist` | deterministic parser가 실패하거나 confidence가 낮은 구간을 LLM으로 보정하는 단계. |
| 워크플로우 | Workflow | `workflow` | discovery → extraction → normalize → llm assist → QA gate를 조율하는 실행 흐름. |
| 품질 검증 | Quality Assurance | `qa` | 샘플 기반 회귀검증, 필드 누락률, source span, 중복 저장 여부를 검증하는 단계. |
| 단지 | Complex | `complex`, `complex_name` | 하나 이상의 건물 또는 주택을 묶는 공급/관리 단위. SH 엑셀에서는 주택명과 겹칠 수 있다. |
| 건물 | Building | `building`, `building_name` | 물리적인 건축물 또는 동. 하나의 건물 안에 여러 주택/호실이 들어갈 수 있다. |
| 주택 | Housing Unit | `housing_unit`, `unit` | HIC의 핵심 개별 정보 단위. 건물 안의 각 호실 하나를 주택으로 표현한다. |
| 호실 | Unit Number | `unit_no` | 주택을 건물 안에서 식별하는 번호. 예: `0404`. |
| 층 | Floor Number | `floor_no` | 주택이 위치한 층. 호실에서 추정할 수도 있고 별도 컬럼으로 올 수도 있다. |
| 주택명 | Housing Name | `housing_name` | 엑셀 원본의 주택명. 예: `Oaktreevil DR2`. 단지명과 동일하게 쓰일 수 있다. |
| 동/건물명 | Building Name | `building_name` | 동, 건물명, 주건축물명 등 건물 식별값. 원본에 없으면 비워둘 수 있다. |
| 주소 | Address | `address` | 주택의 전체 소재지 주소. raw가 아니라 필수 정규화 필드다. |
| 자치구 | District | `district` | 서울시 구 단위 행정구역. 예: 영등포구. |
| 법정동 | Legal Dong | `legal_dong` | 주소에서 추출 가능한 법정동 후보. 예: 대림동. |
| 상세주소 | Address Detail | `address_detail` | 번지, 건물명, 괄호 안 주소 등 상세 주소 정보. |
| 공급구분 | Supply Category | `supply_category` | 신규공급, 재공급 등 공급 상태/유형. |
| 목록번호 | List Number | `list_no` | 첨부 주택목록에서 사용하는 행 번호. |
| 주택형 | Unit Type | `unit_type` | 27A, 40A 등 면적/구조/타입을 나타내는 원본 값. |
| 주택구조 | Structure Type | `structure_type` | 개방형원룸, 분리형원룸 등 구조 유형. |
| 엘리베이터 설치 여부 | Elevator Installed | `elevator_installed` | 엘리베이터 설치 여부를 boolean으로 정규화한 값. |
| 전용면적 | Exclusive Area | `exclusive_area_m2` | 전용면적 제곱미터 값. |
| 평수 | Area Pyeong | `area_pyeong` | 전용면적을 평 단위로 환산한 값. 필요 시 계산 필드로 둔다. |
| 임대보증금 | Deposit | `deposit_krw` | 원 단위 임대보증금. 문자열이 아니라 숫자 필드로 저장한다. |
| 월임대료 | Monthly Rent | `monthly_rent_krw` | 원 단위 월임대료. 문자열이 아니라 숫자 필드로 저장한다. |
| 상호전환제도 | Rent Conversion Rule | `rent_conversion_rule` | 월임대료를 보증금으로, 또는 보증금을 월임대료로 전환하는 규칙. |
| 월임대료 보증금 전환 가능 여부 | Rent-to-Deposit Allowed | `rent_to_deposit_allowed` | 월임대료 일부를 보증금으로 전환할 수 있는지 여부. |
| 월임대료 보증금 전환 최대비율 | Rent-to-Deposit Max Ratio | `rent_to_deposit_max_ratio` | 월임대료 중 최대 몇 %를 보증금으로 전환 가능한지. 예: `0.60`. |
| 월임대료 보증금 전환요율 | Rent-to-Deposit Annual Rate | `rent_to_deposit_annual_rate` | 보증금 전환에 적용되는 연 전환요율. 예: `0.067`. |
| 임대료 단위금액 | Rent Unit Amount | `rent_unit_krw` | 예시 계산 기준이 되는 월임대료 단위. 예: `10000`. |
| 단위임대료당 추가보증금 | Deposit Per Rent Unit | `deposit_per_rent_unit_krw` | 임대료 단위금액당 추가 납부해야 하는 보증금. 예: `1800000`. |
| 보증금 월임대료 전환 가능 여부 | Deposit-to-Rent Allowed | `deposit_to_rent_allowed` | 보증금을 낮추고 월임대료를 올릴 수 있는지 여부. |
| 최대 전환 가능 월임대료 | Max Convertible Rent | `max_convertible_rent_krw` | 주택별 월임대료와 전환 최대비율로 계산한 전환 가능 월임대료. |
| 예상 추가보증금 | Estimated Additional Deposit | `estimated_additional_deposit_krw` | 최대 전환 시 추가 납부할 보증금 계산값. |
| 전환 후 최저 월임대료 | Estimated Minimum Monthly Rent | `estimated_min_monthly_rent_krw` | 최대 전환 후 남는 월임대료 계산값. |
| 공급일정 | Notice Schedule | `notice_schedule` | 청약, 서류제출, 발표, 계약, 입주 등 공고 진행 일정. |
| 일정 종류 | Schedule Type | `schedule_type` | `application`, `document_submission`, `winner_announcement`, `contract`, `move_in` 등. |
| 시작일시 | Start At | `starts_at` | 일정 시작 날짜/시간. |
| 종료일시 | End At | `ends_at` | 일정 종료 날짜/시간. |
| 원문 날짜 | Date Text | `date_text` | 파싱 실패나 검증을 위해 보존하는 원문 날짜 표현. |
| 신청자격 | Eligibility Rule | `eligibility_rule` | 무주택, 소득, 자산, 연령, 세대구성 등 신청 조건. |
| 신뢰도 | Confidence | `confidence` | 필드 추출/정규화 결과의 신뢰도. |
| 원본 위치 | Source Span | `source_span` | PDF 페이지, XLSX sheet/row/cell, HTML selector 등 필드 근거 위치. |
| 소스 행 | Source Row | `source_row` | XLSX 원본 행 JSON. 조회 기준이 아니라 감사/재처리용 보조 증거. |
| QA 게이트 | QA Gate | `qa_gate` | 필수 필드와 품질 기준을 통과해야 serving/API로 승격하는 검증 관문. |
| 서비스 승격 | Serving Promotion | `serving_promotion` | 검증을 통과한 정규화 데이터를 API/화면에서 조회 가능하게 만드는 단계. |
