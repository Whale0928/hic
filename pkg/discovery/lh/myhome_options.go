package lh

import "strings"

func MyHomePagesToFetch(totalCount int, numRows int, allPages bool, requestedPages int) int {
	if requestedPages <= 0 {
		requestedPages = 1
	}
	if !allPages {
		return requestedPages
	}
	if numRows <= 0 || totalCount <= 0 {
		return requestedPages
	}
	pages := (totalCount + numRows - 1) / numRows
	if pages < 1 {
		return 1
	}
	return pages
}

func FilterMyHomeItemsByAgency(items []MyHomeNoticeItem, agencyFilter string) []MyHomeNoticeItem {
	agencyFilter = strings.TrimSpace(agencyFilter)
	if agencyFilter == "" {
		return items
	}
	out := make([]MyHomeNoticeItem, 0, len(items))
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Agency), agencyFilter) {
			out = append(out, item)
		}
	}
	return out
}
