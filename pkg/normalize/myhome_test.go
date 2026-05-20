package normalize

import (
	"testing"
	"time"

	lhdiscovery "hic/pkg/discovery/lh"
)

func TestOfferingFromMyHomeItem_신청단위와일정을정규화한다(t *testing.T) {
	count := 60
	deposit := int64(7303000)
	rent := int64(109640)
	item := lhdiscovery.MyHomeNoticeItem{
		NoticeID:       "20364",
		HouseSN:        1,
		Title:          "함평기산 통합공공임대주택 입주자모집",
		Agency:         "LH",
		HouseType:      "아파트",
		SupplyType:     "통합공공임대",
		PostedDate:     "20260519",
		ApplicationBeg: "20260605",
		ApplicationEnd: "20260609",
		ComplexName:    "함평기산",
		Province:       "전라남도",
		City:           "함평군",
		Address:        "전라남도 함평군 함평읍 기각리 775-1",
		LegalDong:      "함평읍",
		SupplyCount:    &count,
		DepositKRW:     &deposit,
		MonthlyRent:    &rent,
		DetailURL:      "https://www.myhome.go.kr/hws/portal/sch/selectRsdtRcritNtcDetailView.do?pblancId=20364&houseSn=1",
		SourceURL:      "https://apply.lh.or.kr/lhapply/apply/wt/wrtanc/selectWrtancInfo.do",
	}

	offering := OfferingFromMyHomeItem(item, "object://myhome/rsdtRcritNtcList/20364-1.json")
	if offering.UnitNo != "" {
		t.Fatalf("UnitNo = %q, want empty because MyHome item is a grouped application unit", offering.UnitNo)
	}
	if offering.ApplicationUnitLabel != "함평기산 통합공공임대 아파트 60호" {
		t.Fatalf("ApplicationUnitLabel = %q", offering.ApplicationUnitLabel)
	}
	if offering.District != "전라남도 함평군" || offering.Address != item.Address || offering.LegalDong != "함평읍" {
		t.Fatalf("address fields = %+v", offering)
	}
	if offering.DepositKRW == nil || *offering.DepositKRW != 7303000 ||
		offering.MonthlyRentKRW == nil || *offering.MonthlyRentKRW != 109640 {
		t.Fatalf("money fields = %+v", offering)
	}

	schedule, ok := ApplicationScheduleFromMyHomeItem(item, 7, "object://myhome/rsdtRcritNtcList/20364-1.json")
	if !ok {
		t.Fatalf("ApplicationScheduleFromMyHomeItem() ok = false")
	}
	if schedule.NoticeID != 7 || schedule.ScheduleType != "application" {
		t.Fatalf("schedule = %+v", schedule)
	}
	if !schedule.StartsAt.Equal(time.Date(2026, 6, 5, 0, 0, 0, 0, time.Local)) ||
		!schedule.EndsAt.Equal(time.Date(2026, 6, 9, 23, 59, 59, 0, time.Local)) {
		t.Fatalf("schedule times = %+v", schedule)
	}
}

func TestOfferingFromMyHomeItem_공공분양납부금액을QA가능필드로정규화한다(t *testing.T) {
	count := 317
	contract := int64(46654000)
	interim := int64(93308000)
	balance := int64(271578000)
	item := lhdiscovery.MyHomeNoticeItem{
		NoticeID:           "1411",
		HouseSN:            1,
		Title:              "[정정공고]인천계양 A9블록 신혼희망타운(공공분양) 입주자모집공고",
		Agency:             "LH",
		HouseType:          "아파트",
		ComplexName:        "인천계양 A9블록",
		SupplyCount:        &count,
		ContractPaymentKRW: &contract,
		InterimPaymentKRW:  &interim,
		BalancePaymentKRW:  &balance,
	}

	offering := OfferingFromMyHomeItem(item, "myhome://ltRsdtRcritNtcList/1411/1")

	if offering.ContractDepositKRW == nil || *offering.ContractDepositKRW != 46654000 {
		t.Fatalf("ContractDepositKRW = %v", offering.ContractDepositKRW)
	}
	wantBalance := int64(364886000)
	if offering.BalancePaymentKRW == nil || *offering.BalancePaymentKRW != wantBalance {
		t.Fatalf("BalancePaymentKRW = %v, want %d", offering.BalancePaymentKRW, wantBalance)
	}
	if offering.RawRow["interim_payment_krw"] != interim {
		t.Fatalf("RawRow = %+v", offering.RawRow)
	}
}

func TestOfferingFromMyHomeItem_공급호수없는MyHome행은HouseSN을목록번호로보존한다(t *testing.T) {
	zeroCount := 0
	item := lhdiscovery.MyHomeNoticeItem{
		NoticeID:    "1341",
		HouseSN:     11,
		Agency:      "LH",
		HouseType:   "다가구주택",
		ComplexName: "우아연립",
		SupplyCount: &zeroCount,
	}

	offering := OfferingFromMyHomeItem(item, "myhome://ltRsdtRcritNtcList/1341/11")

	if offering.ListNo != "11" {
		t.Fatalf("ListNo = %q, want 11", offering.ListNo)
	}
	if offering.ApplicationUnitLabel != "우아연립 다가구주택" {
		t.Fatalf("ApplicationUnitLabel = %q", offering.ApplicationUnitLabel)
	}
	if offering.SupplyCount == nil || *offering.SupplyCount != 0 {
		t.Fatalf("SupplyCount = %v, want preserved zero", offering.SupplyCount)
	}
}
