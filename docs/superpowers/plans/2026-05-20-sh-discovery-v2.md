# SH Discovery v2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build SH Discovery v2 so SH internet application status pages and SH board notices are discovered separately, then reconciled so active notices such as 든든주택 are not missed.

**Architecture:** Keep common board primitives in `pkg/discovery`, and place SH-specific application status and reconciliation code under `pkg/discovery/sh`. `collect-sh` gains an active-application mode that first reads SH appUser rows, then scans the rental board deeply enough to find matching original notices. DB-backed tests clean the database per test with `TRUNCATE ... RESTART IDENTITY CASCADE`.

**Tech Stack:** Go, Cobra, goquery, PostgreSQL, sqlc-style query files, Docker PostgreSQL for DB validation.

---

### File Structure

- Create: `pkg/discovery/sh/application.go`
  - SH appUser endpoint registry, appUser fetcher, HTML parser, status model.
- Create: `pkg/discovery/sh/application_test.go`
  - RED/GREEN tests for `splyTy=12` 든든주택 parsing and empty status pages.
- Create: `pkg/discovery/sh/reconcile.go`
  - Title/date normalization and appUser-to-board candidate reconciliation.
- Create: `pkg/discovery/sh/reconcile_test.go`
  - RED/GREEN tests proving page-4 board row can be matched to an active appUser row.
- Modify: `pkg/discovery/discoverer.go`
  - Add optional target-title scan behavior while preserving existing cutoff behavior when no targets exist.
- Modify: `pkg/discovery/discoverer_test.go`
  - Add RED/GREEN test that cutoff does not stop before active targets are found.
- Modify: `pkg/cli/root.go`
  - Add `discovery sh-applications` and `workflow collect-sh --active-applications --active-sply-ty --active-max-pages`.
- Modify: `pkg/cli/root_test.go`
  - Assert new commands/options are exposed and report status fields.
- Modify: `schema/schema.sql`
  - Add `application_notices` table for SH appUser status rows.
- Modify: `pkg/persistence/queries/poc.sql`
  - Add `UpsertApplicationNotice` and `LinkApplicationNoticeToSourceNotice`.
- Modify: `pkg/persistence/repository.go`
  - Add repository methods to save application notices and link source notices.
- Modify: `pkg/persistence/repository_test.go`
  - Add DB integration helper and isolated DB tests for application notice upsert/linking.
- Phase 2 design doc update after Phase 1 validation:
  - Modify `docs/architecture-redesign.md` or add `docs/discovery-v2-phase2.md`.

### Task 1: SH AppUser Parser

**Files:**
- Create: `pkg/discovery/sh/application_test.go`
- Create: `pkg/discovery/sh/application.go`

- [ ] **Step 1: Write the failing parser test**

```go
func TestParseApplicationList_든든주택청약중행을파싱한다(t *testing.T) {
	html := `<table><tbody><tr>
	<td>1</td>
	<td><a onclick="userSsnCheck('202620092','','12', '32', 'N', '', '2026년 전세임대형 든든주택 입주자 모집 공고(2026.4.29.)')">2026년 전세임대형 든든주택 입주자 모집 공고(2026.4.29.)</a></td>
	<td>500</td><td>2026-04-29</td>
	<td><a class="btn btnGreen">청약중</a></td>
	</tr></tbody></table>`

	rows, err := ParseApplicationList(strings.NewReader(html), ApplicationEndpoint{SupplyType: "12"})

	if err != nil {
		t.Fatalf("ParseApplicationList() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.RecruitNoticeCode != "202620092" || row.SupplyType != "12" || row.RecruitType != "32" {
		t.Fatalf("codes = %+v", row)
	}
	if row.Status != StatusOpen || row.SupplyCount == nil || *row.SupplyCount != 500 {
		t.Fatalf("status/count = %+v", row)
	}
}
```

- [ ] **Step 2: Run RED**

Run: `go test ./pkg/discovery/sh -run TestParseApplicationList_든든주택청약중행을파싱한다 -count=1`

Expected: FAIL because package or function does not exist.

- [ ] **Step 3: Implement minimal parser**

Add `ApplicationEndpoint`, `ApplicationNotice`, `StatusOpen`, `StatusPending`, and `ParseApplicationList`.

- [ ] **Step 4: Run GREEN**

Run: `go test ./pkg/discovery/sh -run TestParseApplicationList_든든주택청약중행을파싱한다 -count=1`

Expected: PASS.

### Task 2: SH Application Endpoint Registry and Fetcher

**Files:**
- Modify: `pkg/discovery/sh/application.go`
- Modify: `pkg/discovery/sh/application_test.go`

- [ ] **Step 1: Write registry and fetcher tests**

```go
func TestDefaultApplicationEndpoints_전세임대12를포함한다(t *testing.T) {
	endpoints := DefaultApplicationEndpoints()
	ep, ok := endpoints.BySupplyType("12")
	if !ok {
		t.Fatal("splyTy=12 endpoint not found")
	}
	if ep.Path == "" || !strings.Contains(ep.Path, "appUser_list.do") {
		t.Fatalf("endpoint = %+v", ep)
	}
}
```

- [ ] **Step 2: Run RED**

Run: `go test ./pkg/discovery/sh -run TestDefaultApplicationEndpoints_전세임대12를포함한다 -count=1`

Expected: FAIL because registry does not exist.

- [ ] **Step 3: Implement registry and HTTP fetcher**

Use SH paths discovered from the live menu, including `splyTy=12` at `/app/lay2/program/S48T572C4932/m_78/appNoti/appUser_list.do`.

- [ ] **Step 4: Run GREEN**

Run: `go test ./pkg/discovery/sh -run TestDefaultApplicationEndpoints_전세임대12를포함한다 -count=1`

Expected: PASS.

### Task 3: Reconcile AppUser Rows to Board Candidates

**Files:**
- Create: `pkg/discovery/sh/reconcile_test.go`
- Create: `pkg/discovery/sh/reconcile.go`

- [ ] **Step 1: Write failing reconcile test**

```go
func TestReconcileApplications_제목날짜로게시판후보를연결한다(t *testing.T) {
	posted := time.Date(2026, 4, 29, 0, 0, 0, 0, time.UTC)
	apps := []ApplicationNotice{{
		RecruitNoticeCode: "202620092",
		SupplyType:        "12",
		Title:             "2026년 전세임대형 든든주택 입주자 모집 공고(2026.4.29.)",
		PostedAt:          posted,
		Status:            StatusOpen,
	}}
	candidates := []discovery.Candidate{{
		Agency: "SH", BoardKind: "rental", Seq: "303584",
		Title: "2026년 전세임대형 든든주택 입주자 모집공고(2026.04.29.)",
		PostedAt: posted,
	}}

	result := ReconcileApplications(apps, candidates)

	if len(result.Linked) != 1 || result.Linked[0].BoardSeq != "303584" {
		t.Fatalf("result = %+v", result)
	}
}
```

- [ ] **Step 2: Run RED**

Run: `go test ./pkg/discovery/sh -run TestReconcileApplications_제목날짜로게시판후보를연결한다 -count=1`

Expected: FAIL because reconcile function does not exist.

- [ ] **Step 3: Implement normalization and matching**

Normalize whitespace, punctuation, date tokens, and `모집 공고` vs `모집공고`.

- [ ] **Step 4: Run GREEN**

Run: `go test ./pkg/discovery/sh -run TestReconcileApplications_제목날짜로게시판후보를연결한다 -count=1`

Expected: PASS.

### Task 4: Target-Aware Board Discovery

**Files:**
- Modify: `pkg/discovery/discoverer.go`
- Modify: `pkg/discovery/discoverer_test.go`

- [ ] **Step 1: Write failing cutoff override test**

```go
func TestDiscoverer_Discover_목표공고가남아있으면컷오프에서멈추지않는다(t *testing.T) {
	fetcher := pagedFakeFetcher{
		listHTMLByPage: map[int]string{
			1: `<table id="listTb"><tbody><tr><td>1</td><td><a href="javascript:getDetailView('100');">오래된 모집공고</a></td><td>공급부</td><td>2026-04-01</td><td>1</td></tr></tbody></table>`,
			2: `<table id="listTb"><tbody><tr><td>1</td><td><a href="javascript:getDetailView('303584');">2026년 전세임대형 든든주택 입주자 모집공고(2026.04.29.)</a></td><td>공급부</td><td>2026-04-29</td><td>1</td></tr></tbody></table>`,
		},
		detailHTMLs: map[string]string{
			"303584": `<table><tr><th>제목</th><td>2026년 전세임대형 든든주택 입주자 모집공고(2026.04.29.)</td></tr><tr><td class="cont">공급대상 있음</td></tr></table>`,
		},
	}
	cutoff := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)

	report, err := NewDiscoverer(fetcher).Discover(context.Background(), Board{Agency: "SH", BoardKind: "rental"}, Options{
		Pages:        2,
		CutoffDate:   cutoff,
		TargetTitles: []string{"2026년 전세임대형 든든주택 입주자 모집공고"},
	})

	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if report.Pages != 2 || len(report.Candidates) != 1 || report.Candidates[0].Seq != "303584" {
		t.Fatalf("report = %+v", report)
	}
}
```

- [ ] **Step 2: Run RED**

Run: `go test ./pkg/discovery -run TestDiscoverer_Discover_목표공고가남아있으면컷오프에서멈추지않는다 -count=1`

Expected: FAIL because `Options.TargetTitles` does not exist.

- [ ] **Step 3: Implement target-aware scan**

Add `TargetTitles []string` and keep scanning until all targets are matched or max pages is reached. Existing no-target cutoff behavior remains unchanged.

- [ ] **Step 4: Run GREEN**

Run: `go test ./pkg/discovery -run TestDiscoverer_Discover_목표공고가남아있으면컷오프에서멈추지않는다 -count=1`

Expected: PASS.

### Task 5: Persistence for Application Notices

**Files:**
- Modify: `schema/schema.sql`
- Modify: `pkg/persistence/queries/poc.sql`
- Modify: `pkg/persistence/repository.go`
- Modify: `pkg/persistence/repository_test.go`

- [ ] **Step 1: Write isolated DB test**

```go
func TestRepository_ApplicationNotice_테스트별클린DB에서UpsertLink한다(t *testing.T) {
	repo := openCleanTestRepository(t)
	noticeID := saveSourceNoticeFixture(t, repo, "303584", "2026년 전세임대형 든든주택 입주자 모집공고(2026.04.29.)")
	appID := upsertApplicationNoticeFixture(t, repo, "202620092", "12", "청약중")

	if err := repo.LinkApplicationNoticeToSourceNotice(context.Background(), appID, noticeID); err != nil {
		t.Fatalf("LinkApplicationNoticeToSourceNotice() error = %v", err)
	}
	linked := applicationNoticeFixture(t, repo, "202620092")
	if linked.NoticeID == nil || *linked.NoticeID != noticeID {
		t.Fatalf("linked notice id = %+v, want %d", linked.NoticeID, noticeID)
	}
}
```

- [ ] **Step 2: Run RED**

Run: `go test ./pkg/persistence -run TestRepository_ApplicationNotice_테스트별클린DB에서UpsertLink한다 -count=1`

Expected: FAIL because persistence methods/table do not exist.

- [ ] **Step 3: Implement table and repository methods**

Create `application_notices` with unique `(agency, source, sply_ty, recrnoti_cd)`, nullable `notice_id`, title/status/supply_count/raw_metadata fields, and link method.

- [ ] **Step 4: Run GREEN**

Run: `go test ./pkg/persistence -run TestRepository_ApplicationNotice_테스트별클린DB에서UpsertLink한다 -count=1`

Expected: PASS with clean DB before and after the test.

### Task 6: CLI Commands and Workflow Wiring

**Files:**
- Modify: `pkg/cli/root.go`
- Modify: `pkg/cli/root_test.go`

- [ ] **Step 1: Write CLI help tests**

```go
func TestNewRootCommand_SHApplicationsHelp를노출한다(t *testing.T) {
	cmd := NewRootCommand(context.Background())
	cmd.SetArgs([]string{"discovery", "sh-applications", "--help"})
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	out := buf.String()
	for _, want := range []string{"--sply-ty", "--all-active", "--show-items"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help missing %s:\n%s", want, out)
		}
	}
}
```

- [ ] **Step 2: Run RED**

Run: `go test ./pkg/cli -run TestNewRootCommand_SHApplicationsHelp를노출한다 -count=1`

Expected: FAIL because command does not exist.

- [ ] **Step 3: Implement CLI**

Add `discovery sh-applications` dry-run command and `workflow collect-sh --active-applications --active-sply-ty 12 --active-max-pages 10`.

- [ ] **Step 4: Run GREEN**

Run: `go test ./pkg/cli -run TestNewRootCommand_SHApplicationsHelp를노출한다 -count=1`

Expected: PASS.

### Task 7: Verification and Phase 2 Design

**Files:**
- Modify: `docs/architecture-redesign.md` or create `docs/discovery-v2-phase2.md`

- [ ] **Step 1: Run unit tests**

Run: `go test ./...`

Expected: PASS.

- [ ] **Step 2: Run SH application dry-run**

Run: `go run . discovery sh-applications --sply-ty 12 --dry-run=true --show-items`

Expected output includes `application recrnoti_cd=202620092 sply_ty=12 status=청약중 supply_count=500`.

- [ ] **Step 3: Run active board dry-run**

Run: `go run . workflow collect-sh --board rental --dry-run=true --active-applications --active-sply-ty 12 --active-max-pages 10 --max-age-days 20`

Expected output includes `candidate seq=303584` or active reconciliation counts showing 든든주택 linked.

- [ ] **Step 4: Run DB validation**

Run collection against clean local DB or isolated object root, then query via Docker PostgreSQL:

```bash
docker compose exec postgres psql -U shdata -d shdata -Atc "select recrnoti_cd, sply_ty, notice_id from application_notices where recrnoti_cd='202620092';"
```

Expected: one row with non-null `notice_id`.

- [ ] **Step 5: Write Phase 2 common Discovery model note**

Document `DiscoverySource`, `DiscoveredNotice`, `ApplicationStatus`, and agency-specific packages: `pkg/discovery/sh`, `pkg/discovery/lh` or `pkg/discovery/myhome`.
