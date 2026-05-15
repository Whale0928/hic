package discovery

import (
	"strings"
	"testing"
	"time"
)

func TestParseBoardList_주택임대목록에서공고행을추출한다(t *testing.T) {
	html := `
	<table id="listTb">
	<tbody>
	<tr>
	  <td>1613</td>
	  <td class="txtL"><a href="javascript:getDetailView('304295');">2026년 휘경마을 두레주택 잔여세대 입주자 모집공고(26. 5. 13.) <span class="icoNew">NEW</span></a></td>
	  <td>맞춤주택공급부</td>
	  <td>2026-05-13</td>
	  <td>60</td>
	</tr>
	</tbody>
	</table>`

	rows, err := ParseBoardList(strings.NewReader(html), Board{Agency: "SH", BoardKind: "rental"})
	if err != nil {
		t.Fatalf("ParseBoardList() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}

	got := rows[0]
	if got.Seq != "304295" {
		t.Fatalf("Seq = %q, want 304295", got.Seq)
	}
	if got.Title != "2026년 휘경마을 두레주택 잔여세대 입주자 모집공고(26. 5. 13.)" {
		t.Fatalf("Title = %q", got.Title)
	}
	if got.Department != "맞춤주택공급부" {
		t.Fatalf("Department = %q", got.Department)
	}
	if !got.IsNew {
		t.Fatalf("IsNew = false, want true")
	}
	if got.PostedAt.Format(time.DateOnly) != "2026-05-13" {
		t.Fatalf("PostedAt = %s", got.PostedAt.Format(time.DateOnly))
	}
	if got.ViewCount != 60 {
		t.Fatalf("ViewCount = %d, want 60", got.ViewCount)
	}
}

func TestParseBoardDetail_첨부파일과본문을추출한다(t *testing.T) {
	html := `
	<script>
	var initParam = {};
	initParam.downList = [{"brdId":"GS0401","seq":"304295","fileSeq":"1","filename":"[공고문] 테스트.pdf","size":"339384"}];
	</script>
	<table>
	  <tbody>
	    <tr><th>제목</th><td>2026년 휘경마을 두레주택 잔여세대 입주자 모집공고</td></tr>
	    <tr><th>작성일</th><td>2026-05-13</td></tr>
	    <tr><th>조회수</th><td>60</td></tr>
	    <tr><td class="cont">공급호수 : 2호<br>접수일 : 2026. 5. 26.(화) ~ 5. 28.(목)</td></tr>
	  </tbody>
	</table>
	<a href="/app/com/util/htmlConverter.do?brd_id=GS0401&seq=304295&data_tp=A&file_seq=1">미리보기</a>`

	detail, err := ParseBoardDetail(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseBoardDetail() error = %v", err)
	}

	if detail.Title != "2026년 휘경마을 두레주택 잔여세대 입주자 모집공고" {
		t.Fatalf("Title = %q", detail.Title)
	}
	if detail.BodyText != "공급호수 : 2호 접수일 : 2026. 5. 26.(화) ~ 5. 28.(목)" {
		t.Fatalf("BodyText = %q", detail.BodyText)
	}
	if len(detail.Attachments) != 1 {
		t.Fatalf("len(Attachments) = %d, want 1", len(detail.Attachments))
	}
	att := detail.Attachments[0]
	if att.FileSeq != "1" || att.DisplayFilename() != "[공고문] 테스트.pdf" || att.PreviewPath == "" {
		t.Fatalf("Attachment = %+v", att)
	}
}

func TestParseBoardDetail_캡션형상세화면에서제목과작성일을추출한다(t *testing.T) {
	html := `
	<div class="detailTable gs0401Table">
	<table>
	  <caption>2026년 위국헌신청년주택 입주자 모집공고(2026. 5. 15.)</caption>
	  <thead>
	    <tr><th scope="col" colspan="2">2026년 위국헌신청년주택 입주자 모집공고(2026. 5. 15.)</th></tr>
	  </thead>
	  <tbody>
	    <tr>
	      <td colspan="2">
	        <ul>
	          <li><strong>등록일 : </strong>2026-05-15</li>
	          <li><strong>조회수 : </strong>567</li>
	        </ul>
	      </td>
	    </tr>
	    <tr><td class="cont">공급대상 있음</td></tr>
	  </tbody>
	</table>
	</div>`

	detail, err := ParseBoardDetail(strings.NewReader(html))
	if err != nil {
		t.Fatalf("ParseBoardDetail() error = %v", err)
	}

	if detail.Title != "2026년 위국헌신청년주택 입주자 모집공고(2026. 5. 15.)" {
		t.Fatalf("Title = %q", detail.Title)
	}
	if detail.PostedAt == nil || detail.PostedAt.Format(time.DateOnly) != "2026-05-15" {
		t.Fatalf("PostedAt = %#v", detail.PostedAt)
	}
	if detail.ViewCount != 567 {
		t.Fatalf("ViewCount = %d, want 567", detail.ViewCount)
	}
}
