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
		{name: "잔여세대", title: "두레주택 잔여세대 입주자 모집공고", want: NoticeCategoryRecruitment},
		{name: "당첨자", title: "서류심사대상자 발표 안내", want: NoticeCategoryRejected},
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
