import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { CaseFile, Drawing, DrawingKind } from '../api/types'
import { Empty, ErrorText, Modal, useLoad } from '../components'

const kindLabels: Record<DrawingKind, string> = {
  site_plan: 'Situationsplan',
  floor_plan: 'Grundplan',
  elevation: 'Opstalt',
  section: 'Snit',
  detail: 'Detalje',
  other: 'Andet',
}

export default function DrawingsPage() {
  const { projectId } = useParams()
  const { data: drawings, error, reload } = useLoad(
    () => api.get<Drawing[]>(`/projects/${projectId}/drawings`), [projectId])
  const { data: cases } = useLoad(() => api.get<CaseFile[]>(`/projects/${projectId}/case-files`), [projectId])
  const [creating, setCreating] = useState(false)

  return (
    <>
      <h1>Tegninger</h1>
      <p className="page-sub">Målfaste 2D-grundplaner med vægge, rum, døre/vinduer og placering på grunden. Versioneret.</p>
      <ErrorText error={error} />

      <div className="grid cols-3">
        {drawings?.map((d) => (
          <Link key={d.id} to={d.id} style={{ textDecoration: 'none', color: 'inherit' }}>
            <div className="card" style={{ marginBottom: 0 }}>
              <strong>{d.title}</strong>
              <div style={{ fontSize: 13, color: 'var(--text-dim)', marginTop: 4 }}>
                {kindLabels[d.kind]}
                {d.caseFileId && cases && <> · {cases.find((c) => c.id === d.caseFileId)?.title}</>}
              </div>
            </div>
          </Link>
        ))}
      </div>
      {drawings?.length === 0 && <Empty>Ingen tegninger endnu.</Empty>}

      <div style={{ marginTop: 16 }}>
        <button className="btn" onClick={() => setCreating(true)}>+ Ny tegning</button>
      </div>

      {creating && (
        <NewDrawingModal projectId={projectId!} cases={cases ?? []} onDone={() => (setCreating(false), reload())} onClose={() => setCreating(false)} />
      )}
    </>
  )
}

function NewDrawingModal({ projectId, cases, onDone, onClose }: {
  projectId: string
  cases: CaseFile[]
  onDone: () => void
  onClose: () => void
}) {
  const [title, setTitle] = useState('')
  const [kind, setKind] = useState<DrawingKind>('floor_plan')
  const [caseFileId, setCaseFileId] = useState('')
  const [error, setError] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    try {
      await api.post(`/projects/${projectId}/drawings`, {
        title, kind,
        caseFileId: caseFileId || undefined,
      })
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  return (
    <Modal title="Ny tegning" onClose={onClose}>
      <form onSubmit={submit}>
        <div className="form-row">
          <div className="field" style={{ flex: 2 }}>
            <label>Titel *</label>
            <input value={title} onChange={(e) => setTitle(e.target.value)} required autoFocus />
          </div>
          <div className="field">
            <label>Type</label>
            <select value={kind} onChange={(e) => setKind(e.target.value as DrawingKind)}>
              {Object.entries(kindLabels).map(([k, label]) => (
                <option key={k} value={k}>{label}</option>
              ))}
            </select>
          </div>
        </div>
        <div className="field">
          <label>Knyt til byggesag</label>
          <select value={caseFileId} onChange={(e) => setCaseFileId(e.target.value)}>
            <option value="">(ingen)</option>
            {cases.map((c) => <option key={c.id} value={c.id}>{c.title}</option>)}
          </select>
        </div>
        <ErrorText error={error} />
        <div className="form-row" style={{ justifyContent: 'flex-end', marginTop: 12 }}>
          <button type="button" className="btn secondary" onClick={onClose}>Annullér</button>
          <button className="btn">Opret</button>
        </div>
      </form>
    </Modal>
  )
}
