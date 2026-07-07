import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { CaseFile } from '../api/types'
import { Empty, ErrorText, Modal, StatusBadge, statusLabel, useLoad } from '../components'

export default function CasesPage() {
  const { projectId } = useParams()
  const { data: cases, error, reload } = useLoad(
    () => api.get<CaseFile[]>(`/projects/${projectId}/case-files`), [projectId])
  const [creating, setCreating] = useState(false)

  return (
    <>
      <h1>Byggesager</h1>
      <p className="page-sub">Kommunale ansøgninger — fra beskrivelse til afgørelse.</p>
      <ErrorText error={error} />

      {cases?.map((c) => (
        <Link key={c.id} to={c.id} style={{ textDecoration: 'none', color: 'inherit' }}>
          <div className="card">
            <div style={{ display: 'flex', justifyContent: 'space-between', gap: 8 }}>
              <strong style={{ fontSize: 15 }}>{c.title}</strong>
              <StatusBadge status={c.status} />
            </div>
            <div style={{ fontSize: 13, color: 'var(--text-dim)', marginTop: 4 }}>
              {statusLabel(c.caseType)}
              {c.municipalCaseNumber && <> · sagsnr. {c.municipalCaseNumber}</>}
            </div>
          </div>
        </Link>
      ))}
      {cases?.length === 0 && <Empty>Ingen byggesager endnu.</Empty>}

      <button className="btn" onClick={() => setCreating(true)}>+ Ny byggesag</button>

      {creating && (
        <Modal title="Ny byggesag" onClose={() => setCreating(false)}>
          <NewCaseForm projectId={projectId!} onDone={() => (setCreating(false), reload())} />
        </Modal>
      )}
    </>
  )
}

function NewCaseForm({ projectId, onDone }: { projectId: string; onDone: () => void }) {
  const [title, setTitle] = useState('')
  const [caseType, setCaseType] = useState('unknown')
  const [description, setDescription] = useState('')
  const [error, setError] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    try {
      await api.post(`/projects/${projectId}/case-files`, { title, caseType, description })
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  return (
    <form onSubmit={submit}>
      <div className="form-row">
        <div className="field" style={{ flex: 2 }}>
          <label>Titel *</label>
          <input value={title} onChange={(e) => setTitle(e.target.value)} required autoFocus />
        </div>
        <div className="field">
          <label>Sagstype</label>
          <select value={caseType} onChange={(e) => setCaseType(e.target.value)}>
            <option value="unknown">Ikke afklaret endnu</option>
            <option value="notification">Anmeldelse</option>
            <option value="building_permit">Byggetilladelse</option>
          </select>
        </div>
      </div>
      <div className="field">
        <label>Beskriv det ønskede byggeri (fritekst)</label>
        <textarea
          rows={6}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Fx: Opførelse af nyt sommerhus på 68 m² med saddeltag…"
        />
      </div>
      <ErrorText error={error} />
      <div className="form-row" style={{ justifyContent: 'flex-end', marginTop: 12 }}>
        <button className="btn">Opret sag</button>
      </div>
    </form>
  )
}
