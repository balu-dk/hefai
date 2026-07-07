# Hefai

Byggeprojekt-platform der dækker hele processen med at bygge eller renovere et
hus — fra planlægning over kommunal byggesag til statiker-forberedelse. Navnet
kommer fra Hefaistos, smedeguden og byggeren.

**Hefai forbereder, strukturerer og sparer tid — men træffer aldrig
myndighedsafgørelser og erstatter ikke en autoriseret statiker eller
rådgiver.** Hvor faglig godkendelse kræves, producerer Hefai et gennemarbejdet
udgangspunkt, tydeligt markeret som kladde der kræver godkendelse.

## Moduler

**Projekt & proces** — faser med tidslinje, opgaver med afhængigheder
("hvad kan jeg gøre nu / hvad blokerer hvad"), budget med øre-præcis
brugt/tilbage pr. fase og kategori, materialeliste med indkøbsliste grupperet
pr. leverandør, kontaktregister, dokumentarkiv med tags/søgning/visning af
PDF og billeder i browseren, og rum/zoner der samler alt om ét rum.

**Byggesag** — sagsobjekt med status-flow og automatisk logget tidslinje,
korrespondance-log, målfast 2D-tegneflade (SVG: vægge, rum, døre/vinduer,
grund/skel, versioneret), deterministisk PDF-generering (plantegning,
situationsplan, arealopgørelse, beskrivelse, ansøgningsoversigt — alle med
synligt kladde-banner), ikke-bindende compliance-tjekliste med
kildehenvisninger, og en AI-assistent der KUN svarer ud fra indlæst
kildemateriale (BR18, lokalplan, kommunens krav) og altid citerer kilder.

**Statiker-forberedelse** — konstruktionselementer, laster med udledning og
standardreference, vejledende beregninger i deterministisk Go-kode (snelast
og vindlast efter Eurocode 1 + DK NA, træbjælke-overslag efter Eurocode 5)
med alle antagelser eksplicit og testet mod håndregnede referenceværdier,
versioneret statiker-pakke som PDF med svarfelter, og et feedback-loop hvor
statikerens punkt-for-punkt-verdicts spores tilbage på laster og beregninger.

## Stack & arkitektur

- **Backend:** Go 1.24 + PostgreSQL 16 (pgvector). Lagdelt:
  `domain` (rene modeller) → `repository` (pgx/SQL) → `service`
  (forretningslogik + adgangskontrol) → `httpapi` (JSON-handlers).
  Stdlib-router, JWT-auth, bcrypt.
- **Beregninger:** `internal/calc` — rene funktioner med eksplicitte formler
  og standardreferencer. Aldrig LLM-genererede tal; uden for gyldighedsområdet
  afvises med henvisning til statiker.
- **RAG:** `internal/rag` chunker kildetekst med §-genkendelse; dansk
  fuldtekstsøgning i Postgres. `internal/ai` er et provider-agnostisk
  LLM-interface — provider vælges senere; indtil da returnerer assistenten
  rene kildeuddrag med citater.
- **Frontend:** React 18 + TypeScript + Vite, react-router. Ingen
  UI-framework-afhængighed.
- **Multi-projekt og multi-bruger** (ejer/medlem/læseadgang pr. projekt).

## Udvikling

```bash
# Database (kræver PostgreSQL 16 med pgvector)
createdb hefai && psql -d hefai -c 'CREATE EXTENSION vector'

# Backend — migrationer køres automatisk ved opstart
cd backend
DATABASE_URL=postgres://hefai:hefai@localhost:5432/hefai \
JWT_SECRET=dev-secret \
MIGRATIONS_DIR=../db/migrations \
go run ./cmd/hefai

# Frontend (proxier /api til :8080)
cd frontend && npm install && npm run dev
```

Tests: `cd backend && go test ./...` — inkl. beregningstests mod håndregnede
værdier, cykel-detektion, chunker og board-logik.

## Deploy

Selvhostet via Docker Compose på enhver Docker-vært — ingen tredjeparts-SaaS
til kernefunktioner:

```bash
export JWT_SECRET=$(openssl rand -hex 32)
export POSTGRES_PASSWORD=$(openssl rand -hex 16)
docker compose up -d --build
# → http://<host>:8090
```

Platforme med Docker Compose-understøttelse kan pege direkte på dette repo;
sæt `JWT_SECRET`/`POSTGRES_PASSWORD` som miljøvariabler. Uploads ligger i
`files`-volumen, databasen i `pgdata`.

### Valgfrie integrationer (miljøvariabler)

| Variabel | Effekt |
| --- | --- |
| `LLM_BASE_URL` | OpenAI-kompatibelt endpoint (fx `https://api.deepseek.com/v1` eller en gateway). Aktiverer AI-assistentens formulerede svar og LLM-genererede projektplaner. |
| `LLM_API_KEY`, `LLM_MODEL` | Nøgle og modelnavn (standard `deepseek-chat`). |
| `AI_DOCS_DIR` | Mappe med redigerbare MD-instruktioner til AI-funktionerne (standard `ai-docs/` — se `blueprint.md`). |
| `ORTHO_TOKEN` | Gratis token fra dataforsyningen.dk. Aktiverer danske luftfotos under 2D-tegningen og i 3D-modellen. |
| `ORTHO_WMS_URL`, `ORTHO_WMS_LAYER` | Alternativ WMS-fototjeneste (standard Dataforsyningens ortofoto). |

Uden disse degraderer alt pænt: assistenten viser rene kildeuddrag,
AI-projektstarten bruger den deterministiske skabelon, og luftfoto-felterne
forklarer hvordan de slås til.

## Database

Migrationer ligger i `db/migrations` som `NNNN_navn.up.sql`/`.down.sql` og
køres automatisk af backenden ved opstart (tracket i `schema_migrations`).
Schemaet dækker alle tre moduler — se `0001_init.up.sql` for det fulde
skema med kommentarer.

## AI-assistentens grundregler

Indbygget i system-prompt og produktdesign:

1. Henviser kun til paragraffer og krav der står ordret i det indlæste
   kildemateriale — opfinder aldrig referencer.
2. Findes svaret ikke i kilderne, siges det direkte med henvisning til
   kommune/rådgiver.
3. Garanterer aldrig godkendelse; alt der kræver bekræftelse markeres
   "KRÆVER BEKRÆFTELSE".
4. Regner aldrig selv — henviser til beregningsmodulet og statikeren.
