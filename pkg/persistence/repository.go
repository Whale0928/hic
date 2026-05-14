package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"hic/pkg/discovery"
	"hic/pkg/extraction"
	"hic/pkg/normalize"
	"hic/pkg/persistence/db"
	"hic/pkg/workflow"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

type PersistedAttachment struct {
	NoticeID       int64
	StoredObjectID int64
	AttachmentID   int64
	FileSeq        string
	ObjectKey      string
	Kind           extraction.AttachmentKind
}

type HousingUnitView struct {
	ID              int64    `json:"id"`
	Agency          string   `json:"agency"`
	Source          string   `json:"source"`
	SupplyCategory  string   `json:"supply_category"`
	ListNo          string   `json:"list_no"`
	District        string   `json:"district"`
	Address         string   `json:"address"`
	HousingName     string   `json:"housing_name"`
	ComplexName     string   `json:"complex_name"`
	BuildingName    string   `json:"building_name"`
	UnitNo          string   `json:"unit_no"`
	UnitType        string   `json:"unit_type"`
	ExclusiveAreaM2 *float64 `json:"exclusive_area_m2,omitempty"`
	DepositText     string   `json:"deposit_text"`
	DepositKRW      *int64   `json:"deposit_krw,omitempty"`
	MonthlyRentText string   `json:"monthly_rent_text"`
	MonthlyRentKRW  *int64   `json:"monthly_rent_krw,omitempty"`
	SourceSheet     string   `json:"source_sheet"`
	SourceRow       *int32   `json:"source_row,omitempty"`
	SourceSpan      string   `json:"source_span"`
	QAStatus        string   `json:"qa_status"`
}

type SourceNoticeView struct {
	ID         int64  `json:"id"`
	Agency     string `json:"agency"`
	BoardKind  string `json:"board_kind"`
	Seq        string `json:"seq"`
	NoticeType string `json:"notice_type"`
	Title      string `json:"title"`
	PostedAt   string `json:"posted_at"`
	SourceURL  string `json:"source_url"`
}

type CollectionRunStatus string

const (
	CollectionRunStatusSucceeded CollectionRunStatus = "succeeded"
	CollectionRunStatusFailed    CollectionRunStatus = "failed"
)

func Open(ctx context.Context, databaseURL string) (*Repository, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Repository{pool: pool, queries: db.New(pool)}, nil
}

func (r *Repository) Close() {
	r.pool.Close()
}

func (r *Repository) CreateCollectionRun(ctx context.Context, source string) (int64, error) {
	return r.queries.CreateCollectionRun(ctx, source)
}

func (r *Repository) FinishCollectionRun(ctx context.Context, runID int64, status CollectionRunStatus, stats map[string]any, errorText string) error {
	return r.queries.FinishCollectionRun(ctx, db.FinishCollectionRunParams{
		ID:        runID,
		Status:    string(status),
		Stats:     mustJSONAny(stats),
		ErrorText: stringValue(errorText),
	})
}

func ValidatePersistableCandidate(candidate discovery.Candidate) error {
	category := discovery.ClassifyNotice(candidate.Title, "")
	if category != discovery.NoticeCategoryRecruitment {
		return fmt.Errorf("non-recruitment notice must not be persisted: agency=%s board=%s seq=%s category=%s title=%q",
			candidate.Agency,
			candidate.BoardKind,
			candidate.Seq,
			category,
			candidate.Title,
		)
	}
	return nil
}

func (r *Repository) SaveCandidatePreservation(ctx context.Context, board discovery.Board, candidate discovery.Candidate, report workflow.PreserveReport) ([]PersistedAttachment, error) {
	if err := ValidatePersistableCandidate(candidate); err != nil {
		return nil, err
	}

	boardID, err := r.queries.UpsertSourceBoard(ctx, db.UpsertSourceBoardParams{
		Agency:    board.Agency,
		BoardKind: board.BoardKind,
		Name:      board.Name,
		SourceUrl: board.BaseURL + board.ListPath,
	})
	if err != nil {
		return nil, err
	}

	noticeID, err := r.queries.UpsertSourceNotice(ctx, db.UpsertSourceNoticeParams{
		SourceBoardID: int8Value(boardID),
		Agency:        candidate.Agency,
		BoardKind:     candidate.BoardKind,
		Seq:           candidate.Seq,
		Category:      string(discovery.NoticeCategoryRecruitment),
		NoticeType:    "recruitment",
		NoticeSubtype: "",
		Title:         candidate.Title,
		PostedAt:      dateValue(candidate.PostedAt),
		SourceUrl:     board.BaseURL + board.ViewPath,
		BodyText:      "",
	})
	if err != nil {
		return nil, err
	}

	byFileSeq := make(map[string]discovery.AttachmentMeta, len(candidate.Attachments))
	for _, attachment := range candidate.Attachments {
		byFileSeq[attachment.FileSeq] = attachment
	}

	persisted := make([]PersistedAttachment, 0, len(report.Objects))
	for _, object := range report.Objects {
		storedID, err := r.queries.UpsertStoredObject(ctx, db.UpsertStoredObjectParams{
			Bucket:           "hic-originals",
			ObjectKey:        object.StoredObject.Key,
			StorageBackend:   "local_filesystem",
			ContentType:      object.StoredObject.ContentType,
			OriginalFilename: object.StoredObject.OriginalName,
			Sha256:           object.StoredObject.SHA256,
			SizeBytes:        object.StoredObject.SizeBytes,
			Metadata:         mustJSON(object.StoredObject.Metadata),
		})
		if err != nil {
			return nil, err
		}

		attachment := byFileSeq[object.FileSeq]
		attachmentID, err := r.queries.UpsertAttachment(ctx, db.UpsertAttachmentParams{
			NoticeID:         int8Value(noticeID),
			StoredObjectID:   int8Value(storedID),
			BrdID:            attachment.BRDID,
			Seq:              firstNonEmpty(attachment.Seq, candidate.Seq),
			FileSeq:          object.FileSeq,
			OriginalFilename: object.Filename,
			FileExt:          strings.TrimPrefix(strings.ToLower(filepath.Ext(object.Filename)), "."),
			FileSize:         object.StoredObject.SizeBytes,
			ContentType:      object.StoredObject.ContentType,
			ObjectKey:        object.StoredObject.Key,
			Sha256:           object.StoredObject.SHA256,
			AttachmentKind:   string(object.Kind),
			ExtractorStatus:  "preserved",
			RawMetadata: mustJSON(map[string]string{
				"brd_id":          attachment.BRDID,
				"seq":             firstNonEmpty(attachment.Seq, candidate.Seq),
				"file_seq":        object.FileSeq,
				"attachment_kind": string(object.Kind),
			}),
		})
		if err != nil {
			return nil, err
		}
		persisted = append(persisted, PersistedAttachment{
			NoticeID:       noticeID,
			StoredObjectID: storedID,
			AttachmentID:   attachmentID,
			FileSeq:        object.FileSeq,
			ObjectKey:      object.StoredObject.Key,
			Kind:           object.Kind,
		})
	}

	return persisted, nil
}

func (r *Repository) InsertArtifact(ctx context.Context, attachmentID int64, storedObjectID int64, artifact extraction.ExtractedArtifact) (int64, error) {
	return r.queries.InsertExtractedArtifact(ctx, db.InsertExtractedArtifactParams{
		AttachmentID:   int8Value(attachmentID),
		StoredObjectID: int8Value(storedObjectID),
		ArtifactType:   string(artifact.Type),
		Extractor:      artifact.Extractor,
		Status:         string(artifact.Status),
		SchemaVersion:  artifact.SchemaVersion,
		SheetName:      artifact.SourceSheet,
		PageNo:         int4Value(artifact.SourcePage),
		RowNo:          int4Value(artifact.SourceRow),
		CellRef:        artifact.SourceCell,
		RawText:        artifact.RawText,
		ContentJson:    mustJSONAny(artifact.Content),
		SourceSpan:     artifact.SourceSpan,
		Confidence:     numericValue(artifact.Confidence),
		ErrorText:      artifact.ErrorText,
	})
}

func (r *Repository) UpsertHousingUnit(ctx context.Context, attachment PersistedAttachment, sourceArtifactID int64, unit normalize.HousingUnitCandidate) (int64, error) {
	return r.queries.UpsertHousingUnit(ctx, db.UpsertHousingUnitParams{
		NoticeID:         int8Value(attachment.NoticeID),
		AttachmentID:     int8Value(attachment.AttachmentID),
		SourceArtifactID: int8Value(sourceArtifactID),
		Agency:           "SH",
		Source:           "sh",
		SupplyCategory:   unit.SupplyCategory,
		ListNo:           unit.ListNo,
		District:         unit.District,
		Address:          unit.Address,
		LegalDong:        unit.LegalDong,
		AddressDetail:    unit.AddressDetail,
		HousingName:      unit.HousingName,
		ComplexName:      unit.ComplexName,
		BuildingName:     unit.BuildingName,
		UnitNo:           unit.UnitNo,
		Floor:            int4PtrValue(unit.FloorNo),
		FloorNo:          int4PtrValue(unit.FloorNo),
		UnitType:         unit.UnitType,
		StructureType:    unit.StructureType,
		ExclusiveAreaM2:  numericPtrValue(unit.ExclusiveAreaM2),
		AreaPyeong:       numericPtrValue(unit.AreaPyeong),
		DepositText:      unit.DepositText,
		DepositKrw:       int8PtrValue(unit.DepositKRW),
		MonthlyRentText:  unit.MonthlyRentText,
		MonthlyRentKrw:   int8PtrValue(unit.MonthlyRentKRW),
		SupplyCount:      int4PtrValue(unit.SupplyCount),
		Direction:        unit.Direction,
		Status:           unit.Status,
		SourceSheet:      unit.SourceSheet,
		SourceRow:        int4Value(unit.SourceRow),
		SourceCell:       unit.SourceCell,
		SourcePage:       int4Value(unit.SourcePage),
		SourceSpan:       unit.SourceSpan,
		RawRow:           mustJSONAny(unit.RawRow),
		Confidence:       numericValue(unit.Confidence),
		QaStatus:         "pending",
	})
}

func (r *Repository) Counts(ctx context.Context) (storedObjects int64, artifacts int64, err error) {
	storedObjects, err = r.queries.CountStoredObjects(ctx)
	if err != nil {
		return 0, 0, err
	}
	artifacts, err = r.queries.CountExtractedArtifacts(ctx)
	return storedObjects, artifacts, err
}

func (r *Repository) CountHousingUnits(ctx context.Context) (int64, error) {
	return r.queries.CountHousingUnits(ctx)
}

func (r *Repository) ExistingNoticeSeqs(ctx context.Context, agency string, boardKind string) (map[string]bool, error) {
	seqs, err := r.queries.ListExistingNoticeSeqs(ctx, db.ListExistingNoticeSeqsParams{
		Agency:    agency,
		BoardKind: boardKind,
	})
	if err != nil {
		return nil, err
	}
	known := make(map[string]bool, len(seqs))
	for _, seq := range seqs {
		seq = strings.TrimSpace(seq)
		if seq != "" {
			known[seq] = true
		}
	}
	return known, nil
}

func (r *Repository) ListHousingUnits(ctx context.Context, limit int32) ([]HousingUnitView, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := r.queries.ListHousingUnits(ctx, limit)
	if err != nil {
		return nil, err
	}
	units := make([]HousingUnitView, 0, len(rows))
	for _, row := range rows {
		units = append(units, HousingUnitView{
			ID:              row.ID,
			Agency:          row.Agency,
			Source:          row.Source,
			SupplyCategory:  row.SupplyCategory,
			ListNo:          row.ListNo,
			District:        row.District,
			Address:         row.Address,
			HousingName:     row.HousingName,
			ComplexName:     row.ComplexName,
			BuildingName:    row.BuildingName,
			UnitNo:          row.UnitNo,
			UnitType:        row.UnitType,
			ExclusiveAreaM2: numericToFloat64Ptr(row.ExclusiveAreaM2),
			DepositText:     row.DepositText,
			DepositKRW:      int8ToInt64Ptr(row.DepositKrw),
			MonthlyRentText: row.MonthlyRentText,
			MonthlyRentKRW:  int8ToInt64Ptr(row.MonthlyRentKrw),
			SourceSheet:     row.SourceSheet,
			SourceRow:       int4ToInt32Ptr(row.SourceRow),
			SourceSpan:      row.SourceSpan,
			QAStatus:        row.QaStatus,
		})
	}
	return units, nil
}

func (r *Repository) ListSourceNotices(ctx context.Context, limit int32) ([]SourceNoticeView, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := r.queries.ListSourceNotices(ctx, limit)
	if err != nil {
		return nil, err
	}
	notices := make([]SourceNoticeView, 0, len(rows))
	for _, row := range rows {
		notices = append(notices, SourceNoticeView{
			ID:         row.ID,
			Agency:     row.Agency,
			BoardKind:  row.BoardKind,
			Seq:        row.Seq,
			NoticeType: row.NoticeType,
			Title:      row.Title,
			PostedAt:   dateToString(row.PostedAt),
			SourceURL:  row.SourceUrl,
		})
	}
	return notices, nil
}

func int8Value(value int64) pgtype.Int8 {
	return pgtype.Int8{Int64: value, Valid: true}
}

func int4Value(value int) pgtype.Int4 {
	if value == 0 {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(value), Valid: true}
}

func int4PtrValue(value *int) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*value), Valid: true}
}

func int8PtrValue(value *int64) pgtype.Int8 {
	if value == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *value, Valid: true}
}

func int8ToInt64Ptr(value pgtype.Int8) *int64 {
	if !value.Valid {
		return nil
	}
	return &value.Int64
}

func int4ToInt32Ptr(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	return &value.Int32
}

func dateValue(value time.Time) pgtype.Date {
	if value.IsZero() {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: value, Valid: true}
}

func numericPtrValue(value *float64) pgtype.Numeric {
	if value == nil {
		return pgtype.Numeric{}
	}
	return numericValue(*value)
}

func numericToFloat64Ptr(value pgtype.Numeric) *float64 {
	f8, err := value.Float64Value()
	if err != nil || !f8.Valid {
		return nil
	}
	return &f8.Float64
}

func dateToString(value pgtype.Date) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format(time.DateOnly)
}

func numericValue(value float64) pgtype.Numeric {
	var numeric pgtype.Numeric
	if err := numeric.Scan(fmt.Sprintf("%f", value)); err != nil {
		return pgtype.Numeric{}
	}
	return numeric
}

func mustJSON(value map[string]string) []byte {
	if value == nil {
		return []byte("{}")
	}
	out, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return out
}

func mustJSONAny(value map[string]any) []byte {
	if value == nil {
		return []byte("{}")
	}
	out, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return out
}

func stringValue(value string) pgtype.Text {
	if strings.TrimSpace(value) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
