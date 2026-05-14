-- name: CreateCollectionRun :one
insert into collection_runs (source, status, stats)
values ($1, 'running', '{}'::jsonb)
returning id;

-- name: FinishCollectionRun :exec
update collection_runs
set
	status = $2,
	finished_at = now(),
	stats = $3,
	error_text = $4
where id = $1;

-- name: UpsertSourceBoard :one
insert into source_boards (agency, board_kind, name, source_url)
values ($1, $2, $3, $4)
on conflict (agency, board_kind)
do update set
	name = excluded.name,
	source_url = excluded.source_url,
	updated_at = now()
returning id;

-- name: UpsertSourceNotice :one
insert into source_notices (
	source_board_id,
	agency,
	board_kind,
	seq,
	category,
	notice_type,
	notice_subtype,
	title,
	posted_at,
	source_url,
	body_text
)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
on conflict (agency, board_kind, seq)
do update set
	source_board_id = excluded.source_board_id,
	category = excluded.category,
	notice_type = excluded.notice_type,
	notice_subtype = excluded.notice_subtype,
	title = excluded.title,
	posted_at = excluded.posted_at,
	source_url = excluded.source_url,
	body_text = excluded.body_text,
	updated_at = now()
returning id;

-- name: UpsertStoredObject :one
insert into stored_objects (
	bucket,
	object_key,
	storage_backend,
	content_type,
	original_filename,
	sha256,
	size_bytes,
	metadata
)
values ($1, $2, $3, $4, $5, $6, $7, $8)
on conflict (bucket, object_key)
do update set
	storage_backend = excluded.storage_backend,
	content_type = excluded.content_type,
	original_filename = excluded.original_filename,
	sha256 = excluded.sha256,
	size_bytes = excluded.size_bytes,
	metadata = excluded.metadata
returning id;

-- name: UpsertAttachment :one
insert into attachments (
	notice_id,
	stored_object_id,
	brd_id,
	seq,
	file_seq,
	original_filename,
	file_ext,
	file_size,
	content_type,
	object_key,
	sha256,
	attachment_kind,
	extractor_status,
	raw_metadata
)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
on conflict (brd_id, seq, file_seq)
do update set
	notice_id = excluded.notice_id,
	stored_object_id = excluded.stored_object_id,
	original_filename = excluded.original_filename,
	file_ext = excluded.file_ext,
	file_size = excluded.file_size,
	content_type = excluded.content_type,
	object_key = excluded.object_key,
	sha256 = excluded.sha256,
	attachment_kind = excluded.attachment_kind,
	extractor_status = excluded.extractor_status,
	raw_metadata = excluded.raw_metadata
returning id;

-- name: InsertExtractedArtifact :one
insert into extracted_artifacts (
	attachment_id,
	stored_object_id,
	artifact_type,
	extractor,
	status,
	schema_version,
	sheet_name,
	page_no,
	row_no,
	cell_ref,
	raw_text,
	content_json,
	source_span,
	confidence,
	error_text
)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
on conflict (attachment_id, artifact_type, source_span)
do update set
	stored_object_id = excluded.stored_object_id,
	extractor = excluded.extractor,
	status = excluded.status,
	schema_version = excluded.schema_version,
	sheet_name = excluded.sheet_name,
	page_no = excluded.page_no,
	row_no = excluded.row_no,
	cell_ref = excluded.cell_ref,
	raw_text = excluded.raw_text,
	content_json = excluded.content_json,
	confidence = excluded.confidence,
	error_text = excluded.error_text
returning id;

-- name: UpsertHousingUnit :one
insert into housing_units (
	notice_id,
	attachment_id,
	source_artifact_id,
	agency,
	source,
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
	monthly_rent_text,
	monthly_rent_krw,
	supply_count,
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
	$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
	$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
	$21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
	$31, $32, $33, $34, $35, $36
)
on conflict (attachment_id, source_span)
where source_span <> ''
do update set
	source_artifact_id = excluded.source_artifact_id,
	agency = excluded.agency,
	source = excluded.source,
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
	monthly_rent_text = excluded.monthly_rent_text,
	monthly_rent_krw = excluded.monthly_rent_krw,
	supply_count = excluded.supply_count,
	direction = excluded.direction,
	status = excluded.status,
	source_sheet = excluded.source_sheet,
	source_row = excluded.source_row,
	source_cell = excluded.source_cell,
	source_page = excluded.source_page,
	raw_row = excluded.raw_row,
	confidence = excluded.confidence,
	qa_status = case when housing_units.qa_status = 'approved' then 'approved' else excluded.qa_status end
returning id;

-- name: CountStoredObjects :one
select count(*) from stored_objects;

-- name: CountExtractedArtifacts :one
select count(*) from extracted_artifacts;

-- name: CountHousingUnits :one
select count(*) from housing_units;

-- name: CountHousingUnitsByQAStatus :one
select count(*) from housing_units
where qa_status = $1;

-- name: PromoteHousingUnitsQA :exec
update housing_units
set qa_status = case
	when exists (
			select 1
			from source_notices sn
			where sn.id = housing_units.notice_id
				and sn.category = 'recruitment'
		)
		and notice_id is not null
		and attachment_id is not null
		and source_artifact_id is not null
		and trim(unit_no) <> ''
		and trim(address) <> ''
		and exclusive_area_m2 is not null
		and exclusive_area_m2 > 0
		and deposit_krw is not null
		and deposit_krw >= 0
		and monthly_rent_krw is not null
		and monthly_rent_krw >= 0
		and trim(source_span) <> ''
	then 'approved'
	else 'rejected'
end
where qa_status = 'pending';

-- name: ListExistingNoticeSeqs :many
select seq
from source_notices
where agency = $1
	and board_kind = $2
	and category = 'recruitment';

-- name: ListHousingUnits :many
select
	id,
	agency,
	source,
	supply_category,
	list_no,
	district,
	address,
	housing_name,
	complex_name,
	building_name,
	unit_no,
	unit_type,
	exclusive_area_m2,
	deposit_text,
	deposit_krw,
	monthly_rent_text,
	monthly_rent_krw,
	source_sheet,
	source_row,
	source_span,
	qa_status
from housing_units
where qa_status = $2
order by id desc
limit $1;

-- name: ListSourceNotices :many
select
	id,
	agency,
	board_kind,
	seq,
	notice_type,
	title,
	posted_at,
	source_url
from source_notices
where category = 'recruitment'
	and not (
		title like '%당첨자%'
		or title like '%서류심사대상자%'
		or title like '%동호배정%'
		or title like '%결과%'
		or title like '%계약%'
		or title like '%청약경쟁률%'
		or title like '%경쟁률%'
		or title like '%청약접수%'
		or title like '%접수결과%'
		or title like '%접수 결과%'
		or title like '%접수마감%'
		or title like '%시스템%'
		or title like '%서비스 일시중단%'
	)
order by posted_at desc nulls last, id desc
limit $1;
