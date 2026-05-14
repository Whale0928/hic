package discovery

import "strings"

type NoticeCategory string

const (
	NoticeCategoryRecruitment NoticeCategory = "recruitment"
	NoticeCategoryRejected    NoticeCategory = "rejected"
	NoticeCategoryUnknown     NoticeCategory = "unknown"
)

func ClassifyNotice(title string, body string) NoticeCategory {
	switch {
	case containsAny(title, "당첨자", "서류심사대상자", "동호배정", "결과"):
		return NoticeCategoryRejected
	case containsAny(title, "청약경쟁률", "경쟁률", "접수결과", "접수마감"):
		return NoticeCategoryRejected
	case containsAny(title, "계약", "계약체결"):
		return NoticeCategoryRejected
	case containsAny(title, "서비스 일시중단", "시스템", "인증서", "점검", "장애"):
		return NoticeCategoryRejected
	case containsAny(body, "서비스 일시중단", "시스템 점검"):
		return NoticeCategoryRejected
	case containsAny(title, "모집공고", "입주자 모집", "추가모집", "잔여세대", "공급공고", "정정공고"):
		return NoticeCategoryRecruitment
	case containsAny(body, "입주자 모집", "공급대상"):
		return NoticeCategoryRecruitment
	default:
		return NoticeCategoryUnknown
	}
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
