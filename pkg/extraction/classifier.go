package extraction

import (
	"path/filepath"
	"strings"
)

func ClassifyAttachment(filename string) AttachmentKind {
	name := strings.ToLower(strings.TrimSpace(filename))
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")

	if containsAny(name, "당첨자", "예비자", "서류심사대상자", "접수자", "명단", "결과") {
		return AttachmentKindApplicantOrWinnerFile
	}
	if containsAny(name, "신청서", "신청양식", "서식", "개인정보", "동의서") {
		return AttachmentKindApplicationForm
	}
	if ext == "xlsx" || ext == "xlsm" {
		if containsAny(name, "주택목록", "공급대상", "동호", "호실") {
			return AttachmentKindOfferingListXLSX
		}
		return AttachmentKindUnsupported
	}
	if ext == "pdf" {
		if containsAny(name, "일정", "스케줄") {
			return AttachmentKindSchedulePDF
		}
		if containsAny(name, "공고", "모집", "공급", "팸플릿", "주택") {
			return AttachmentKindNoticePDF
		}
		return AttachmentKindUnsupported
	}
	if ext == "hwp" || ext == "hwpx" {
		if containsAny(name, "공고", "모집") {
			return AttachmentKindNoticeHWP
		}
		return AttachmentKindUnsupported
	}

	return AttachmentKindUnsupported
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}
