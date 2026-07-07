-- Hefai — initial schema (migration 0001)
--
-- Design principles:
--   * UUID primary keys (gen_random_uuid, PG 13+), timestamptz audit columns.
--   * Postgres ENUM types for closed state machines; free text where the
--     domain is open-ended (e.g. budget categories).
--   * Money as NUMERIC(12,2) + ISO currency (default DKK).
--   * Cross-entity document attachment via document_links with one nullable
--     FK per target and a CHECK enforcing exactly one — full referential
--     integrity, no polymorphic (entity_type, entity_id) pairs.
--   * Versioning: drawings and structural packages carry explicit version
--     rows; generated documents and calculation estimates are immutable and
--     superseded by new rows.
--   * RAG: source_documents/source_chunks with a pgvector column of
--     unspecified dimension — the dimension (and its index) is fixed in a
--     later migration once the embedding provider is chosen.

CREATE EXTENSION IF NOT EXISTS vector;

-- ---------------------------------------------------------------------------
-- Enums
-- ---------------------------------------------------------------------------

CREATE TYPE project_kind    AS ENUM ('new_build', 'renovation', 'extension', 'other');
CREATE TYPE project_status  AS ENUM ('planning', 'in_progress', 'on_hold', 'completed', 'archived');
CREATE TYPE project_role    AS ENUM ('owner', 'member', 'viewer');

CREATE TYPE phase_status    AS ENUM ('not_started', 'in_progress', 'completed');
CREATE TYPE task_status     AS ENUM ('todo', 'blocked', 'in_progress', 'done', 'cancelled');

CREATE TYPE material_status AS ENUM ('needed', 'ordered', 'delivered', 'in_stock', 'used');

CREATE TYPE room_kind       AS ENUM ('room', 'zone', 'outdoor');

CREATE TYPE document_kind   AS ENUM (
    'architect_drawing', 'construction_drawing', 'receipt', 'photo',
    'warranty', 'datasheet', 'permit', 'correspondence', 'generated', 'other');

-- Byggesag ------------------------------------------------------------------

CREATE TYPE case_type   AS ENUM ('unknown', 'notification', 'building_permit'); -- anmeldelse / byggetilladelse
CREATE TYPE case_status AS ENUM (
    'draft', 'ready_for_submission', 'submitted', 'awaiting_response',
    'questions_from_municipality', 'approved', 'rejected', 'closed');

CREATE TYPE case_event_type          AS ENUM ('status_change', 'correspondence', 'note', 'submission');
CREATE TYPE correspondence_direction AS ENUM ('incoming', 'outgoing', 'internal');

CREATE TYPE drawing_kind AS ENUM ('site_plan', 'floor_plan', 'elevation', 'section', 'detail', 'other');

CREATE TYPE generated_document_kind AS ENUM (
    'site_plan', 'floor_plan', 'elevation', 'area_statement',
    'project_description', 'application_summary', 'structural_package', 'other');
CREATE TYPE generated_document_status AS ENUM ('draft', 'final');

CREATE TYPE compliance_status AS ENUM ('not_checked', 'ok', 'attention', 'needs_confirmation', 'confirmed');

-- RAG / kildemateriale --------------------------------------------------------

CREATE TYPE source_kind   AS ENUM ('br18', 'eurocode', 'local_plan', 'municipal_guidance', 'other');
CREATE TYPE source_status AS ENUM ('processing', 'ready', 'failed');

-- Statik ----------------------------------------------------------------------

CREATE TYPE structural_element_type AS ENUM ('beam', 'column', 'wall', 'foundation', 'roof', 'slab', 'other');
CREATE TYPE structural_material     AS ENUM ('timber', 'steel', 'concrete', 'masonry', 'other');

CREATE TYPE load_type   AS ENUM ('dead', 'live', 'snow', 'wind', 'point', 'line', 'other');
CREATE TYPE load_status AS ENUM ('assumed', 'engineer_confirmed', 'engineer_changed');

CREATE TYPE estimate_status AS ENUM ('advisory', 'verified', 'superseded', 'rejected');

CREATE TYPE package_status AS ENUM ('draft', 'sent', 'reviewed');
CREATE TYPE review_status  AS ENUM ('approved', 'approved_with_changes', 'rejected', 'partial');
CREATE TYPE review_verdict AS ENUM ('approved', 'changed', 'rejected', 'comment');

-- ---------------------------------------------------------------------------
-- Users & projects
-- ---------------------------------------------------------------------------

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         text NOT NULL UNIQUE,
    display_name  text NOT NULL,
    password_hash text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE projects (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name         text NOT NULL,
    description  text NOT NULL DEFAULT '',
    kind         project_kind   NOT NULL DEFAULT 'other',
    status       project_status NOT NULL DEFAULT 'planning',
    address      text NOT NULL DEFAULT '',
    municipality text NOT NULL DEFAULT '',        -- kommune; styrer bl.a. last-zoner og sagsbehandling
    cadastral_id text NOT NULL DEFAULT '',        -- matrikelnummer
    plot_area_m2 numeric(10,2),
    created_by   uuid NOT NULL REFERENCES users (id),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE project_members (
    project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users (id)    ON DELETE CASCADE,
    role       project_role NOT NULL DEFAULT 'member',
    created_at timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, user_id)
);

-- ---------------------------------------------------------------------------
-- Modul 1 — projekt & proces
-- ---------------------------------------------------------------------------

CREATE TABLE phases (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name          text NOT NULL,                  -- fx 'Grund/fundament', 'Råhus', 'Tag' …
    description   text NOT NULL DEFAULT '',
    sort_order    integer NOT NULL DEFAULT 0,
    status        phase_status NOT NULL DEFAULT 'not_started',
    planned_start date,
    planned_end   date,
    actual_start  date,
    actual_end    date,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX phases_project_idx ON phases (project_id, sort_order);

CREATE TABLE rooms (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name        text NOT NULL,                    -- 'Badeværelse', 'Nordfacade', 'Have' …
    kind        room_kind NOT NULL DEFAULT 'room',
    description text NOT NULL DEFAULT '',
    area_m2     numeric(8,2),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX rooms_project_idx ON rooms (project_id);

CREATE TABLE suppliers (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id     uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    company_name   text NOT NULL,
    contact_person text NOT NULL DEFAULT '',
    trade          text NOT NULL DEFAULT '',      -- fag: tømrer, VVS, el …
    phone          text NOT NULL DEFAULT '',
    email          text NOT NULL DEFAULT '',
    notes          text NOT NULL DEFAULT '',
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX suppliers_project_idx ON suppliers (project_id);

CREATE TABLE tasks (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    phase_id            uuid REFERENCES phases (id)    ON DELETE SET NULL,
    room_id             uuid REFERENCES rooms (id)     ON DELETE SET NULL,
    title               text NOT NULL,
    description         text NOT NULL DEFAULT '',
    status              task_status NOT NULL DEFAULT 'todo',
    -- Ansvarlig er enten en bruger (mig) eller en leverandør/håndværker:
    responsible_user_id     uuid REFERENCES users (id)     ON DELETE SET NULL,
    responsible_supplier_id uuid REFERENCES suppliers (id) ON DELETE SET NULL,
    planned_start       date,
    planned_end         date,
    actual_start        date,
    actual_end          date,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tasks_one_responsible CHECK (responsible_user_id IS NULL OR responsible_supplier_id IS NULL)
);
CREATE INDEX tasks_project_idx ON tasks (project_id, status);
CREATE INDEX tasks_phase_idx   ON tasks (phase_id);

-- X kan først starte når Y er færdig. Cykel-forebyggelse håndhæves i
-- service-laget (grafvalidering ved insert).
CREATE TABLE task_dependencies (
    task_id            uuid NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    depends_on_task_id uuid NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    created_at         timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (task_id, depends_on_task_id),
    CONSTRAINT no_self_dependency CHECK (task_id <> depends_on_task_id)
);
CREATE INDEX task_dependencies_reverse_idx ON task_dependencies (depends_on_task_id);

CREATE TABLE budget_items (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    phase_id         uuid REFERENCES phases (id) ON DELETE SET NULL,
    category         text NOT NULL DEFAULT '',   -- fri kategori: 'Materialer', 'Håndværker', 'Gebyrer' …
    description      text NOT NULL,
    estimated_amount numeric(12,2) NOT NULL DEFAULT 0,
    currency         char(3) NOT NULL DEFAULT 'DKK',
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX budget_items_project_idx ON budget_items (project_id);

CREATE TABLE expenses (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id     uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    budget_item_id uuid REFERENCES budget_items (id) ON DELETE SET NULL,
    phase_id       uuid REFERENCES phases (id)       ON DELETE SET NULL,
    supplier_id    uuid REFERENCES suppliers (id)    ON DELETE SET NULL,
    description    text NOT NULL,
    amount         numeric(12,2) NOT NULL,
    currency       char(3) NOT NULL DEFAULT 'DKK',
    incurred_on    date NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
    -- kvittering knyttes via document_links (target expense_id)
);
CREATE INDEX expenses_project_idx ON expenses (project_id, incurred_on);

CREATE TABLE materials (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    phase_id    uuid REFERENCES phases (id)    ON DELETE SET NULL,
    task_id     uuid REFERENCES tasks (id)     ON DELETE SET NULL,
    room_id     uuid REFERENCES rooms (id)     ON DELETE SET NULL,
    supplier_id uuid REFERENCES suppliers (id) ON DELETE SET NULL,
    name        text NOT NULL,
    spec        text NOT NULL DEFAULT '',       -- dimension/kvalitet, fx '45x195 C24'
    quantity    numeric(12,3) NOT NULL DEFAULT 0,
    unit        text NOT NULL DEFAULT 'stk',    -- stk, m, m², m³, kg …
    unit_price  numeric(12,2),
    currency    char(3) NOT NULL DEFAULT 'DKK',
    status      material_status NOT NULL DEFAULT 'needed',
    notes       text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX materials_project_idx ON materials (project_id, status);

-- ---------------------------------------------------------------------------
-- Dokumenter & arkiv
-- ---------------------------------------------------------------------------

CREATE TABLE documents (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    uploaded_by  uuid REFERENCES users (id) ON DELETE SET NULL,
    kind         document_kind NOT NULL DEFAULT 'other',
    title        text NOT NULL,
    description  text NOT NULL DEFAULT '',
    filename     text NOT NULL,
    storage_key  text NOT NULL,                  -- sti/nøgle i filstore (lokal disk/S3-kompatibel)
    mime_type    text NOT NULL,
    size_bytes   bigint NOT NULL DEFAULT 0,
    captured_at  timestamptz,                    -- til billeder: hvornår er det taget
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX documents_project_idx ON documents (project_id, kind);

CREATE TABLE tags (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name       text NOT NULL,
    UNIQUE (project_id, name)
);

CREATE TABLE document_tags (
    document_id uuid NOT NULL REFERENCES documents (id) ON DELETE CASCADE,
    tag_id      uuid NOT NULL REFERENCES tags (id)      ON DELETE CASCADE,
    PRIMARY KEY (document_id, tag_id)
);

-- ---------------------------------------------------------------------------
-- Modul 2 — byggesag
-- ---------------------------------------------------------------------------

CREATE TABLE case_files (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id            uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    title                 text NOT NULL,
    description           text NOT NULL DEFAULT '',   -- fritekst-beskrivelse af det ønskede byggeri
    case_type             case_type   NOT NULL DEFAULT 'unknown',
    status                case_status NOT NULL DEFAULT 'draft',
    municipal_case_number text NOT NULL DEFAULT '',
    submitted_at          timestamptz,
    decided_at            timestamptz,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX case_files_project_idx ON case_files (project_id);

-- Tidslinje + korrespondance-log for sagen.
CREATE TABLE case_events (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    case_file_id uuid NOT NULL REFERENCES case_files (id) ON DELETE CASCADE,
    event_type   case_event_type NOT NULL,
    direction    correspondence_direction,
    occurred_at  timestamptz NOT NULL DEFAULT now(),
    summary      text NOT NULL,
    body         text NOT NULL DEFAULT '',
    document_id  uuid REFERENCES documents (id) ON DELETE SET NULL,
    created_by   uuid REFERENCES users (id)     ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX case_events_case_idx ON case_events (case_file_id, occurred_at);

-- En tegning er en identitet; indholdet versioneres i drawing_versions.
CREATE TABLE drawings (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    case_file_id uuid REFERENCES case_files (id) ON DELETE SET NULL,
    kind         drawing_kind NOT NULL,
    title        text NOT NULL,
    created_by   uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX drawings_project_idx ON drawings (project_id);

CREATE TABLE drawing_versions (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    drawing_id  uuid NOT NULL REFERENCES drawings (id) ON DELETE CASCADE,
    version_no  integer NOT NULL,
    -- Hele 2D-modellen fra tegnefladen: vægge, rum, mål, døre/vinduer,
    -- placering på grund. Skema for JSON'en defineres og valideres i Go.
    data        jsonb NOT NULL,
    scale       text NOT NULL DEFAULT '1:100',
    note        text NOT NULL DEFAULT '',
    created_by  uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (drawing_id, version_no)
);

-- Genererede PDF'er (situationsplan, arealopgørelse, ansøgning …).
-- Immutable: en ny generering giver en ny række med højere version_no.
CREATE TABLE generated_documents (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id     uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    case_file_id   uuid REFERENCES case_files (id) ON DELETE SET NULL,
    kind           generated_document_kind NOT NULL,
    status         generated_document_status NOT NULL DEFAULT 'draft',
    version_no     integer NOT NULL DEFAULT 1,
    -- Snapshot af input-data på genereringstidspunktet (reproducerbarhed):
    input_snapshot jsonb NOT NULL DEFAULT '{}',
    document_id    uuid REFERENCES documents (id) ON DELETE SET NULL, -- den renderede PDF
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX generated_documents_case_idx ON generated_documents (case_file_id);

-- ---------------------------------------------------------------------------
-- RAG-kildemateriale (BR18, lokalplan, kommunens krav)
-- ---------------------------------------------------------------------------

-- project_id NULL = globalt bibliotek (fx BR18 deles på tværs af projekter).
CREATE TABLE source_documents (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    uuid REFERENCES projects (id) ON DELETE CASCADE,
    document_id   uuid REFERENCES documents (id) ON DELETE SET NULL, -- evt. uploadet fil
    title         text NOT NULL,
    kind          source_kind NOT NULL,
    version_label text NOT NULL DEFAULT '',     -- fx 'BR18 pr. 2026-01-01'
    url           text NOT NULL DEFAULT '',
    status        source_status NOT NULL DEFAULT 'processing',
    added_by      uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE source_chunks (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_document_id uuid NOT NULL REFERENCES source_documents (id) ON DELETE CASCADE,
    chunk_index        integer NOT NULL,
    content            text NOT NULL,
    section_ref        text NOT NULL DEFAULT '', -- fx 'BR18 §180' eller 'Lokalplan 42, §6.2'
    page_no            integer,
    -- Dimension fastlægges i senere migration når embedding-provider vælges;
    -- indtil da fungerer fuldtekstsøgning som fallback.
    embedding          vector,
    embedding_model    text,
    UNIQUE (source_document_id, chunk_index)
);
CREATE INDEX source_chunks_fts_idx ON source_chunks
    USING gin (to_tsvector('danish', content));

-- Ikke-bindende egenkontrol-tjekliste. Grounding: hvert punkt kan pege på den
-- kilde-chunk det bygger på — punkter uden kilde skal i UI fremstå som
-- "kræver bekræftelse".
CREATE TABLE compliance_check_items (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    case_file_id    uuid NOT NULL REFERENCES case_files (id) ON DELETE CASCADE,
    category        text NOT NULL DEFAULT '',    -- 'skelafstand', 'højde', 'bebyggelsesprocent' …
    requirement     text NOT NULL,               -- kravet med ordlyd
    expected_value  text NOT NULL DEFAULT '',    -- fx 'min. 2,5 m'
    actual_value    text NOT NULL DEFAULT '',    -- fx '3,1 m (fra tegning v3)'
    status          compliance_status NOT NULL DEFAULT 'not_checked',
    source_chunk_id uuid REFERENCES source_chunks (id) ON DELETE SET NULL,
    note            text NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX compliance_check_items_case_idx ON compliance_check_items (case_file_id);

-- ---------------------------------------------------------------------------
-- Modul 3 — statiker-forberedelse
-- ---------------------------------------------------------------------------

CREATE TABLE structural_elements (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    room_id         uuid REFERENCES rooms (id)    ON DELETE SET NULL,
    drawing_id      uuid REFERENCES drawings (id) ON DELETE SET NULL, -- hvor elementet er tegnet
    element_type    structural_element_type NOT NULL,
    name            text NOT NULL,               -- 'Bjælke over stue', 'Nordvæg' …
    is_load_bearing boolean NOT NULL DEFAULT true,
    material        structural_material NOT NULL,
    material_spec   text NOT NULL DEFAULT '',    -- 'C24', 'S235', 'C25/30' …
    -- Geometri afhænger af elementtype (spændvidde, tværsnit, højde, tykkelse
    -- …). JSON-skemaet pr. elementtype defineres og valideres i Go.
    geometry        jsonb NOT NULL DEFAULT '{}',
    notes           text NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX structural_elements_project_idx ON structural_elements (project_id);

-- Laster. structural_element_id NULL = projekt-niveau (fx snelastzone for
-- grunden), ellers elementspecifik.
CREATE TABLE loads (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id            uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    structural_element_id uuid REFERENCES structural_elements (id) ON DELETE CASCADE,
    load_type             load_type NOT NULL,
    value                 numeric(12,4) NOT NULL,
    unit                  text NOT NULL,          -- 'kN/m²', 'kN/m', 'kN'
    standard_reference    text NOT NULL DEFAULT '', -- fx 'EN 1991-1-3 + DK NA'
    -- Hvordan værdien er udledt (zone, terrænkategori, formtal …):
    derivation            jsonb NOT NULL DEFAULT '{}',
    status                load_status NOT NULL DEFAULT 'assumed',
    notes                 text NOT NULL DEFAULT '',
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX loads_project_idx ON loads (project_id);
CREATE INDEX loads_element_idx ON loads (structural_element_id);

-- Vejledende beregninger fra deterministisk Go-kode. Immutable: genberegning
-- giver en ny række; den gamle markeres 'superseded'.
CREATE TABLE calculation_estimates (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id            uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    structural_element_id uuid REFERENCES structural_elements (id) ON DELETE CASCADE,
    method                text NOT NULL,          -- fx 'timber_beam_udl_v1' — peger på Go-implementering
    method_version        text NOT NULL DEFAULT '1',
    standard_reference    text NOT NULL,          -- fx 'EN 1995-1-1 (Eurocode 5) + DK NA'
    inputs                jsonb NOT NULL,         -- alle inputværdier
    assumptions           jsonb NOT NULL,         -- eksplicitte antagelser, vises altid i UI/PDF
    results               jsonb NOT NULL,         -- resultater inkl. udnyttelsesgrader
    status                estimate_status NOT NULL DEFAULT 'advisory',
    notes                 text NOT NULL DEFAULT '',
    created_at            timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX calculation_estimates_element_idx ON calculation_estimates (structural_element_id);

-- Statiker-pakken: versioneret eksport af elementer + laster + beregninger.
CREATE TABLE structural_packages (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    version_no  integer NOT NULL,
    title       text NOT NULL,
    -- Snapshot af hvilke elementer/laster/beregninger/tegningsversioner der
    -- indgik (id'er + hash), så reviewet altid kan føres tilbage til præcis
    -- det materiale statikeren så:
    snapshot    jsonb NOT NULL,
    document_id uuid REFERENCES documents (id) ON DELETE SET NULL, -- den samlede PDF
    status      package_status NOT NULL DEFAULT 'draft',
    sent_at     timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, version_no)
);

-- Statikerens svar, registreret manuelt (offline-loop i v1).
CREATE TABLE engineer_reviews (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    structural_package_id uuid NOT NULL REFERENCES structural_packages (id) ON DELETE CASCADE,
    reviewer_name        text NOT NULL,
    reviewer_company     text NOT NULL DEFAULT '',
    reviewer_credentials text NOT NULL DEFAULT '', -- fx 'Anerkendt statiker' / certificeringsklasse
    received_at          date NOT NULL,
    overall_status       review_status NOT NULL,
    summary              text NOT NULL DEFAULT '',
    response_document_id uuid REFERENCES documents (id) ON DELETE SET NULL, -- statikerens returnerede dokument
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now()
);

-- Punkt-for-punkt-feedback: hvad er OK, hvad skal ændres. Højst ét mål pr.
-- række; alle NULL = generel kommentar. Opdateringer af selve elementet/
-- lasten/beregningen sker som nye rækker/statusskift i deres egne tabeller,
-- så systemet kan vise bekræftede vs. ændrede antagelser.
CREATE TABLE engineer_review_items (
    id                      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    engineer_review_id      uuid NOT NULL REFERENCES engineer_reviews (id) ON DELETE CASCADE,
    structural_element_id   uuid REFERENCES structural_elements (id)   ON DELETE SET NULL,
    load_id                 uuid REFERENCES loads (id)                 ON DELETE SET NULL,
    calculation_estimate_id uuid REFERENCES calculation_estimates (id) ON DELETE SET NULL,
    drawing_id              uuid REFERENCES drawings (id)              ON DELETE SET NULL,
    verdict                 review_verdict NOT NULL,
    comment                 text NOT NULL DEFAULT '',
    corrected_values        jsonb NOT NULL DEFAULT '{}', -- statikerens rettede værdier
    created_at              timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT review_item_at_most_one_target CHECK (
        (structural_element_id   IS NOT NULL)::int +
        (load_id                 IS NOT NULL)::int +
        (calculation_estimate_id IS NOT NULL)::int +
        (drawing_id              IS NOT NULL)::int <= 1
    )
);
CREATE INDEX engineer_review_items_review_idx ON engineer_review_items (engineer_review_id);

-- ---------------------------------------------------------------------------
-- Dokument-tilknytning på tværs af moduler
-- ---------------------------------------------------------------------------

-- Ét dokument kan knyttes til mange entiteter; hver række peger på præcis én.
-- Nullable FK'er + CHECK giver fuld referentiel integritet (frem for
-- entity_type/entity_id uden FK).
CREATE TABLE document_links (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id           uuid NOT NULL REFERENCES documents (id) ON DELETE CASCADE,
    phase_id              uuid REFERENCES phases (id)              ON DELETE CASCADE,
    task_id               uuid REFERENCES tasks (id)               ON DELETE CASCADE,
    room_id               uuid REFERENCES rooms (id)               ON DELETE CASCADE,
    expense_id            uuid REFERENCES expenses (id)            ON DELETE CASCADE,
    material_id           uuid REFERENCES materials (id)           ON DELETE CASCADE,
    supplier_id           uuid REFERENCES suppliers (id)           ON DELETE CASCADE,
    case_file_id          uuid REFERENCES case_files (id)          ON DELETE CASCADE,
    structural_element_id uuid REFERENCES structural_elements (id) ON DELETE CASCADE,
    created_at            timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT document_link_exactly_one_target CHECK (
        (phase_id              IS NOT NULL)::int +
        (task_id               IS NOT NULL)::int +
        (room_id               IS NOT NULL)::int +
        (expense_id            IS NOT NULL)::int +
        (material_id           IS NOT NULL)::int +
        (supplier_id           IS NOT NULL)::int +
        (case_file_id          IS NOT NULL)::int +
        (structural_element_id IS NOT NULL)::int = 1
    )
);
CREATE INDEX document_links_document_idx ON document_links (document_id);
CREATE INDEX document_links_task_idx     ON document_links (task_id)     WHERE task_id     IS NOT NULL;
CREATE INDEX document_links_room_idx     ON document_links (room_id)     WHERE room_id     IS NOT NULL;
CREATE INDEX document_links_case_idx     ON document_links (case_file_id) WHERE case_file_id IS NOT NULL;

-- ---------------------------------------------------------------------------
-- updated_at-trigger
-- ---------------------------------------------------------------------------

CREATE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
DECLARE
    t text;
BEGIN
    FOREACH t IN ARRAY ARRAY[
        'users', 'projects', 'phases', 'rooms', 'suppliers', 'tasks',
        'budget_items', 'expenses', 'materials', 'documents', 'case_files',
        'drawings', 'source_documents', 'compliance_check_items',
        'structural_elements', 'loads', 'engineer_reviews'
    ] LOOP
        EXECUTE format(
            'CREATE TRIGGER %I_set_updated_at BEFORE UPDATE ON %I
             FOR EACH ROW EXECUTE FUNCTION set_updated_at()', t, t);
    END LOOP;
END;
$$;
