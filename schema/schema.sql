-- House Information Collector canonical PostgreSQL schema.
-- This file is the SQL artifact used by the prototype migrator and later sqlc setup.

create table if not exists collection_runs (
	id bigserial primary key,
	source text not null,
	status text not null,
	started_at timestamptz not null default now(),
	finished_at timestamptz,
	stats jsonb not null default '{}'::jsonb,
	error_text text
);

create table if not exists source_boards (
	id bigserial primary key,
	agency text not null,
	board_kind text not null,
	name text not null,
	source_url text not null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (agency, board_kind)
);

create table if not exists raw_documents (
	id bigserial primary key,
	run_id bigint references collection_runs(id),
	source text not null,
	source_url text not null,
	method text not null,
	request_body text not null default '',
	content_type text not null default '',
	fetched_at timestamptz not null default now(),
	body_text text not null default '',
	body_json jsonb,
	body_sha256 text not null,
	metadata jsonb not null default '{}'::jsonb
);

create index if not exists idx_raw_documents_source_url on raw_documents(source, source_url);
create index if not exists idx_raw_documents_body_json on raw_documents using gin(body_json);

create table if not exists stored_objects (
	id bigserial primary key,
	bucket text not null default 'local',
	object_key text not null,
	storage_backend text not null default 'local_filesystem',
	content_type text not null default '',
	original_filename text not null default '',
	sha256 text not null,
	size_bytes bigint not null default 0,
	stored_at timestamptz not null default now(),
	metadata jsonb not null default '{}'::jsonb,
	unique (bucket, object_key)
);

alter table if exists stored_objects drop constraint if exists stored_objects_sha256_size_bytes_key;
create index if not exists idx_stored_objects_sha256 on stored_objects(sha256);
create index if not exists idx_stored_objects_metadata on stored_objects using gin(metadata);

create table if not exists source_notices (
	id bigserial primary key,
	source_board_id bigint references source_boards(id),
	agency text not null,
	board_kind text not null,
	seq text not null,
	category text not null,
	notice_type text not null default 'recruitment',
	notice_subtype text not null default '',
	title text not null,
	department text not null default '',
	posted_at date,
	view_count integer not null default 0,
	detail_url text not null default '',
	source_url text not null default '',
	body_text text not null default '',
	raw_document_id bigint references raw_documents(id),
	metadata jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (agency, board_kind, seq)
);

alter table if exists source_notices add column if not exists source_board_id bigint references source_boards(id);
alter table if exists source_notices add column if not exists notice_type text not null default 'recruitment';
alter table if exists source_notices add column if not exists notice_subtype text not null default '';
alter table if exists source_notices add column if not exists source_url text not null default '';

create index if not exists idx_source_notices_category on source_notices(category);
create index if not exists idx_source_notices_notice_type on source_notices(notice_type, notice_subtype);
create index if not exists idx_source_notices_posted_at on source_notices(posted_at desc);
create index if not exists idx_source_notices_metadata on source_notices using gin(metadata);

create table if not exists attachments (
	id bigserial primary key,
	notice_id bigint references source_notices(id),
	source_document_id bigint references raw_documents(id),
	stored_object_id bigint references stored_objects(id),
	brd_id text not null,
	seq text not null,
	file_seq text not null,
	original_filename text not null,
	file_ext text not null default '',
	file_size bigint not null default 0,
	content_type text not null default '',
	preview_url text not null default '',
	download_url text not null default '',
	storage_path text not null default '',
	object_key text not null default '',
	sha256 text not null default '',
	attachment_kind text not null default '',
	extractor_status text not null default '',
	raw_metadata jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now(),
	unique (brd_id, seq, file_seq)
);

alter table if exists attachments add column if not exists stored_object_id bigint references stored_objects(id);
alter table if exists attachments add column if not exists content_type text not null default '';
alter table if exists attachments add column if not exists object_key text not null default '';
alter table if exists attachments add column if not exists sha256 text not null default '';
alter table if exists attachments add column if not exists attachment_kind text not null default '';
alter table if exists attachments add column if not exists extractor_status text not null default '';

create index if not exists idx_attachments_notice_id on attachments(notice_id);
create index if not exists idx_attachments_stored_object_id on attachments(stored_object_id);
create index if not exists idx_attachments_kind on attachments(attachment_kind);
create index if not exists idx_attachments_raw_metadata on attachments using gin(raw_metadata);

create table if not exists attachment_extractions (
	id bigserial primary key,
	attachment_id bigint references attachments(id),
	extractor text not null,
	status text not null,
	extracted_at timestamptz not null default now(),
	sheet_name text not null default '',
	page_no integer,
	row_no integer,
	raw_text text not null default '',
	raw_json jsonb not null default '{}'::jsonb,
	error_text text not null default ''
);

create index if not exists idx_attachment_extractions_attachment_id on attachment_extractions(attachment_id);
create index if not exists idx_attachment_extractions_raw_json on attachment_extractions using gin(raw_json);

create table if not exists extracted_artifacts (
	id bigserial primary key,
	attachment_id bigint references attachments(id),
	stored_object_id bigint references stored_objects(id),
	artifact_type text not null,
	extractor text not null,
	status text not null,
	schema_version text not null default 'v1',
	sheet_name text not null default '',
	page_no integer,
	row_no integer,
	cell_ref text not null default '',
	raw_text text not null default '',
	content_json jsonb not null default '{}'::jsonb,
	source_span text not null default '',
	confidence numeric not null default 0,
	error_text text not null default '',
	created_at timestamptz not null default now()
);

create index if not exists idx_extracted_artifacts_attachment_id on extracted_artifacts(attachment_id);
create index if not exists idx_extracted_artifacts_type_status on extracted_artifacts(artifact_type, status);
create index if not exists idx_extracted_artifacts_content_json on extracted_artifacts using gin(content_json);
create unique index if not exists uq_extracted_artifacts_attachment_type_span
	on extracted_artifacts(attachment_id, artifact_type, source_span);

do $$
begin
	if to_regclass('public.offerings') is null and to_regclass('public.housing_units') is not null then
		alter table housing_units rename to offerings;
	end if;
	if to_regclass('public.offering_conversion_estimates') is null and to_regclass('public.housing_unit_conversion_estimates') is not null then
		alter table housing_unit_conversion_estimates rename to offering_conversion_estimates;
	end if;
	if to_regclass('public.offering_conversion_estimates') is not null
		and exists (
			select 1
			from information_schema.columns
			where table_schema = 'public'
				and table_name = 'offering_conversion_estimates'
				and column_name = 'housing_unit_id'
		)
		and not exists (
			select 1
			from information_schema.columns
			where table_schema = 'public'
				and table_name = 'offering_conversion_estimates'
				and column_name = 'offering_id'
		)
	then
		alter table offering_conversion_estimates rename column housing_unit_id to offering_id;
	end if;
end $$;

create table if not exists offerings (
	id bigserial primary key,
	notice_id bigint references source_notices(id),
	attachment_id bigint references attachments(id),
	source_artifact_id bigint references extracted_artifacts(id),
	agency text not null,
	source text not null,
	offering_type text not null default 'unit',
	supply_category text not null default '',
	list_no text not null default '',
	district text not null default '',
	address text not null default '',
	legal_dong text not null default '',
	address_detail text not null default '',
	housing_name text not null default '',
	complex_name text not null default '',
	building_name text not null default '',
	unit_no text,
	floor integer,
	floor_no integer,
	unit_type text not null default '',
	structure_type text not null default '',
	elevator_installed boolean,
	exclusive_area_m2 numeric,
	area_pyeong numeric,
	deposit_text text not null default '',
	deposit_amount numeric,
	deposit_krw bigint,
	jeonse_deposit_text text not null default '',
	jeonse_deposit_krw bigint,
	monthly_rent_text text not null default '',
	monthly_rent_amount numeric,
	monthly_rent_krw bigint,
	supply_count integer,
	direction text not null default '',
	status text not null default '',
	source_sheet text not null default '',
	source_row integer,
	source_cell text not null default '',
	source_page integer,
	source_span text not null default '',
	raw_row jsonb not null default '{}'::jsonb,
	confidence numeric not null default 0,
	qa_status text not null default 'pending',
	created_at timestamptz not null default now()
);

alter table if exists offerings add column if not exists source_artifact_id bigint references extracted_artifacts(id);
alter table if exists offerings add column if not exists offering_type text not null default 'unit';
alter table if exists offerings add column if not exists supply_category text not null default '';
alter table if exists offerings add column if not exists list_no text not null default '';
alter table if exists offerings add column if not exists district text not null default '';
alter table if exists offerings add column if not exists address text not null default '';
alter table if exists offerings add column if not exists legal_dong text not null default '';
alter table if exists offerings add column if not exists address_detail text not null default '';
alter table if exists offerings add column if not exists housing_name text not null default '';
alter table if exists offerings add column if not exists floor_no integer;
alter table if exists offerings add column if not exists structure_type text not null default '';
alter table if exists offerings add column if not exists elevator_installed boolean;
alter table if exists offerings add column if not exists deposit_krw bigint;
alter table if exists offerings add column if not exists jeonse_deposit_text text not null default '';
alter table if exists offerings add column if not exists jeonse_deposit_krw bigint;
alter table if exists offerings add column if not exists monthly_rent_krw bigint;
alter table if exists offerings add column if not exists source_sheet text not null default '';
alter table if exists offerings add column if not exists source_row integer;
alter table if exists offerings add column if not exists source_cell text not null default '';
alter table if exists offerings add column if not exists source_page integer;
alter table if exists offerings add column if not exists source_span text not null default '';
alter table if exists offerings add column if not exists qa_status text not null default 'pending';
alter table if exists offerings alter column unit_no drop not null;

drop index if exists idx_housing_units_notice_id;
drop index if exists idx_housing_units_attachment_id;
drop index if exists idx_housing_units_source_artifact_id;
drop index if exists idx_housing_units_area;
drop index if exists idx_housing_units_rent;
drop index if exists idx_housing_units_address;
drop index if exists idx_housing_units_qa_status;
drop index if exists idx_housing_units_raw_row;
drop index if exists uq_housing_units_attachment_source_span;
drop index if exists uq_housing_units_source_identity;

create index if not exists idx_offerings_notice_id on offerings(notice_id);
create index if not exists idx_offerings_attachment_id on offerings(attachment_id);
create index if not exists idx_offerings_source_artifact_id on offerings(source_artifact_id);
create index if not exists idx_offerings_type on offerings(offering_type);
create index if not exists idx_offerings_area on offerings(exclusive_area_m2);
create index if not exists idx_offerings_rent on offerings(deposit_amount, monthly_rent_amount);
create index if not exists idx_offerings_address on offerings(district, legal_dong);
create index if not exists idx_offerings_qa_status on offerings(qa_status);
create index if not exists idx_offerings_raw_row on offerings using gin(raw_row);
create unique index if not exists uq_offerings_attachment_source_span
	on offerings(attachment_id, source_span)
	where source_span <> '';
create unique index if not exists uq_offerings_source_identity
	on offerings(notice_id, attachment_id, coalesce(source_row, -1), coalesce(unit_no, ''), coalesce(nullif(housing_name, ''), complex_name))
	where qa_status = 'approved';

create table if not exists rent_conversion_rules (
	id bigserial primary key,
	notice_id bigint not null references source_notices(id),
	source_artifact_id bigint references extracted_artifacts(id),
	rent_to_deposit_allowed boolean not null default false,
	rent_to_deposit_max_ratio numeric,
	rent_to_deposit_annual_rate numeric,
	rent_unit_krw bigint,
	deposit_per_rent_unit_krw bigint,
	deposit_to_rent_allowed boolean not null default false,
	application_cycle text not null default '',
	reapply_limit text not null default '',
	application_method text not null default '',
	source_text text not null default '',
	source_span text not null default '',
	confidence numeric not null default 0,
	created_at timestamptz not null default now()
);

create index if not exists idx_rent_conversion_rules_notice_id on rent_conversion_rules(notice_id);

create table if not exists offering_conversion_estimates (
	id bigserial primary key,
	offering_id bigint not null references offerings(id),
	rent_conversion_rule_id bigint not null references rent_conversion_rules(id),
	max_convertible_rent_krw bigint,
	estimated_additional_deposit_krw bigint,
	estimated_min_monthly_rent_krw bigint,
	calculation_version text not null default 'v1',
	source_span text not null default '',
	created_at timestamptz not null default now(),
	unique (offering_id, rent_conversion_rule_id, calculation_version)
);

drop index if exists idx_housing_unit_conversion_estimates_unit_id;
create index if not exists idx_offering_conversion_estimates_offering_id on offering_conversion_estimates(offering_id);

create table if not exists notice_schedules (
	id bigserial primary key,
	notice_id bigint not null references source_notices(id),
	source_artifact_id bigint references extracted_artifacts(id),
	schedule_type text not null,
	label text not null default '',
	starts_at timestamptz,
	ends_at timestamptz,
	date_text text not null default '',
	channel text not null default '',
	note text not null default '',
	source_text text not null default '',
	source_span text not null default '',
	confidence numeric not null default 0,
	created_at timestamptz not null default now()
);

create index if not exists idx_notice_schedules_notice_id on notice_schedules(notice_id);
create index if not exists idx_notice_schedules_type on notice_schedules(schedule_type);
create index if not exists idx_notice_schedules_starts_at on notice_schedules(starts_at);

create table if not exists llm_repair_attempts (
	id bigserial primary key,
	artifact_id bigint not null references extracted_artifacts(id),
	schema_version text not null,
	prompt_version text not null,
	model text not null,
	input_hash text not null,
	output_hash text not null default '',
	status text not null,
	confidence numeric not null default 0,
	source_span text not null default '',
	request_json jsonb not null default '{}'::jsonb,
	response_json jsonb not null default '{}'::jsonb,
	error_text text not null default '',
	created_at timestamptz not null default now()
);

create index if not exists idx_llm_repair_attempts_artifact_id on llm_repair_attempts(artifact_id);
create index if not exists idx_llm_repair_attempts_status on llm_repair_attempts(status);

create table if not exists qa_decisions (
	id bigserial primary key,
	target_type text not null,
	target_id bigint not null,
	decision text not null,
	reason text not null default '',
	checks_json jsonb not null default '{}'::jsonb,
	decided_at timestamptz not null default now(),
	decided_by text not null default 'system'
);

create index if not exists idx_qa_decisions_target on qa_decisions(target_type, target_id);
create index if not exists idx_qa_decisions_decision on qa_decisions(decision);

comment on table collection_runs is 'Pipeline execution record for discovery, extraction, normalization, and QA runs.';
comment on column collection_runs.stats is 'Run-level counters and reports. This is operational metadata, not the serving data model.';

comment on table source_boards is 'Agency board registry. SH rental and sale boards are separate board_kind values.';
comment on column source_boards.agency is 'Source agency such as SH or LH.';
comment on column source_boards.board_kind is 'Board code such as rental or sale.';

comment on table raw_documents is 'Fetched HTTP documents kept as audit and reprocessing evidence.';
comment on column raw_documents.body_json is 'Optional parsed JSON body for API responses. Not a primary normalized query model.';

comment on table stored_objects is 'ObjectStore records for original attachments and generated artifacts.';
comment on column stored_objects.storage_backend is 'Logical backend name such as local_filesystem or s3_compatible. Domain logic must depend on ObjectStore, not this implementation detail.';
comment on column stored_objects.object_key is 'Portable logical object key independent of local path or MinIO bucket layout.';
comment on column stored_objects.sha256 is 'Checksum used for integrity checks and deduplication.';

comment on table source_notices is 'Recruitment-family notices that passed discovery gating.';
comment on column source_notices.category is 'Legacy prototype category retained for compatibility.';
comment on column source_notices.notice_type is 'Canonical notice type. HIC persists recruitment-family notices only.';
comment on column source_notices.notice_subtype is 'Correction, additional recruitment, remaining-unit recruitment, or empty when not applicable.';
comment on column source_notices.seq is 'Source site notice sequence. SH examples are numeric strings.';

comment on table attachments is 'Attachment metadata for persisted recruitment-family notices.';
comment on column attachments.stored_object_id is 'ObjectStore record for the preserved original attachment.';
comment on column attachments.attachment_kind is 'Classified attachment role such as notice_pdf, offering_list_xlsx, schedule_pdf, applicant_or_winner_file, application_form, or unsupported.';
comment on column attachments.storage_path is 'Legacy local prototype path retained for compatibility until ObjectStore migration is complete.';

comment on table attachment_extractions is 'Legacy prototype extraction rows retained for compatibility.';
comment on table extracted_artifacts is 'Mechanical extraction outputs from PDF, XLSX, HWP, or HTML before normalization.';
comment on column extracted_artifacts.artifact_type is 'Artifact kind such as xlsx_row, pdf_text, html_preview, hwp_unsupported, or table_candidate.';
comment on column extracted_artifacts.source_span is 'Machine-readable or human-readable pointer to the source page, sheet, row, cell, or HTML selector.';
comment on column extracted_artifacts.content_json is 'Extractor payload for audit and reprocessing. Serving APIs should use normalized tables.';

comment on table offerings is 'Normalized offering records extracted from recruitment notices. An offering may represent one known unit or a group whose exact unit numbers are assigned later.';
comment on column offerings.offering_type is 'Offering grain: unit for a specific unit or room, group for grouped supply by complex/type/area/count.';
comment on column offerings.supply_category is 'Supply category such as 신규공급 or 재공급 when provided by the source.';
comment on column offerings.list_no is 'Original row/list number from an offering list attachment.';
comment on column offerings.district is 'Seoul district extracted from the address, such as 영등포구.';
comment on column offerings.address is 'Full normalized address. This must not be left only in raw_row.';
comment on column offerings.legal_dong is 'Legal dong candidate parsed from the address.';
comment on column offerings.housing_name is 'Source housing name. This may overlap with complex_name in SH spreadsheets.';
comment on column offerings.unit_no is 'Unit or room number inside a building. Null for group offerings whose unit numbers are assigned later.';
comment on column offerings.floor_no is 'Floor number when the source provides it or it can be inferred from unit_no.';
comment on column offerings.deposit_krw is 'Normalized deposit amount in KRW.';
comment on column offerings.jeonse_deposit_krw is 'Normalized jeonse deposit amount in KRW when the offering is deposit-only.';
comment on column offerings.monthly_rent_krw is 'Normalized monthly rent amount in KRW.';
comment on column offerings.supply_count is 'Number of units represented by this offering. Unit offerings usually represent one unit; group offerings may represent many.';
comment on column offerings.source_span is 'Evidence pointer for the normalized offering record.';
comment on column offerings.qa_status is 'Promotion state for serving. Only QA-approved records should be exposed by default.';
comment on column offerings.raw_row is 'Raw row evidence for audit and reprocessing, not the primary query model.';

comment on table rent_conversion_rules is 'Notice-level rent conversion rules extracted from notice text or attachments.';
comment on column rent_conversion_rules.rent_to_deposit_max_ratio is 'Maximum monthly rent ratio that can be converted into additional deposit. Example: 0.60.';
comment on column rent_conversion_rules.rent_to_deposit_annual_rate is 'Annual conversion rate applied when monthly rent is converted to deposit. Example: 0.067.';
comment on column rent_conversion_rules.deposit_per_rent_unit_krw is 'Additional deposit required per rent_unit_krw reduction in monthly rent.';

comment on table offering_conversion_estimates is 'Per-offering calculated estimates derived from rent conversion rules.';
comment on column offering_conversion_estimates.estimated_min_monthly_rent_krw is 'Estimated minimum monthly rent after maximum rent-to-deposit conversion.';

comment on table notice_schedules is 'Structured notice schedules such as application, document submission, winner announcement, contract, and move-in.';
comment on column notice_schedules.date_text is 'Original date expression retained for review when parsed timestamps are partial or ambiguous.';
comment on column notice_schedules.source_span is 'Evidence pointer for the schedule extraction.';

comment on table llm_repair_attempts is 'Constrained LLM repair attempts for low-confidence deterministic extraction segments.';
comment on column llm_repair_attempts.input_hash is 'Hash of the exact LLM input segment.';
comment on column llm_repair_attempts.source_span is 'Required source evidence for accepting LLM output.';

comment on table qa_decisions is 'QA gate decisions controlling promotion to serving/API models.';
comment on column qa_decisions.decision is 'Decision value such as approved, rejected, quarantined, or needs_review.';
comment on column qa_decisions.checks_json is 'Structured QA check results with exact failure reasons and measured values.';
