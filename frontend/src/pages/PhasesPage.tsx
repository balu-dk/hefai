import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Phase } from '../api/types'
import { confirmDelete, ErrorText, Modal, StatusBadge, useLoad } from '../components'

export default function PhasesPage() {
  const { projectId } = useParams()
  const { data: phases, error, reload } = useLoad(
    () => api.get<Phase[]>(`/projects/${projectId}/phases`),
    [projectId],
  )
  const [editing, setEditing] = useState<Phase | null>(null)
  const [creating, setCreating] = useState(false)

  return (
    <>
      <h1>Faser</h1>
      <p className="page-sub">Byggeriets faser med planlagte og faktiske datoer.</p>
      <ErrorText error={error} />

      <div className="card">
        <table className="tbl">
          <thead>
            <tr>
              <th>Fase</th>
              <th>Status</th>
              <th>Planlagt start</th>
              <th>Planlagt slut</th>
              <th>Faktisk start</th>
              <th>Faktisk slut</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {phases?.map((p) => (
              <tr key={p.id}>
                <td>
                  <strong>{p.name}</strong>
                  {p.description && <div style={{ fontSize: 12.5, color: 'var(--text-dim)' }}>{p.description}</div>}
                </td>
                <td><StatusBadge status={p.status} /></td>
                <td>{p.plannedStart?.slice(0, 10) ?? '–'}</td>
                <td>{p.plannedEnd?.slice(0, 10) ?? '–'}</td>
                <td>{p.actualStart?.slice(0, 10) ?? '–'}</td>
                <td>{p.actualEnd?.slice(0, 10) ?? '–'}</td>
                <td className="row-actions">
                  <button className="btn small secondary" onClick={() => setEditing(p)}>Redigér</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <button className="btn" onClick={() => setCreating(true)}>+ Ny fase</button>

      {(editing || creating) && (
        <PhaseModal
          phase={editing}
          projectId={projectId!}
          onDone={() => (setEditing(null), setCreating(false), reload())}
          onClose={() => (setEditing(null), setCreating(false))}
        />
      )}
    </>
  )
}

function PhaseModal({ phase, projectId, onDone, onClose }: {
  phase: Phase | null
  projectId: string
  onDone: () => void
  onClose: () => void
}) {
  const [name, setName] = useState(phase?.name ?? '')
  const [status, setStatus] = useState(phase?.status ?? 'not_started')
  const [plannedStart, setPlannedStart] = useState(phase?.plannedStart?.slice(0, 10) ?? '')
  const [plannedEnd, setPlannedEnd] = useState(phase?.plannedEnd?.slice(0, 10) ?? '')
  const [actualStart, setActualStart] = useState(phase?.actualStart?.slice(0, 10) ?? '')
  const [actualEnd, setActualEnd] = useState(phase?.actualEnd?.slice(0, 10) ?? '')
  const [error, setError] = useState<string | null>(null)

  const asDate = (v: string) => (v ? `${v}T00:00:00Z` : undefined)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    const body = {
      name,
      status,
      plannedStart: asDate(plannedStart),
      plannedEnd: asDate(plannedEnd),
      actualStart: asDate(actualStart),
      actualEnd: asDate(actualEnd),
    }
    try {
      if (phase) await api.patch(`/phases/${phase.id}`, body)
      else await api.post(`/projects/${projectId}/phases`, body)
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function remove() {
    if (!phase || !confirmDelete(`fasen "${phase.name}"`)) return
    try {
      await api.del(`/phases/${phase.id}`)
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  return (
    <Modal title={phase ? 'Redigér fase' : 'Ny fase'} onClose={onClose}>
      <form onSubmit={submit}>
        <div className="form-row">
          <div className="field" style={{ flex: 2 }}>
            <label>Navn *</label>
            <input value={name} onChange={(e) => setName(e.target.value)} required />
          </div>
          <div className="field">
            <label>Status</label>
            <select value={status} onChange={(e) => setStatus(e.target.value as Phase['status'])}>
              <option value="not_started">Ikke startet</option>
              <option value="in_progress">I gang</option>
              <option value="completed">Afsluttet</option>
            </select>
          </div>
        </div>
        <div className="form-row">
          <div className="field">
            <label>Planlagt start</label>
            <input type="date" value={plannedStart} onChange={(e) => setPlannedStart(e.target.value)} />
          </div>
          <div className="field">
            <label>Planlagt slut</label>
            <input type="date" value={plannedEnd} onChange={(e) => setPlannedEnd(e.target.value)} />
          </div>
        </div>
        <div className="form-row">
          <div className="field">
            <label>Faktisk start</label>
            <input type="date" value={actualStart} onChange={(e) => setActualStart(e.target.value)} />
          </div>
          <div className="field">
            <label>Faktisk slut</label>
            <input type="date" value={actualEnd} onChange={(e) => setActualEnd(e.target.value)} />
          </div>
        </div>
        <ErrorText error={error} />
        <div className="form-row" style={{ justifyContent: 'space-between', marginTop: 12 }}>
          <div>{phase && <button type="button" className="btn danger" onClick={remove}>Slet</button>}</div>
          <div className="row-actions">
            <button type="button" className="btn secondary" onClick={onClose}>Annullér</button>
            <button className="btn">Gem</button>
          </div>
        </div>
      </form>
    </Modal>
  )
}
