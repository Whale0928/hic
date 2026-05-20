package lh

import "testing"

func TestParseMyHomeNoticeDetailHTML_전세임대조건을구조화한다(t *testing.T) {
	html := `
<table>
  <tr><th>공급호수</th><td>5700호</td></tr>
  <tr><th>대상 주택 정보</th><td>전용면적&nbsp;85㎡&nbsp;이하&nbsp;전세&nbsp;또는&nbsp;보증부월세로&nbsp;계약&nbsp;가능한&nbsp;주택</td></tr>
  <tr><th>지원한도액</th><td>수도권&nbsp;14,500만원,&nbsp;광역시(세종시포함)&nbsp;11,000만원,&nbsp;기타&nbsp;도지역&nbsp;9,500만원</td></tr>
  <tr><th>임대조건</th><td>-&nbsp;임대보증금&nbsp;:&nbsp;지원한도액&nbsp;범위&nbsp;내&nbsp;전세보증금의&nbsp;5%<br />-&nbsp;월임대료&nbsp;:&nbsp;LH&nbsp;지원금의&nbsp;연&nbsp;1.2~2.2%이자<br />-&nbsp;상세내역은&nbsp;입주자모집&nbsp;공고문&nbsp;참조</td></tr>
</table>`

	detail, err := ParseMyHomeNoticeDetailHTML(html)
	if err != nil {
		t.Fatalf("ParseMyHomeNoticeDetailHTML() error = %v", err)
	}

	if detail.SupplyCount == nil || *detail.SupplyCount != 5700 {
		t.Fatalf("SupplyCount = %v, want 5700", detail.SupplyCount)
	}
	if detail.SupportLimitText != "수도권 14,500만원, 광역시(세종시포함) 11,000만원, 기타 도지역 9,500만원" {
		t.Fatalf("SupportLimitText = %q", detail.SupportLimitText)
	}
	if detail.DepositConditionText != "임대보증금 : 지원한도액 범위 내 전세보증금의 5%" {
		t.Fatalf("DepositConditionText = %q", detail.DepositConditionText)
	}
	if detail.MonthlyRentConditionText != "월임대료 : LH 지원금의 연 1.2~2.2%이자" {
		t.Fatalf("MonthlyRentConditionText = %q", detail.MonthlyRentConditionText)
	}
}

func TestParseMyHomeNoticeDetailHTML_공고문참조만있는임대조건은금액근거로쓰지않는다(t *testing.T) {
	html := `
<table>
  <tr><th>공급호수</th><td>11호</td></tr>
  <tr><th>임대조건</th><td>자세한 사항은 모집 공고문 확인 바랍니다.</td></tr>
</table>`

	detail, err := ParseMyHomeNoticeDetailHTML(html)
	if err != nil {
		t.Fatalf("ParseMyHomeNoticeDetailHTML() error = %v", err)
	}

	if detail.RentConditionText == "" {
		t.Fatalf("RentConditionText should preserve audit text")
	}
	if detail.DepositConditionText != "" || detail.MonthlyRentConditionText != "" {
		t.Fatalf("placeholder detail should not become money evidence: %+v", detail)
	}
}

func TestParseMyHomeNoticeDetailHTML_매입임대단일임대조건을양쪽금액텍스트로보존한다(t *testing.T) {
	html := `
<table>
  <tr><th>임대조건</th><td>시중&nbsp;전세가격의&nbsp;90%&nbsp;수준의&nbsp;임대보증금&nbsp;및&nbsp;임대료</td></tr>
</table>`

	detail, err := ParseMyHomeNoticeDetailHTML(html)
	if err != nil {
		t.Fatalf("ParseMyHomeNoticeDetailHTML() error = %v", err)
	}

	if detail.DepositConditionText != "시중 전세가격의 90% 수준의 임대보증금 및 임대료" {
		t.Fatalf("DepositConditionText = %q", detail.DepositConditionText)
	}
	if detail.MonthlyRentConditionText != "시중 전세가격의 90% 수준의 임대보증금 및 임대료" {
		t.Fatalf("MonthlyRentConditionText = %q", detail.MonthlyRentConditionText)
	}
}

func TestParseMyHomeNoticeDetailHTML_매입임대상세임대조건을분리한다(t *testing.T) {
	html := `
<table>
  <tr><th>임대조건</th><td>시중&nbsp;전세가격의&nbsp;90%&nbsp;수준의&nbsp;임대보증금&nbsp;및&nbsp;임대료<br />-&nbsp;임대보증금&nbsp;:&nbsp;해당주택&nbsp;매입금액의&nbsp;50%수준<br />-&nbsp;임대료&nbsp;:&nbsp;시중&nbsp;전세가격의&nbsp;90%수준에서&nbsp;임대보증금을&nbsp;제외한&nbsp;임대료</td></tr>
</table>`

	detail, err := ParseMyHomeNoticeDetailHTML(html)
	if err != nil {
		t.Fatalf("ParseMyHomeNoticeDetailHTML() error = %v", err)
	}

	if detail.DepositConditionText != "임대보증금 : 해당주택 매입금액의 50%수준" {
		t.Fatalf("DepositConditionText = %q", detail.DepositConditionText)
	}
	if detail.MonthlyRentConditionText != "임대료 : 시중 전세가격의 90%수준에서 임대보증금을 제외한 임대료" {
		t.Fatalf("MonthlyRentConditionText = %q", detail.MonthlyRentConditionText)
	}
}
