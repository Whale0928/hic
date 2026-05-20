package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"hic/pkg/discovery"
	lhdiscovery "hic/pkg/discovery/lh"
	"hic/pkg/extraction"
	"hic/pkg/global"
	"hic/pkg/llm"
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
	NoticeID              int64
	StoredObjectID        int64
	AttachmentID          int64
	PreviewStoredObjectID int64
	FileSeq               string
	Filename              string
	ObjectKey             string
	PreviewObjectKey      string
	Kind                  extraction.AttachmentKind
}

type ApplicationNoticeInput struct {
	NoticeID          int64
	Agency            string
	Source            string
	SupplyType        string
	RecruitNoticeCode string
	RecruitType       string
	NoticeNoHouse     string
	RegionPriority    string
	Title             string
	Status            string
	SupplyCount       *int
	PostedAt          time.Time
	SourceURL         string
	RawMetadata       map[string]any
}

type DiscoverySeenCacheInput struct {
	Agency        string
	BoardKind     string
	Seq           string
	Status        discovery.SeenCacheStatus
	Reason        discovery.NoticeCategory
	Title         string
	PostedAt      time.Time
	ExpiresAt     time.Time
	PolicyVersion string
	ParserVersion string
	Evidence      map[string]any
}

type OfferingView struct {
	ID                   int64    `json:"id"`
	Agency               string   `json:"agency"`
	Source               string   `json:"source"`
	ApplicationUnitLabel string   `json:"application_unit_label"`
	SupplyMethod         string   `json:"supply_method"`
	ApplicationCategory  string   `json:"application_category"`
	SupplyCategory       string   `json:"supply_category"`
	ListNo               string   `json:"list_no"`
	District             string   `json:"district"`
	Address              string   `json:"address"`
	HousingName          string   `json:"housing_name"`
	ComplexName          string   `json:"complex_name"`
	BuildingName         string   `json:"building_name"`
	UnitNo               string   `json:"unit_no"`
	UnitType             string   `json:"unit_type"`
	ExclusiveAreaM2      *float64 `json:"exclusive_area_m2,omitempty"`
	DepositText          string   `json:"deposit_text"`
	DepositKRW           *int64   `json:"deposit_krw,omitempty"`
	JeonseDepositText    string   `json:"jeonse_deposit_text"`
	JeonseDepositKRW     *int64   `json:"jeonse_deposit_krw,omitempty"`
	ContractDepositKRW   *int64   `json:"contract_deposit_krw,omitempty"`
	BalancePaymentKRW    *int64   `json:"balance_payment_krw,omitempty"`
	MonthlyRentText      string   `json:"monthly_rent_text"`
	MonthlyRentKRW       *int64   `json:"monthly_rent_krw,omitempty"`
	SupplyCount          *int32   `json:"supply_count,omitempty"`
	ReservedCount        *int32   `json:"reserved_count,omitempty"`
	GenderRequirement    string   `json:"gender_requirement"`
	OccupancyType        string   `json:"occupancy_type"`
	CapacityPersons      *int32   `json:"capacity_persons,omitempty"`
	DormitoryFeeKRW      *int64   `json:"dormitory_fee_krw,omitempty"`
	HeatingMethod        string   `json:"heating_method"`
	MoveInStartText      string   `json:"move_in_start_text"`
	SourceSheet          string   `json:"source_sheet"`
	SourceRow            *int32   `json:"source_row,omitempty"`
	SourceSpan           string   `json:"source_span"`
	QAStatus             string   `json:"qa_status"`
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

type ScheduleView struct {
	ID           int64  `json:"id"`
	NoticeID     int64  `json:"notice_id"`
	ScheduleType string `json:"schedule_type"`
	Label        string `json:"label"`
	StartsAt     string `json:"starts_at"`
	EndsAt       string `json:"ends_at"`
	DateText     string `json:"date_text"`
	Channel      string `json:"channel"`
	Note         string `json:"note"`
	SourceText   string `json:"source_text"`
	SourceSpan   string `json:"source_span"`
}

type QASummary struct {
	Approved int64
	Rejected int64
	Pending  int64
}

type LLMRepairArtifact struct {
	ID               int64
	NoticeID         int64
	AttachmentID     int64
	ArtifactType     string
	SchemaVersion    string
	SourceSpan       string
	RawText          string
	ContentJSON      []byte
	Confidence       float64
	NoticeSeq        string
	NoticeTitle      string
	OriginalFilename string
}

func (a LLMRepairArtifact) ValidateLLMRepairOfferingTarget() error {
	if a.NoticeID <= 0 || a.AttachmentID <= 0 {
		return fmt.Errorf("LLM repair offering persistence requires an attachment-backed artifact")
	}
	return nil
}

func (a LLMRepairArtifact) LLMRepairAttachmentRef() (PersistedAttachment, error) {
	if err := a.ValidateLLMRepairOfferingTarget(); err != nil {
		return PersistedAttachment{}, err
	}
	return PersistedAttachment{
		NoticeID:     a.NoticeID,
		AttachmentID: a.AttachmentID,
		Filename:     a.OriginalFilename,
	}, nil
}

func PrepareLLMRepairOfferings(output llm.RepairOutput) []normalize.OfferingCandidate {
	return normalize.OfferingsFromLLMRepairOutput(output)
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

func (r *Repository) UpsertApplicationNotice(ctx context.Context, input ApplicationNoticeInput) (int64, error) {
	var id int64
	noticeID := pgtype.Int8{}
	if input.NoticeID > 0 {
		noticeID = int8Value(input.NoticeID)
	}
	err := r.pool.QueryRow(ctx, `
insert into application_notices (
	notice_id,
	agency,
	source,
	sply_ty,
	recrnoti_cd,
	recr_ty,
	noti_no_hs_at,
	region_prior_rspe,
	title,
	status,
	supply_count,
	posted_at,
	source_url,
	raw_metadata
)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
on conflict (agency, source, sply_ty, recrnoti_cd)
do update set
	notice_id = coalesce(excluded.notice_id, application_notices.notice_id),
	recr_ty = excluded.recr_ty,
	noti_no_hs_at = excluded.noti_no_hs_at,
	region_prior_rspe = excluded.region_prior_rspe,
	title = excluded.title,
	status = excluded.status,
	supply_count = excluded.supply_count,
	posted_at = coalesce(excluded.posted_at, application_notices.posted_at),
	source_url = excluded.source_url,
	raw_metadata = excluded.raw_metadata,
	updated_at = now()
returning id
`,
		noticeID,
		firstNonEmpty(input.Agency, "SH"),
		firstNonEmpty(input.Source, "sh_app_user"),
		input.SupplyType,
		input.RecruitNoticeCode,
		input.RecruitType,
		input.NoticeNoHouse,
		input.RegionPriority,
		input.Title,
		input.Status,
		int4PtrValue(input.SupplyCount),
		dateValue(input.PostedAt),
		input.SourceURL,
		mustJSONAny(input.RawMetadata),
	).Scan(&id)
	return id, err
}

func (r *Repository) LinkApplicationNoticeToSourceNotice(ctx context.Context, applicationNoticeID int64, noticeID int64) error {
	_, err := r.pool.Exec(ctx, `
update application_notices
set
	notice_id = $2,
	updated_at = now()
where id = $1
`, applicationNoticeID, noticeID)
	return err
}

func (r *Repository) UpsertDiscoverySeenCache(ctx context.Context, input DiscoverySeenCacheInput) error {
	_, err := r.pool.Exec(ctx, `
insert into discovery_seen_cache (
	agency,
	board_kind,
	seq,
	status,
	reason,
	list_title,
	list_title_hash,
	posted_at,
	evidence_json,
	expires_at,
	policy_version,
	parser_version,
	last_detail_fetched_at
)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, now())
on conflict (agency, board_kind, seq)
do update set
	status = excluded.status,
	reason = excluded.reason,
	list_title = excluded.list_title,
	list_title_hash = excluded.list_title_hash,
	posted_at = excluded.posted_at,
	evidence_json = excluded.evidence_json,
	expires_at = excluded.expires_at,
	policy_version = excluded.policy_version,
	parser_version = excluded.parser_version,
	last_seen_at = now(),
	last_detail_fetched_at = now(),
	seen_count = discovery_seen_cache.seen_count + 1
`,
		input.Agency,
		input.BoardKind,
		input.Seq,
		string(input.Status),
		string(input.Reason),
		input.Title,
		discovery.SeenTitleHash(input.Title),
		dateValue(input.PostedAt),
		mustJSONAny(input.Evidence),
		timestamptzValue(input.ExpiresAt),
		input.PolicyVersion,
		input.ParserVersion,
	)
	return err
}

func (r *Repository) FreshDiscoverySeenCache(ctx context.Context, agency string, boardKind string, now time.Time) (map[string]discovery.SeenCacheEntry, error) {
	rows, err := r.pool.Query(ctx, `
select seq, status, list_title_hash, posted_at, expires_at
from discovery_seen_cache
where agency = $1
	and board_kind = $2
	and (expires_at is null or expires_at > $3)
`, agency, boardKind, timestamptzValue(now))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make(map[string]discovery.SeenCacheEntry)
	for rows.Next() {
		var entry discovery.SeenCacheEntry
		var status string
		var postedAt pgtype.Date
		var expiresAt pgtype.Timestamptz
		if err := rows.Scan(&entry.Seq, &status, &entry.ListTitleHash, &postedAt, &expiresAt); err != nil {
			return nil, err
		}
		entry.Status = discovery.SeenCacheStatus(status)
		if postedAt.Valid {
			entry.PostedAt = postedAt.Time
		}
		if expiresAt.Valid {
			entry.ExpiresAt = expiresAt.Time
		}
		entries[entry.Seq] = entry
	}
	return entries, rows.Err()
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
		var previewStoredID int64
		var previewObjectKey string
		if object.PreviewStoredObject != nil {
			previewStoredID, err = r.queries.UpsertStoredObject(ctx, db.UpsertStoredObjectParams{
				Bucket:           "hic-artifacts",
				ObjectKey:        object.PreviewStoredObject.Key,
				StorageBackend:   "local_filesystem",
				ContentType:      object.PreviewStoredObject.ContentType,
				OriginalFilename: object.PreviewStoredObject.OriginalName,
				Sha256:           object.PreviewStoredObject.SHA256,
				SizeBytes:        object.PreviewStoredObject.SizeBytes,
				Metadata:         mustJSON(object.PreviewStoredObject.Metadata),
			})
			if err != nil {
				return nil, err
			}
			previewObjectKey = object.PreviewStoredObject.Key
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
			NoticeID:              noticeID,
			StoredObjectID:        storedID,
			AttachmentID:          attachmentID,
			PreviewStoredObjectID: previewStoredID,
			FileSeq:               object.FileSeq,
			Filename:              object.Filename,
			ObjectKey:             object.StoredObject.Key,
			PreviewObjectKey:      previewObjectKey,
			Kind:                  object.Kind,
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

func (r *Repository) SaveMyHomeNoticeFile(ctx context.Context, noticeID int64, endpoint lhdiscovery.MyHomeEndpoint, item lhdiscovery.MyHomeNoticeItem, file lhdiscovery.MyHomeNoticeFile, stored global.StoredObject) (PersistedAttachment, error) {
	storedID, err := r.queries.UpsertStoredObject(ctx, db.UpsertStoredObjectParams{
		Bucket:           "hic-originals",
		ObjectKey:        stored.Key,
		StorageBackend:   "local_filesystem",
		ContentType:      stored.ContentType,
		OriginalFilename: stored.OriginalName,
		Sha256:           stored.SHA256,
		SizeBytes:        stored.SizeBytes,
		Metadata:         mustJSON(stored.Metadata),
	})
	if err != nil {
		return PersistedAttachment{}, err
	}
	kind := extraction.ClassifyAttachment(file.Filename)
	attachmentID, err := r.queries.UpsertAttachment(ctx, db.UpsertAttachmentParams{
		NoticeID:         int8Value(noticeID),
		StoredObjectID:   int8Value(storedID),
		BrdID:            "myhome:" + string(endpoint),
		Seq:              item.SourceSeq(),
		FileSeq:          file.FileSN,
		OriginalFilename: file.Filename,
		FileExt:          strings.TrimPrefix(strings.ToLower(filepath.Ext(file.Filename)), "."),
		FileSize:         stored.SizeBytes,
		ContentType:      stored.ContentType,
		ObjectKey:        stored.Key,
		Sha256:           stored.SHA256,
		AttachmentKind:   string(kind),
		ExtractorStatus:  "preserved",
		RawMetadata: mustJSON(map[string]string{
			"atch_file_id":    file.AtchFileID,
			"file_sn":         file.FileSN,
			"endpoint":        string(endpoint),
			"attachment_kind": string(kind),
		}),
	})
	if err != nil {
		return PersistedAttachment{}, err
	}
	return PersistedAttachment{
		NoticeID:       noticeID,
		StoredObjectID: storedID,
		AttachmentID:   attachmentID,
		FileSeq:        file.FileSN,
		Filename:       file.Filename,
		ObjectKey:      stored.Key,
		Kind:           kind,
	}, nil
}

func (r *Repository) UpsertOffering(ctx context.Context, attachment PersistedAttachment, sourceArtifactID int64, offering normalize.OfferingCandidate) (int64, error) {
	return r.queries.UpsertOffering(ctx, db.UpsertOfferingParams{
		NoticeID:             int8Value(attachment.NoticeID),
		AttachmentID:         int8Value(attachment.AttachmentID),
		SourceArtifactID:     int8Value(sourceArtifactID),
		Agency:               "SH",
		Source:               "sh",
		ApplicationUnitLabel: offering.ApplicationUnitLabel,
		SupplyMethod:         offering.SupplyMethod,
		ApplicationCategory:  offering.ApplicationCategory,
		SupplyCategory:       offering.SupplyCategory,
		ListNo:               offering.ListNo,
		District:             offering.District,
		Address:              offering.Address,
		LegalDong:            offering.LegalDong,
		AddressDetail:        offering.AddressDetail,
		HousingName:          offering.HousingName,
		ComplexName:          offering.ComplexName,
		BuildingName:         offering.BuildingName,
		UnitNo:               stringValue(offering.UnitNo),
		Floor:                int4PtrValue(offering.FloorNo),
		FloorNo:              int4PtrValue(offering.FloorNo),
		UnitType:             offering.UnitType,
		StructureType:        offering.StructureType,
		ExclusiveAreaM2:      numericPtrValue(offering.ExclusiveAreaM2),
		AreaPyeong:           numericPtrValue(offering.AreaPyeong),
		DepositText:          offering.DepositText,
		DepositKrw:           int8PtrValue(offering.DepositKRW),
		JeonseDepositText:    offering.JeonseDepositText,
		JeonseDepositKrw:     int8PtrValue(offering.JeonseDepositKRW),
		ContractDepositKrw:   int8PtrValue(offering.ContractDepositKRW),
		BalancePaymentKrw:    int8PtrValue(offering.BalancePaymentKRW),
		MonthlyRentText:      offering.MonthlyRentText,
		MonthlyRentKrw:       int8PtrValue(offering.MonthlyRentKRW),
		SupplyCount:          int4PtrValue(offering.SupplyCount),
		ReservedCount:        int4PtrValue(offering.ReservedCount),
		GenderRequirement:    offering.GenderRequirement,
		OccupancyType:        offering.OccupancyType,
		CapacityPersons:      int4PtrValue(offering.CapacityPersons),
		DormitoryFeeKrw:      int8PtrValue(offering.DormitoryFeeKRW),
		HeatingMethod:        offering.HeatingMethod,
		MoveInStartText:      offering.MoveInStartText,
		Direction:            offering.Direction,
		Status:               offering.Status,
		SourceSheet:          offering.SourceSheet,
		SourceRow:            int4Value(offering.SourceRow),
		SourceCell:           offering.SourceCell,
		SourcePage:           int4Value(offering.SourcePage),
		SourceSpan:           offering.SourceSpan,
		RawRow:               mustJSONAny(offering.RawRow),
		Confidence:           numericValue(offering.Confidence),
		QaStatus:             "pending",
	})
}

func (r *Repository) SaveMyHomeNotice(ctx context.Context, endpoint lhdiscovery.MyHomeEndpoint, item lhdiscovery.MyHomeNoticeItem) (int64, error) {
	boardKind := myHomeBoardKind(endpoint)
	boardID, err := r.queries.UpsertSourceBoard(ctx, db.UpsertSourceBoardParams{
		Agency:    firstNonEmpty(item.Agency, "LH"),
		BoardKind: boardKind,
		Name:      myHomeBoardName(endpoint),
		SourceUrl: "https://apis.data.go.kr/1613000/HWSPR02/" + string(endpoint),
	})
	if err != nil {
		return 0, err
	}

	return r.queries.UpsertSourceNotice(ctx, db.UpsertSourceNoticeParams{
		SourceBoardID: int8Value(boardID),
		Agency:        firstNonEmpty(item.Agency, "LH"),
		BoardKind:     boardKind,
		Seq:           item.SourceSeq(),
		Category:      string(discovery.NoticeCategoryRecruitment),
		NoticeType:    "recruitment",
		NoticeSubtype: "",
		Title:         item.Title,
		PostedAt:      dateValue(parseYYYYMMDDDate(item.PostedDate)),
		SourceUrl:     firstNonEmpty(item.DetailURL, item.SourceURL),
		BodyText:      "",
	})
}

func (r *Repository) InsertMyHomeArtifact(ctx context.Context, endpoint lhdiscovery.MyHomeEndpoint, item lhdiscovery.MyHomeNoticeItem) (int64, string, error) {
	sourceSpan := lhdiscovery.MyHomeSourceSpan(endpoint, item)
	content, err := json.Marshal(item.Raw)
	if err != nil {
		return 0, "", err
	}
	var id int64
	err = r.pool.QueryRow(ctx, `
insert into extracted_artifacts (
	artifact_type,
	extractor,
	status,
	schema_version,
	raw_text,
	content_json,
	source_span,
	confidence
)
values ($1, $2, $3, $4, $5, $6, $7, $8)
on conflict (artifact_type, source_span)
where attachment_id is null and source_span <> ''
do update set
	extractor = excluded.extractor,
	status = excluded.status,
	schema_version = excluded.schema_version,
	raw_text = excluded.raw_text,
	content_json = excluded.content_json,
	confidence = excluded.confidence
returning id
`,
		string(extraction.ArtifactTypeMyHomeAPIItem),
		"myhome-openapi",
		string(extraction.ArtifactStatusExtracted),
		"v1",
		item.Title,
		content,
		sourceSpan,
		numericValue(1),
	).Scan(&id)
	return id, sourceSpan, err
}

func (r *Repository) InsertMyHomeDetailArtifact(ctx context.Context, endpoint lhdiscovery.MyHomeEndpoint, item lhdiscovery.MyHomeNoticeItem, detail lhdiscovery.MyHomeNoticeDetail) (int64, string, error) {
	sourceSpan := lhdiscovery.MyHomeSourceSpan(endpoint, item) + "#detail"
	content, err := detail.JSONContent()
	if err != nil {
		return 0, "", err
	}
	var id int64
	err = r.pool.QueryRow(ctx, `
insert into extracted_artifacts (
	artifact_type,
	extractor,
	status,
	schema_version,
	raw_text,
	content_json,
	source_span,
	confidence
)
values ($1, $2, $3, $4, $5, $6, $7, $8)
on conflict (artifact_type, source_span)
where attachment_id is null and source_span <> ''
do update set
	extractor = excluded.extractor,
	status = excluded.status,
	schema_version = excluded.schema_version,
	raw_text = excluded.raw_text,
	content_json = excluded.content_json,
	confidence = excluded.confidence
returning id
`,
		string(extraction.ArtifactTypeMyHomeDetail),
		"myhome-detail-html",
		string(extraction.ArtifactStatusExtracted),
		"v1",
		detail.RawText,
		content,
		sourceSpan,
		numericValue(1),
	).Scan(&id)
	return id, sourceSpan, err
}

func (r *Repository) UpsertMyHomeOffering(ctx context.Context, noticeID int64, sourceArtifactID int64, agency string, offering normalize.OfferingCandidate) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
insert into offerings (
	notice_id,
	source_artifact_id,
	agency,
	source,
	application_unit_label,
	supply_method,
	application_category,
	supply_category,
	list_no,
	district,
	address,
	legal_dong,
	address_detail,
	housing_name,
	complex_name,
	building_name,
	unit_no,
	floor,
	floor_no,
	unit_type,
	structure_type,
	exclusive_area_m2,
	area_pyeong,
	deposit_text,
	deposit_krw,
	jeonse_deposit_text,
	jeonse_deposit_krw,
	contract_deposit_krw,
	balance_payment_krw,
	monthly_rent_text,
	monthly_rent_krw,
	supply_count,
	reserved_count,
	gender_requirement,
	occupancy_type,
	capacity_persons,
	dormitory_fee_krw,
	heating_method,
	move_in_start_text,
	direction,
	status,
	source_sheet,
	source_row,
	source_cell,
	source_page,
	source_span,
	raw_row,
	confidence,
	qa_status
)
values (
	$1, $2, $3, 'myhome', $4, $5, $6, $7, $8, $9,
	$10, $11, $12, $13, $14, $15, $16, $17, $18, $19,
	$20, $21, $22, $23, $24, $25, $26, $27, $28, $29,
	$30, $31, $32, $33, $34, $35, $36, $37, $38, $39,
	$40, $41, $42, $43, $44, $45, $46, $47, $48
)
on conflict (notice_id, source_span)
where attachment_id is null and source_span <> ''
do update set
	source_artifact_id = excluded.source_artifact_id,
	agency = excluded.agency,
	application_unit_label = excluded.application_unit_label,
	supply_method = excluded.supply_method,
	application_category = excluded.application_category,
	supply_category = excluded.supply_category,
	list_no = excluded.list_no,
	district = excluded.district,
	address = excluded.address,
	legal_dong = excluded.legal_dong,
	address_detail = excluded.address_detail,
	housing_name = excluded.housing_name,
	complex_name = excluded.complex_name,
	building_name = excluded.building_name,
	unit_no = excluded.unit_no,
	floor = excluded.floor,
	floor_no = excluded.floor_no,
	unit_type = excluded.unit_type,
	structure_type = excluded.structure_type,
	exclusive_area_m2 = excluded.exclusive_area_m2,
	area_pyeong = excluded.area_pyeong,
	deposit_text = excluded.deposit_text,
	deposit_krw = excluded.deposit_krw,
	jeonse_deposit_text = excluded.jeonse_deposit_text,
	jeonse_deposit_krw = excluded.jeonse_deposit_krw,
	contract_deposit_krw = excluded.contract_deposit_krw,
	balance_payment_krw = excluded.balance_payment_krw,
	monthly_rent_text = excluded.monthly_rent_text,
	monthly_rent_krw = excluded.monthly_rent_krw,
	supply_count = excluded.supply_count,
	reserved_count = excluded.reserved_count,
	gender_requirement = excluded.gender_requirement,
	occupancy_type = excluded.occupancy_type,
	capacity_persons = excluded.capacity_persons,
	dormitory_fee_krw = excluded.dormitory_fee_krw,
	heating_method = excluded.heating_method,
	move_in_start_text = excluded.move_in_start_text,
	direction = excluded.direction,
	status = excluded.status,
	source_sheet = excluded.source_sheet,
	source_row = excluded.source_row,
	source_cell = excluded.source_cell,
	source_page = excluded.source_page,
	raw_row = excluded.raw_row,
	confidence = excluded.confidence,
	qa_status = case when offerings.qa_status = 'approved' then 'approved' else excluded.qa_status end
returning id
`,
		int8Value(noticeID),
		int8Value(sourceArtifactID),
		firstNonEmpty(agency, "LH"),
		offering.ApplicationUnitLabel,
		offering.SupplyMethod,
		offering.ApplicationCategory,
		offering.SupplyCategory,
		offering.ListNo,
		offering.District,
		offering.Address,
		offering.LegalDong,
		offering.AddressDetail,
		offering.HousingName,
		offering.ComplexName,
		offering.BuildingName,
		stringValue(offering.UnitNo),
		int4PtrValue(offering.FloorNo),
		int4PtrValue(offering.FloorNo),
		offering.UnitType,
		offering.StructureType,
		numericPtrValue(offering.ExclusiveAreaM2),
		numericPtrValue(offering.AreaPyeong),
		offering.DepositText,
		int8PtrValue(offering.DepositKRW),
		offering.JeonseDepositText,
		int8PtrValue(offering.JeonseDepositKRW),
		int8PtrValue(offering.ContractDepositKRW),
		int8PtrValue(offering.BalancePaymentKRW),
		offering.MonthlyRentText,
		int8PtrValue(offering.MonthlyRentKRW),
		int4PtrValue(offering.SupplyCount),
		int4PtrValue(offering.ReservedCount),
		offering.GenderRequirement,
		offering.OccupancyType,
		int4PtrValue(offering.CapacityPersons),
		int8PtrValue(offering.DormitoryFeeKRW),
		offering.HeatingMethod,
		offering.MoveInStartText,
		offering.Direction,
		offering.Status,
		offering.SourceSheet,
		int4Value(offering.SourceRow),
		offering.SourceCell,
		int4Value(offering.SourcePage),
		offering.SourceSpan,
		mustJSONAny(offering.RawRow),
		numericValue(offering.Confidence),
		"pending",
	).Scan(&id)
	return id, err
}

func (r *Repository) UpsertNoticeSchedule(ctx context.Context, schedule normalize.NoticeScheduleCandidate) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
insert into notice_schedules (
	notice_id,
	source_artifact_id,
	schedule_type,
	label,
	starts_at,
	ends_at,
	date_text,
	channel,
	note,
	source_text,
	source_span,
	confidence
)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
on conflict (notice_id, schedule_type, source_span)
where source_span <> ''
do update set
	source_artifact_id = excluded.source_artifact_id,
	label = excluded.label,
	starts_at = excluded.starts_at,
	ends_at = excluded.ends_at,
	date_text = excluded.date_text,
	channel = excluded.channel,
	note = excluded.note,
	source_text = excluded.source_text,
	confidence = excluded.confidence
returning id
`,
		schedule.NoticeID,
		int8Value(schedule.SourceArtifactID),
		schedule.ScheduleType,
		schedule.Label,
		timestamptzValue(schedule.StartsAt),
		timestamptzValue(schedule.EndsAt),
		schedule.DateText,
		schedule.Channel,
		schedule.Note,
		schedule.SourceText,
		schedule.SourceSpan,
		numericValue(schedule.Confidence),
	).Scan(&id)
	return id, err
}

func (r *Repository) Counts(ctx context.Context) (storedObjects int64, artifacts int64, err error) {
	storedObjects, err = r.queries.CountStoredObjects(ctx)
	if err != nil {
		return 0, 0, err
	}
	artifacts, err = r.queries.CountExtractedArtifacts(ctx)
	return storedObjects, artifacts, err
}

func (r *Repository) CountOfferings(ctx context.Context) (int64, error) {
	return r.queries.CountOfferings(ctx)
}

func (r *Repository) GetLLMRepairArtifact(ctx context.Context, artifactID int64) (LLMRepairArtifact, error) {
	row := r.pool.QueryRow(ctx, `
select
	ea.id,
	coalesce(a.notice_id, 0),
	coalesce(ea.attachment_id, 0),
	ea.artifact_type,
	ea.schema_version,
	ea.source_span,
	ea.raw_text,
	ea.content_json,
	ea.confidence,
	coalesce(sn.seq, ''),
	coalesce(sn.title, ''),
	coalesce(a.original_filename, '')
from extracted_artifacts ea
left join attachments a on a.id = ea.attachment_id
left join source_notices sn on sn.id = a.notice_id
where ea.id = $1
`, artifactID)

	var artifact LLMRepairArtifact
	var confidence pgtype.Numeric
	if err := row.Scan(
		&artifact.ID,
		&artifact.NoticeID,
		&artifact.AttachmentID,
		&artifact.ArtifactType,
		&artifact.SchemaVersion,
		&artifact.SourceSpan,
		&artifact.RawText,
		&artifact.ContentJSON,
		&confidence,
		&artifact.NoticeSeq,
		&artifact.NoticeTitle,
		&artifact.OriginalFilename,
	); err != nil {
		return LLMRepairArtifact{}, err
	}
	if value := numericToFloat64Ptr(confidence); value != nil {
		artifact.Confidence = *value
	}
	return artifact, nil
}

func (r *Repository) ListLLMRepairCandidates(ctx context.Context, limit int32, includeApprovedNotices bool) ([]LLMRepairArtifact, error) {
	if limit <= 0 || limit > 1000 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
select
	ea.id,
	coalesce(a.notice_id, 0),
	coalesce(ea.attachment_id, 0),
	ea.artifact_type,
	ea.schema_version,
	ea.source_span,
	ea.raw_text,
	ea.content_json,
	ea.confidence,
	coalesce(sn.seq, ''),
	coalesce(sn.title, ''),
	coalesce(a.original_filename, '')
from extracted_artifacts ea
join attachments a on a.id = ea.attachment_id
join source_notices sn on sn.id = a.notice_id
where sn.category = 'recruitment'
	and ea.status = 'extracted'
	and ea.source_span like 'object://%'
	and trim(ea.raw_text) <> ''
	and ea.artifact_type in ('pdf_text', 'html_preview', 'hwp_text', 'hwpx_text')
	and a.attachment_kind in ('notice_pdf', 'notice_hwp')
	and not exists (
		select 1
		from offerings o
		where o.source_artifact_id = ea.id
	)
	and not exists (
		select 1
		from llm_repair_attempts attempt
		where attempt.artifact_id = ea.id
			and attempt.status = 'succeeded'
	)
	and (
		$2::boolean
		or not exists (
			select 1
			from offerings approved
			where approved.notice_id = sn.id
				and approved.qa_status = 'approved'
		)
	)
order by
	case when exists (
		select 1
		from offerings approved
		where approved.notice_id = sn.id
			and approved.qa_status = 'approved'
	) then 1 else 0 end,
	case ea.artifact_type when 'pdf_text' then 0 else 1 end,
	length(ea.raw_text) desc,
	ea.id
limit $1
`, limit, includeApprovedNotices)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []LLMRepairArtifact
	for rows.Next() {
		var artifact LLMRepairArtifact
		var confidence pgtype.Numeric
		if err := rows.Scan(
			&artifact.ID,
			&artifact.NoticeID,
			&artifact.AttachmentID,
			&artifact.ArtifactType,
			&artifact.SchemaVersion,
			&artifact.SourceSpan,
			&artifact.RawText,
			&artifact.ContentJSON,
			&confidence,
			&artifact.NoticeSeq,
			&artifact.NoticeTitle,
			&artifact.OriginalFilename,
		); err != nil {
			return nil, err
		}
		if value := numericToFloat64Ptr(confidence); value != nil {
			artifact.Confidence = *value
		}
		candidates = append(candidates, artifact)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return candidates, nil
}

func (r *Repository) InsertLLMRepairAttempt(ctx context.Context, attempt llm.AttemptRecord) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
insert into llm_repair_attempts (
	artifact_id,
	schema_version,
	prompt_version,
	model,
	input_hash,
	output_hash,
	status,
	confidence,
	source_span,
	request_json,
	response_json,
	error_text
)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
returning id
`,
		attempt.ArtifactID,
		attempt.SchemaVersion,
		attempt.PromptVersion,
		attempt.Model,
		attempt.InputHash,
		attempt.OutputHash,
		attempt.Status,
		numericValue(attempt.Confidence),
		attempt.SourceSpan,
		jsonBytesOrEmpty(attempt.RequestJSON),
		jsonBytesOrEmpty(attempt.ResponseJSON),
		attempt.ErrorText,
	).Scan(&id)
	return id, err
}

func (r *Repository) CountLLMRepairAttempts(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `select count(*) from llm_repair_attempts`).Scan(&count)
	return count, err
}

func (r *Repository) DeleteLLMRepairOfferings(ctx context.Context, sourceArtifactID int64) error {
	_, err := r.pool.Exec(ctx, `
delete from offerings
where source_artifact_id = $1
	and raw_row->>'source' = 'llm_repair'
`, sourceArtifactID)
	return err
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

func (r *Repository) ExistingNoticeCandidates(ctx context.Context, agency string, boardKind string) ([]discovery.Candidate, map[string]int64, error) {
	rows, err := r.pool.Query(ctx, `
select id, seq, title, posted_at
from source_notices
where agency = $1
	and board_kind = $2
	and category = 'recruitment'
`, agency, boardKind)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var candidates []discovery.Candidate
	idsBySeq := make(map[string]int64)
	for rows.Next() {
		var id int64
		var seq string
		var title string
		var postedAt time.Time
		if err := rows.Scan(&id, &seq, &title, &postedAt); err != nil {
			return nil, nil, err
		}
		seq = strings.TrimSpace(seq)
		if seq == "" {
			continue
		}
		idsBySeq[seq] = id
		candidates = append(candidates, discovery.Candidate{
			Agency:    agency,
			BoardKind: boardKind,
			Seq:       seq,
			Title:     title,
			PostedAt:  postedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return candidates, idsBySeq, nil
}

func (r *Repository) PromoteOfferingsQA(ctx context.Context) (QASummary, error) {
	if err := r.queries.PromoteOfferingsQA(ctx); err != nil {
		return QASummary{}, err
	}
	return r.OfferingsQASummary(ctx)
}

func (r *Repository) OfferingsQASummary(ctx context.Context) (QASummary, error) {
	approved, err := r.queries.CountOfferingsByQAStatus(ctx, "approved")
	if err != nil {
		return QASummary{}, err
	}
	rejected, err := r.queries.CountOfferingsByQAStatus(ctx, "rejected")
	if err != nil {
		return QASummary{}, err
	}
	pending, err := r.queries.CountOfferingsByQAStatus(ctx, "pending")
	if err != nil {
		return QASummary{}, err
	}
	return QASummary{Approved: approved, Rejected: rejected, Pending: pending}, nil
}

func (r *Repository) ListOfferings(ctx context.Context, limit int32, qaStatus string) ([]OfferingView, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	if strings.TrimSpace(qaStatus) == "" {
		qaStatus = "approved"
	}
	rows, err := r.queries.ListOfferings(ctx, db.ListOfferingsParams{
		Limit:    limit,
		QaStatus: qaStatus,
	})
	if err != nil {
		return nil, err
	}
	offerings := make([]OfferingView, 0, len(rows))
	for _, row := range rows {
		offerings = append(offerings, OfferingView{
			ID:                   row.ID,
			Agency:               row.Agency,
			Source:               row.Source,
			ApplicationUnitLabel: row.ApplicationUnitLabel,
			SupplyMethod:         row.SupplyMethod,
			ApplicationCategory:  row.ApplicationCategory,
			SupplyCategory:       row.SupplyCategory,
			ListNo:               row.ListNo,
			District:             row.District,
			Address:              row.Address,
			HousingName:          row.HousingName,
			ComplexName:          row.ComplexName,
			BuildingName:         row.BuildingName,
			UnitNo:               textToString(row.UnitNo),
			UnitType:             row.UnitType,
			ExclusiveAreaM2:      numericToFloat64Ptr(row.ExclusiveAreaM2),
			DepositText:          row.DepositText,
			DepositKRW:           int8ToInt64Ptr(row.DepositKrw),
			JeonseDepositText:    row.JeonseDepositText,
			JeonseDepositKRW:     int8ToInt64Ptr(row.JeonseDepositKrw),
			ContractDepositKRW:   int8ToInt64Ptr(row.ContractDepositKrw),
			BalancePaymentKRW:    int8ToInt64Ptr(row.BalancePaymentKrw),
			MonthlyRentText:      row.MonthlyRentText,
			MonthlyRentKRW:       int8ToInt64Ptr(row.MonthlyRentKrw),
			SupplyCount:          int4ToInt32Ptr(row.SupplyCount),
			ReservedCount:        int4ToInt32Ptr(row.ReservedCount),
			GenderRequirement:    row.GenderRequirement,
			OccupancyType:        row.OccupancyType,
			CapacityPersons:      int4ToInt32Ptr(row.CapacityPersons),
			DormitoryFeeKRW:      int8ToInt64Ptr(row.DormitoryFeeKrw),
			HeatingMethod:        row.HeatingMethod,
			MoveInStartText:      row.MoveInStartText,
			SourceSheet:          row.SourceSheet,
			SourceRow:            int4ToInt32Ptr(row.SourceRow),
			SourceSpan:           row.SourceSpan,
			QAStatus:             row.QaStatus,
		})
	}
	return offerings, nil
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

func (r *Repository) ListSchedules(ctx context.Context, limit int32) ([]ScheduleView, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := r.pool.Query(ctx, `
select
	ns.id,
	ns.notice_id,
	ns.schedule_type,
	ns.label,
	ns.starts_at,
	ns.ends_at,
	ns.date_text,
	ns.channel,
	ns.note,
	ns.source_text,
	ns.source_span
from notice_schedules ns
where exists (
	select 1
	from offerings o
	where o.notice_id = ns.notice_id
		and o.qa_status = 'approved'
)
order by ns.starts_at nulls last, ns.id
limit $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []ScheduleView
	for rows.Next() {
		var schedule ScheduleView
		var startsAt pgtype.Timestamptz
		var endsAt pgtype.Timestamptz
		if err := rows.Scan(
			&schedule.ID,
			&schedule.NoticeID,
			&schedule.ScheduleType,
			&schedule.Label,
			&startsAt,
			&endsAt,
			&schedule.DateText,
			&schedule.Channel,
			&schedule.Note,
			&schedule.SourceText,
			&schedule.SourceSpan,
		); err != nil {
			return nil, err
		}
		schedule.StartsAt = timestamptzToString(startsAt)
		schedule.EndsAt = timestamptzToString(endsAt)
		schedules = append(schedules, schedule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return schedules, nil
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

func timestamptzValue(value time.Time) pgtype.Timestamptz {
	if value.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: value, Valid: true}
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

func timestamptzToString(value pgtype.Timestamptz) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format(time.RFC3339)
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

func jsonBytesOrEmpty(value []byte) []byte {
	if len(value) == 0 {
		return []byte("{}")
	}
	return value
}

func stringValue(value string) pgtype.Text {
	if strings.TrimSpace(value) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func textToString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func parseYYYYMMDDDate(value string) time.Time {
	value = strings.TrimSpace(value)
	if len(value) != 8 {
		return time.Time{}
	}
	parsed, err := time.ParseInLocation("20060102", value, time.Local)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func myHomeBoardKind(endpoint lhdiscovery.MyHomeEndpoint) string {
	switch endpoint {
	case lhdiscovery.MyHomeSale:
		return "myhome_sale"
	default:
		return "myhome_rental"
	}
}

func myHomeBoardName(endpoint lhdiscovery.MyHomeEndpoint) string {
	switch endpoint {
	case lhdiscovery.MyHomeSale:
		return "마이홈 공공분양주택 모집공고"
	default:
		return "마이홈 공공임대주택 모집공고"
	}
}
