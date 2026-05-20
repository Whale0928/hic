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
