package discovery

import "testing"

func TestClassifyNotice_모집공고계열만통과시킨다(t *testing.T) {
	tests := []struct {
		name  string
		title string
		body  string
		want  NoticeCategory
	}{
		{name: "모집공고", title: "청년안심주택 입주자 모집공고", want: NoticeCategoryRecruitment},
		{name: "정정공고", title: "행복주택 입주자 모집 정정공고", want: NoticeCategoryRecruitment},
		{name: "추가모집", title: "국민임대주택 예비입주자 추가모집 공고", want: NoticeCategoryRecruitment},
		{name: "띄어쓴추가모집", title: "[토지임대부 사회주택] 함께주택6호 입주자 추가 모집 공고문", want: NoticeCategoryRecruitment},
		{name: "잔여세대", title: "두레주택 잔여세대 입주자 모집공고", want: NoticeCategoryRecruitment},
		{name: "당첨자", title: "서류심사대상자 발표 안내", want: NoticeCategoryRejected},
		{name: "띄어쓴서류심사대상자", title: "제49차 장기전세주택 서류심사 대상자 발표 및 서류제출 안내", want: NoticeCategoryRejected},
		{name: "공급대상자발표", title: "희망하우징 입주자 모집공고 예비1차 공급대상자 발표", want: NoticeCategoryRejected},
		{name: "입주대상자발표", title: "기존주택 전세임대 입주자 모집공고 예비1차 입주대상자 발표", want: NoticeCategoryRejected},
		{name: "최종심사대상자발표", title: "만리동 예술인 협동조합주택 최종심사 대상자 발표", want: NoticeCategoryRejected},
		{name: "최종접수현황", title: "청년 매입임대주택 우선공급 입주자 모집공고 최종접수현황 게시", want: NoticeCategoryRejected},
		{name: "동호선정", title: "신혼·신생아 매입임대주택 입주대기자 모집공고 동호선정 대상주택 안내", want: NoticeCategoryRejected},
		{name: "계약안내", title: "행복주택 계약체결 안내", want: NoticeCategoryRejected},
		{name: "청약경쟁률", title: "청년안심주택 입주자 모집공고 최종 청약경쟁률 게시", want: NoticeCategoryRejected},
		{name: "접수결과", title: "잔여세대 입주자모집공고 선순위 청약접수결과 및 접수마감 안내", want: NoticeCategoryRejected},
		{name: "시스템공지", title: "인터넷청약시스템 서비스 일시중단 안내", want: NoticeCategoryRejected},
		{name: "일반알림", title: "주택임대 알림", want: NoticeCategoryUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyNotice(tt.title, tt.body); got != tt.want {
				t.Fatalf("ClassifyNotice() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSHRegistry_임대게시판을제공한다(t *testing.T) {
	registry := NewStaticSiteRegistry()

	board, ok := registry.Get("SH", "rental")
	if !ok {
		t.Fatalf("SH rental board not found")
	}
	if board.Agency != "SH" || board.BoardKind != "rental" || board.ListPath == "" {
		t.Fatalf("board = %+v", board)
	}
}
