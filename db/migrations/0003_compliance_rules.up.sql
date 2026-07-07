-- Strukturerede lovkrav til den automatiske egenkontrol. Hver regel er en
-- målbar grænseværdi (fx maks. bebyggelsesprocent) med kildehenvisning til
-- det indlæste materiale. Regler foreslås af AI eller indtastes manuelt og
-- er først aktive når brugeren har bekræftet dem.

CREATE TYPE rule_status AS ENUM ('suggested', 'confirmed', 'rejected');

CREATE TABLE compliance_rules (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    parameter       text NOT NULL,          -- fx 'max_bebyggelsesprocent' — katalog i Go
    value           numeric(12,3) NOT NULL, -- grænseværdien
    source_chunk_id uuid REFERENCES source_chunks (id) ON DELETE SET NULL,
    quote           text NOT NULL DEFAULT '', -- ordret citat fra kilden
    status          rule_status NOT NULL DEFAULT 'suggested',
    note            text NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, parameter)
);

CREATE TRIGGER compliance_rules_set_updated_at BEFORE UPDATE ON compliance_rules
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
