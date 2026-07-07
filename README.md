# Hefai

Byggeprojekt-platform der dækker hele processen med at bygge eller renovere et
hus — fra planlægning over kommunal byggesag til statiker-forberedelse. Navnet
kommer fra Hefaistos, smedeguden og byggeren.

Hefai forbereder, strukturerer og sparer tid — men træffer aldrig
myndighedsafgørelser og erstatter ikke en autoriseret statiker eller rådgiver.
Hvor faglig godkendelse kræves, producerer Hefai et gennemarbejdet udgangspunkt,
tydeligt markeret som kladde der kræver godkendelse.

## Stack

- **Backend:** Go + PostgreSQL. Lagdelt: domænemodeller → repositories →
  services → HTTP/JSON-handlers.
- **Frontend:** React + TypeScript.
- **Beregninger:** deterministisk, testbar Go-kode med eksplicitte formler og
  standardreferencer (Eurocodes/BR18) — aldrig LLM-genererede tal.
- **AI-assistent:** RAG over indlæst kildemateriale (BR18, lokalplan,
  kommunekrav) via pgvector. Provider-agnostisk LLM-interface.
- **Drift:** selvhostet i Docker (Coolify/Hetzner).

## Status

Iteration 1: datamodel. Se `db/migrations/0001_init.up.sql` — schemaet
afventer godkendelse før videre udvikling.

## Database

Migrationer ligger i `db/migrations/` som par af `NNNN_navn.up.sql` /
`NNNN_navn.down.sql`. Kræver PostgreSQL 16+ med `pgvector`-udvidelsen
(brug fx `pgvector/pgvector:pg16`-imaget i Docker).
