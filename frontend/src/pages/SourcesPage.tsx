import { useState } from 'react'
import { useOutletContext, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { LocalPlan, SourceDocument, SourceHit, SourceKind } from '../api/types'
import { confirmDelete, Empty, ErrorText, Modal, useLoad } from '../components'
import type { ProjectContext } from './ProjectShell'

const kindLabels: Record<SourceKind, string> = {
  br18: 'BR18',
  eurocode: 'Eurocode',
  local_plan: 'Lokalplan',
  municipal_guidance: 'Kommunal vejledning',
  other: 'Andet',
}

export default function SourcesPage() {
  const { projectId } = useParams()
  const { project } = useOutletContext<ProjectContext>()
  const { data: sources, error, reload } = useLoad(
    () => api.get<SourceDocument[]>(`/projects/${projectId}/sources`), [projectId])
  const [adding, setAdding] = useState(false)
  const [prefill, setPrefill] = useState<{ title: string; url: string } | null>(null)
  const [query, setQuery] = useState('')
  const [hits, setHits] = useState<SourceHit[] | null>(null)
  const [plans, setPlans] = useState<LocalPlan[] | null>(null)
  const [planError, setPlanError] = useState<string | null>(null)
  const [planBusy, setPlanBusy] = useState(false)
  const [importing, setImporting] = useState<string | null>(null)
  const [importNotice, setImportNotice] = useState<string | null>(null)

  async function importPlan(p: LocalPlan) {
    setImporting(p.planId)
    setPlanError(null)
    setImportNotice(null)
    try {
      const result = await api.post<{ notice: string }>(`/projects/${projectId}/localplans/import`, {
        name: p.name || `Lokalplan ${p.planId}`,
        docLink: p.docLink,
      })
      setImportNotice(result.notice)
      reload()
    } catch (err) {
      setPlanError((err as Error).message)
    } finally {
      setImporting(null)
    }
  }

  async function findLocalPlans() {
    if (project.utmX == null || project.utmY == null) {
      setPlanError('Projektet har ingen koordinater — opret projektet med adressesøgning, eller redigér adressen.')
      return
    }
    setPlanBusy(true)
    setPlanError(null)
    try {
      setPlans(await api.get<LocalPlan[]>(`/lookup/localplans?utmX=${project.utmX}&utmY=${project.utmY}`))
    } catch (err) {
      setPlanError((err as Error).message)
    } finally {
      setPlanBusy(false)
    }
  }

  async function search(e: React.FormEvent) {
    e.preventDefault()
    if (!query.trim()) return
    setHits(await api.get<SourceHit[]>(`/projects/${projectId}/sources/search?q=${encodeURIComponent(query)}`))
  }

  async function remove(s: SourceDocument) {
    if (!confirmDelete(`kilden "${s.title}"`)) return
    await api.del(`/sources/${s.id}`)
    reload()
  }

  return (
    <>
      <h1>Kildemateriale</h1>
      <p className="page-sub">
        BR18-tekster, lokalplan og kommunens krav. AI-assistenten svarer KUN ud fra det materiale du indlæser her.
      </p>
      <ErrorText error={error} />

      <div className="card">
        <table className="tbl">
          <thead>
            <tr><th>Titel</th><th>Type</th><th>Version</th><th className="num">Chunks</th><th></th></tr>
          </thead>
          <tbody>
            {sources?.map((s) => (
              <tr key={s.id}>
                <td>
                  <strong>{s.title}</strong>
                  {s.url && <div><a href={s.url} target="_blank" rel="noreferrer" style={{ fontSize: 12 }}>{s.url}</a></div>}
                </td>
                <td>{kindLabels[s.kind]}</td>
                <td>{s.versionLabel || '–'}</td>
                <td className="num">{s.chunkCount}</td>
                <td className="row-actions">
                  {s.projectId && <button className="btn small secondary" onClick={() => remove(s)}>Slet</button>}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {sources?.length === 0 && <Empty>Intet kildemateriale endnu — indsæt fx de relevante BR18-kapitler.</Empty>}
      </div>
      <div className="row-actions">
        <button className="btn" onClick={() => setAdding(true)}>+ Tilføj kilde</button>
        <button className="btn secondary" disabled={planBusy} onClick={findLocalPlans}>
          {planBusy ? 'Slår op i Plandata…' : '🔍 Find lokalplan for adressen'}
        </button>
      </div>
      <ErrorText error={planError} />
      {plans !== null && (
        <div className="card">
          <h3>Lokalplaner der dækker adressen</h3>
          {plans.length === 0 && <Empty>Ingen vedtagne lokalplaner fundet for punktet — dobbelttjek evt. på plandata.dk.</Empty>}
          {plans.map((p) => (
            <div key={p.planId} className="form-row" style={{ alignItems: 'center' }}>
              <strong style={{ flex: 1 }}>{p.name || `Plan ${p.planId}`}</strong>
              {p.docLink && <a href={p.docLink} target="_blank" rel="noreferrer">Åbn PDF</a>}
              {p.docLink && (
                <button className="btn small" disabled={importing !== null} onClick={() => importPlan(p)}>
                  {importing === p.planId ? 'Henter og indlæser…' : '⬇ Hent + indlæs automatisk'}
                </button>
              )}
              <button className="btn small secondary"
                onClick={() => (setPrefill({ title: p.name || `Lokalplan ${p.planId}`, url: p.docLink }), setAdding(true))}>
                Indsæt tekst manuelt
              </button>
            </div>
          ))}
          {importNotice && <div className="notice" style={{ background: 'var(--green-soft)', color: 'var(--green)', borderColor: '#bfe0c6' }}>{importNotice}</div>}
          <p className="hint">
            "Hent + indlæs automatisk" downloader PDF'en til arkivet og udtrækker teksten som kilde.
            Skannede planer uden tekstlag kræver stadig manuel indsættelse.
          </p>
        </div>
      )}

      <h2>Søg i kilderne</h2>
      <form onSubmit={search} className="form-row">
        <div className="field" style={{ flex: 1 }}>
          <input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="fx afstand til skel sommerhus" />
        </div>
        <button className="btn secondary">Søg</button>
      </form>
      {hits?.map((h) => (
        <div key={h.chunkId} className="card">
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <strong>{h.sectionRef || 'uden afsnitsreference'}</strong>
            <span className="badge blue">{h.sourceTitle}</span>
          </div>
          <p style={{ whiteSpace: 'pre-wrap', marginBottom: 0 }}>{h.content}</p>
        </div>
      ))}
      {hits !== null && hits.length === 0 && <Empty>Ingen resultater.</Empty>}

      {adding && (
        <IngestModal projectId={projectId!} prefill={prefill}
          onDone={() => (setAdding(false), setPrefill(null), reload())}
          onClose={() => (setAdding(false), setPrefill(null))} />
      )}
    </>
  )
}

function IngestModal({ projectId, prefill, onDone, onClose }: {
  projectId: string
  prefill?: { title: string; url: string } | null
  onDone: () => void
  onClose: () => void
}) {
  const [title, setTitle] = useState(prefill?.title ?? '')
  const [kind, setKind] = useState<SourceKind>(prefill ? 'local_plan' : 'br18')
  const [versionLabel, setVersionLabel] = useState('')
  const [url, setUrl] = useState(prefill?.url ?? '')
  const [content, setContent] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    try {
      await api.post(`/projects/${projectId}/sources`, { title, kind, versionLabel, url, content })
      onDone()
    } catch (err) {
      setError((err as Error).message)
      setBusy(false)
    }
  }

  return (
    <Modal title="Tilføj kildemateriale" wide onClose={onClose}>
      <form onSubmit={submit}>
        <div className="form-row">
          <div className="field" style={{ flex: 2 }}>
            <label>Titel *</label>
            <input value={title} onChange={(e) => setTitle(e.target.value)} required autoFocus placeholder="BR18 kapitel 8 — Byggeret" />
          </div>
          <div className="field">
            <label>Type</label>
            <select value={kind} onChange={(e) => setKind(e.target.value as SourceKind)}>
              {Object.entries(kindLabels).map(([k, label]) => (
                <option key={k} value={k}>{label}</option>
              ))}
            </select>
          </div>
        </div>
        <div className="form-row">
          <div className="field">
            <label>Versionsmærke</label>
            <input value={versionLabel} onChange={(e) => setVersionLabel(e.target.value)} placeholder="BR18 pr. 01-01-2026" />
          </div>
          <div className="field" style={{ flex: 1 }}>
            <label>Kilde-URL</label>
            <input value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://bygningsreglementet.dk/…" />
          </div>
        </div>
        <div className="field">
          <label>Tekstindhold * (indsæt selve regelteksten — paragraffer genkendes automatisk)</label>
          <textarea rows={12} value={content} onChange={(e) => setContent(e.target.value)} required
            placeholder={'§ 180\nSommerhuse skal holdes mindst 5,0 m fra skel…'} />
        </div>
        <ErrorText error={error} />
        <div className="form-row" style={{ justifyContent: 'flex-end', marginTop: 12 }}>
          <button type="button" className="btn secondary" onClick={onClose}>Annullér</button>
          <button className="btn" disabled={busy}>{busy ? 'Indlæser…' : 'Indlæs kilde'}</button>
        </div>
      </form>
    </Modal>
  )
}
