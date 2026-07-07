import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { api, fetchDocumentObjectURL, formatDate } from '../api/client'
import type {
  CaseEvent, CaseFile, CaseStatus, ComplianceItem, ComplianceStatus, GeneratedDocument, GeneratedKind,
} from '../api/types'
import { confirmDelete, Empty, ErrorText, Modal, StatusBadge, statusLabel, useLoad } from '../components'

const caseStatuses: CaseStatus[] = [
  'draft', 'ready_for_submission', 'submitted', 'awaiting_response',
  'questions_from_municipality', 'approved', 'rejected', 'closed',
]

const generatedKinds: { kind: GeneratedKind; label: string }[] = [
  { kind: 'floor_plan', label: 'Plantegning' },
  { kind: 'site_plan', label: 'Situationsplan' },
  { kind: 'area_statement', label: 'Arealopgørelse' },
  { kind: 'project_description', label: 'Projektbeskrivelse' },
  { kind: 'application_summary', label: 'Ansøgningsoversigt' },
]

export default function CaseDetailPage() {
  const { projectId, caseFileId } = useParams()
  const navigate = useNavigate()
  const { data: caseFile, error, reload } = useLoad(
    () => api.get<CaseFile>(`/case-files/${caseFileId}`), [caseFileId])
  const { data: events, reload: reloadEvents } = useLoad(
    () => api.get<CaseEvent[]>(`/case-files/${caseFileId}/events`), [caseFileId])
  const { data: checklist, reload: reloadChecklist } = useLoad(
    () => api.get<ComplianceItem[]>(`/case-files/${caseFileId}/checklist`), [caseFileId])
  const { data: generated, reload: reloadGenerated } = useLoad(
    () => api.get<GeneratedDocument[]>(`/case-files/${caseFileId}/generated`), [caseFileId])

  const [genError, setGenError] = useState<string | null>(null)
  const [genBusy, setGenBusy] = useState<string | null>(null)
  const [addingEvent, setAddingEvent] = useState(false)
  const [addingCheck, setAddingCheck] = useState(false)
  const [editingDescription, setEditingDescription] = useState(false)

  if (error) return <ErrorText error={error} />
  if (!caseFile) return <p>Indlæser…</p>

  async function setStatus(status: CaseStatus) {
    await api.patch(`/case-files/${caseFileId}`, { status })
    reload()
    reloadEvents()
  }

  async function generate(kind: GeneratedKind) {
    setGenBusy(kind)
    setGenError(null)
    try {
      await api.post(`/case-files/${caseFileId}/generate`, { kind })
      reloadGenerated()
    } catch (err) {
      setGenError((err as Error).message)
    } finally {
      setGenBusy(null)
    }
  }

  async function openGenerated(g: GeneratedDocument) {
    if (!g.documentId) return
    const url = await fetchDocumentObjectURL(g.documentId)
    window.open(url, '_blank')
  }

  async function removeCase() {
    if (!confirmDelete(`byggesagen "${caseFile!.title}"`)) return
    await api.del(`/case-files/${caseFileId}`)
    navigate(`/projects/${projectId}/cases`)
  }

  return (
    <>
      <h1>{caseFile.title}</h1>
      <p className="page-sub">
        {statusLabel(caseFile.caseType)}
        {caseFile.municipalCaseNumber && <> · sagsnr. {caseFile.municipalCaseNumber}</>}
        {caseFile.submittedAt && <> · indsendt {formatDate(caseFile.submittedAt)}</>}
      </p>

      <div className="card">
        <div className="form-row" style={{ alignItems: 'center' }}>
          <strong>Status:</strong>
          <StatusBadge status={caseFile.status} />
          <select value={caseFile.status} onChange={(e) => setStatus(e.target.value as CaseStatus)}>
            {caseStatuses.map((s) => (
              <option key={s} value={s}>{statusLabel(s)}</option>
            ))}
          </select>
        </div>
      </div>

      <div className="grid cols-2">
        <div className="card">
          <h3>Beskrivelse af byggeriet</h3>
          {caseFile.description ? <p style={{ whiteSpace: 'pre-wrap' }}>{caseFile.description}</p> : <Empty>Ingen beskrivelse endnu.</Empty>}
          <button className="btn small secondary" onClick={() => setEditingDescription(true)}>Redigér</button>
          <p className="hint" style={{ marginTop: 10 }}>
            Tegn grundplanen i <Link to={`/projects/${projectId}/drawings`}>tegnefladen</Link> og spørg{' '}
            <Link to={`/projects/${projectId}/assistant`}>assistenten</Link> om sagstype og krav.
          </p>
        </div>

        <div className="card">
          <h3>Generér ansøgningsdokumenter</h3>
          <p className="hint">PDF-kladder klar til at vedhæfte — tydeligt markeret som kladde.</p>
          <div className="row-actions" style={{ flexWrap: 'wrap' }}>
            {generatedKinds.map(({ kind, label }) => (
              <button key={kind} className="btn small secondary" disabled={genBusy !== null} onClick={() => generate(kind)}>
                {genBusy === kind ? 'Genererer…' : label}
              </button>
            ))}
          </div>
          <ErrorText error={genError} />
          {generated && generated.length > 0 && (
            <table className="tbl" style={{ marginTop: 10 }}>
              <tbody>
                {generated.map((g) => (
                  <tr key={g.id}>
                    <td>{generatedKinds.find((k) => k.kind === g.kind)?.label ?? g.kind} v{g.versionNo}</td>
                    <td><StatusBadge status={g.status} /></td>
                    <td>{formatDate(g.createdAt)}</td>
                    <td>
                      <button className="btn small secondary" onClick={() => openGenerated(g)}>Åbn PDF</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      <h2>Compliance-tjekliste (ikke-bindende egenkontrol)</h2>
      <div className="notice">
        Tjeklisten er en hjælp til egenkontrol — den er ikke en myndighedsvurdering og garanterer ikke godkendelse.
      </div>
      <div className="card">
        <table className="tbl">
          <thead>
            <tr><th>Kategori</th><th>Krav</th><th>Krav-værdi</th><th>Projekt-værdi</th><th>Kilde</th><th>Status</th><th></th></tr>
          </thead>
          <tbody>
            {checklist?.map((item) => (
              <ChecklistRow key={item.id} item={item} onChanged={reloadChecklist} />
            ))}
          </tbody>
        </table>
        {checklist?.length === 0 && <Empty>Ingen tjekpunkter endnu.</Empty>}
      </div>
      <button className="btn secondary" onClick={() => setAddingCheck(true)}>+ Tjekpunkt</button>

      <h2>Tidslinje & korrespondance</h2>
      <div className="card">
        {events?.map((e) => (
          <div key={e.id} style={{ padding: '8px 0', borderBottom: '1px solid #efece6' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <strong style={{ fontSize: 13.5 }}>{e.summary}</strong>
              <span style={{ fontSize: 12.5, color: 'var(--text-dim)' }}>
                {formatDate(e.occurredAt)}
                {e.direction && <> · {e.direction === 'incoming' ? 'indgående' : e.direction === 'outgoing' ? 'udgående' : 'internt'}</>}
              </span>
            </div>
            {e.body && <p style={{ margin: '4px 0 0', fontSize: 13, whiteSpace: 'pre-wrap' }}>{e.body}</p>}
          </div>
        ))}
        {events?.length === 0 && <Empty>Ingen hændelser endnu.</Empty>}
      </div>
      <div className="row-actions">
        <button className="btn secondary" onClick={() => setAddingEvent(true)}>+ Korrespondance/notat</button>
        <button className="btn danger" onClick={removeCase}>Slet sag</button>
      </div>

      {editingDescription && (
        <EditDescriptionModal caseFile={caseFile} onDone={() => (setEditingDescription(false), reload())} onClose={() => setEditingDescription(false)} />
      )}
      {addingEvent && (
        <EventModal caseFileId={caseFileId!} onDone={() => (setAddingEvent(false), reloadEvents())} onClose={() => setAddingEvent(false)} />
      )}
      {addingCheck && (
        <ChecklistModal caseFileId={caseFileId!} projectId={projectId!} onDone={() => (setAddingCheck(false), reloadChecklist())} onClose={() => setAddingCheck(false)} />
      )}
    </>
  )
}

function ChecklistRow({ item, onChanged }: { item: ComplianceItem; onChanged: () => void }) {
  const order: ComplianceStatus[] = ['not_checked', 'ok', 'attention', 'needs_confirmation', 'confirmed']
  async function cycle() {
    const next = order[(order.indexOf(item.status) + 1) % order.length]
    await api.patch(`/checklist-items/${item.id}`, { status: next })
    onChanged()
  }
  async function remove() {
    if (!confirmDelete('tjekpunktet')) return
    await api.del(`/checklist-items/${item.id}`)
    onChanged()
  }
  return (
    <tr>
      <td>{item.category || '–'}</td>
      <td>{item.requirement}{item.note && <div className="hint">{item.note}</div>}</td>
      <td>{item.expectedValue || '–'}</td>
      <td>{item.actualValue || '–'}</td>
      <td>{item.sourceRef ? <span className="badge blue">{item.sourceRef}</span> : <span className="badge red">uden kilde</span>}</td>
      <td className="checklist-status" onClick={cycle} title="Klik for at skifte status">
        <StatusBadge status={item.status} />
      </td>
      <td><button className="btn small secondary" onClick={remove}>Slet</button></td>
    </tr>
  )
}

function EditDescriptionModal({ caseFile, onDone, onClose }: {
  caseFile: CaseFile
  onDone: () => void
  onClose: () => void
}) {
  const [description, setDescription] = useState(caseFile.description)
  const [caseType, setCaseType] = useState(caseFile.caseType)
  const [caseNumber, setCaseNumber] = useState(caseFile.municipalCaseNumber)
  const [error, setError] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    try {
      await api.patch(`/case-files/${caseFile.id}`, { description, caseType, municipalCaseNumber: caseNumber })
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  return (
    <Modal title="Redigér sag" onClose={onClose}>
      <form onSubmit={submit}>
        <div className="form-row">
          <div className="field">
            <label>Sagstype</label>
            <select value={caseType} onChange={(e) => setCaseType(e.target.value as CaseFile['caseType'])}>
              <option value="unknown">Ikke afklaret</option>
              <option value="notification">Anmeldelse</option>
              <option value="building_permit">Byggetilladelse</option>
            </select>
          </div>
          <div className="field">
            <label>Kommunens sagsnr.</label>
            <input value={caseNumber} onChange={(e) => setCaseNumber(e.target.value)} />
          </div>
        </div>
        <div className="field">
          <label>Beskrivelse</label>
          <textarea rows={8} value={description} onChange={(e) => setDescription(e.target.value)} />
        </div>
        <ErrorText error={error} />
        <div className="form-row" style={{ justifyContent: 'flex-end', marginTop: 12 }}>
          <button type="button" className="btn secondary" onClick={onClose}>Annullér</button>
          <button className="btn">Gem</button>
        </div>
      </form>
    </Modal>
  )
}

function EventModal({ caseFileId, onDone, onClose }: {
  caseFileId: string
  onDone: () => void
  onClose: () => void
}) {
  const [eventType, setEventType] = useState('note')
  const [direction, setDirection] = useState('')
  const [summary, setSummary] = useState('')
  const [body, setBody] = useState('')
  const [error, setError] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    try {
      await api.post(`/case-files/${caseFileId}/events`, {
        eventType, summary, body,
        direction: direction || undefined,
      })
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  return (
    <Modal title="Ny hændelse" onClose={onClose}>
      <form onSubmit={submit}>
        <div className="form-row">
          <div className="field">
            <label>Type</label>
            <select value={eventType} onChange={(e) => setEventType(e.target.value)}>
              <option value="note">Notat</option>
              <option value="correspondence">Korrespondance</option>
              <option value="submission">Indsendelse</option>
            </select>
          </div>
          <div className="field">
            <label>Retning</label>
            <select value={direction} onChange={(e) => setDirection(e.target.value)}>
              <option value="">–</option>
              <option value="incoming">Indgående (fra kommunen)</option>
              <option value="outgoing">Udgående (til kommunen)</option>
              <option value="internal">Internt</option>
            </select>
          </div>
        </div>
        <div className="field" style={{ marginBottom: 10 }}>
          <label>Resumé *</label>
          <input value={summary} onChange={(e) => setSummary(e.target.value)} required autoFocus />
        </div>
        <div className="field">
          <label>Indhold</label>
          <textarea value={body} onChange={(e) => setBody(e.target.value)} />
        </div>
        <ErrorText error={error} />
        <div className="form-row" style={{ justifyContent: 'flex-end', marginTop: 12 }}>
          <button type="button" className="btn secondary" onClick={onClose}>Annullér</button>
          <button className="btn">Gem</button>
        </div>
      </form>
    </Modal>
  )
}

function ChecklistModal({ caseFileId, projectId, onDone, onClose }: {
  caseFileId: string
  projectId: string
  onDone: () => void
  onClose: () => void
}) {
  const [category, setCategory] = useState('')
  const [requirement, setRequirement] = useState('')
  const [expectedValue, setExpectedValue] = useState('')
  const [actualValue, setActualValue] = useState('')
  const [search, setSearch] = useState('')
  const [chunkId, setChunkId] = useState('')
  const [hits, setHits] = useState<{ chunkId: string; sectionRef: string; content: string }[]>([])
  const [error, setError] = useState<string | null>(null)

  async function findSource() {
    if (!search.trim()) return
    const results = await api.get<any[]>(`/projects/${projectId}/sources/search?q=${encodeURIComponent(search)}&limit=5`)
    setHits(results)
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    try {
      await api.post(`/case-files/${caseFileId}/checklist`, {
        category, requirement, expectedValue, actualValue,
        sourceChunkId: chunkId || undefined,
      })
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  return (
    <Modal title="Nyt tjekpunkt" wide onClose={onClose}>
      <form onSubmit={submit}>
        <div className="form-row">
          <div className="field">
            <label>Kategori</label>
            <input value={category} onChange={(e) => setCategory(e.target.value)} placeholder="skelafstand, højde…" />
          </div>
          <div className="field" style={{ flex: 2 }}>
            <label>Krav *</label>
            <input value={requirement} onChange={(e) => setRequirement(e.target.value)} required />
          </div>
        </div>
        <div className="form-row">
          <div className="field">
            <label>Krav-værdi</label>
            <input value={expectedValue} onChange={(e) => setExpectedValue(e.target.value)} placeholder="min. 5,0 m" />
          </div>
          <div className="field">
            <label>Projektets værdi</label>
            <input value={actualValue} onChange={(e) => setActualValue(e.target.value)} placeholder="6,2 m (fra tegning)" />
          </div>
        </div>

        <h3>Kildegrundlag (anbefalet)</h3>
        <p className="hint">Punkter uden kilde markeres som "kræver bekræftelse".</p>
        <div className="form-row">
          <div className="field" style={{ flex: 1 }}>
            <input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Søg i kildemateriale…" />
          </div>
          <button type="button" className="btn small secondary" onClick={findSource}>Søg</button>
        </div>
        {hits.map((h) => (
          <label key={h.chunkId} style={{ display: 'block', padding: 8, border: '1px solid var(--border)', borderRadius: 6, marginBottom: 6, cursor: 'pointer', background: chunkId === h.chunkId ? 'var(--accent-soft)' : undefined }}>
            <input type="radio" name="chunk" checked={chunkId === h.chunkId} onChange={() => setChunkId(h.chunkId)} />{' '}
            <strong>{h.sectionRef || 'uden ref'}</strong>
            <div style={{ fontSize: 12.5, color: 'var(--text-dim)' }}>{h.content.slice(0, 220)}…</div>
          </label>
        ))}

        <ErrorText error={error} />
        <div className="form-row" style={{ justifyContent: 'flex-end', marginTop: 12 }}>
          <button type="button" className="btn secondary" onClick={onClose}>Annullér</button>
          <button className="btn">Gem tjekpunkt</button>
        </div>
      </form>
    </Modal>
  )
}
