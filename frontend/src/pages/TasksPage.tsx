import { useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { BoardTask, Phase, Room, Supplier, TaskStatus } from '../api/types'
import { confirmDelete, Empty, ErrorText, Modal, StatusBadge, useLoad } from '../components'

export default function TasksPage() {
  const { projectId } = useParams()
  const { data: board, error, reload } = useLoad(
    () => api.get<BoardTask[]>(`/projects/${projectId}/tasks/board`),
    [projectId],
  )
  const { data: phases } = useLoad(() => api.get<Phase[]>(`/projects/${projectId}/phases`), [projectId])
  const { data: rooms } = useLoad(() => api.get<Room[]>(`/projects/${projectId}/rooms`), [projectId])
  const { data: suppliers } = useLoad(() => api.get<Supplier[]>(`/projects/${projectId}/suppliers`), [projectId])
  const [editing, setEditing] = useState<BoardTask | null>(null)
  const [creating, setCreating] = useState(false)

  const byId = useMemo(() => new Map((board ?? []).map((t) => [t.id, t])), [board])
  const phaseName = (id: string | null) => phases?.find((p) => p.id === id)?.name

  const columns: { key: string; title: string; filter: (t: BoardTask) => boolean; cls?: string }[] = [
    { key: 'now', title: 'Klar til start', filter: (t) => t.actionable, cls: 'actionable' },
    { key: 'waiting', title: 'Venter på andre', filter: (t) => (t.status === 'todo' || t.status === 'blocked') && !t.actionable, cls: 'waiting' },
    { key: 'doing', title: 'I gang', filter: (t) => t.status === 'in_progress' },
    { key: 'done', title: 'Færdige', filter: (t) => t.status === 'done' || t.status === 'cancelled' },
  ]

  return (
    <>
      <h1>Opgaver</h1>
      <p className="page-sub">Opgaver med afhængigheder — se hvad du kan gøre nu, og hvad der blokerer hvad.</p>
      <ErrorText error={error} />

      <div className="board">
        {columns.map((col) => {
          const tasks = (board ?? []).filter(col.filter)
          return (
            <div className="col" key={col.key}>
              <h3>
                {col.title} ({tasks.length})
              </h3>
              {tasks.map((t) => (
                <div
                  key={t.id}
                  className={`task-card ${col.cls ?? ''}`}
                  style={{ cursor: 'pointer' }}
                  onClick={() => setEditing(t)}
                >
                  <div className="title">{t.title}</div>
                  <div className="meta">
                    <StatusBadge status={t.status} />
                    {phaseName(t.phaseId ?? null) && <> · {phaseName(t.phaseId)}</>}
                  </div>
                  {t.waitingFor.length > 0 && (
                    <div className="meta" style={{ marginTop: 4 }}>
                      Venter på: {t.waitingFor.map((id) => byId.get(id)?.title ?? '?').join(', ')}
                    </div>
                  )}
                  {t.blocks.length > 0 && (
                    <div className="meta" style={{ marginTop: 2 }}>
                      Blokerer: {t.blocks.map((id) => byId.get(id)?.title ?? '?').join(', ')}
                    </div>
                  )}
                </div>
              ))}
              {tasks.length === 0 && <Empty>Ingen.</Empty>}
            </div>
          )
        })}
      </div>

      <div style={{ marginTop: 16 }}>
        <button className="btn" onClick={() => setCreating(true)}>+ Ny opgave</button>
      </div>

      {(editing || creating) && (
        <TaskModal
          task={editing}
          all={board ?? []}
          projectId={projectId!}
          phases={phases ?? []}
          rooms={rooms ?? []}
          suppliers={suppliers ?? []}
          onDone={() => (setEditing(null), setCreating(false), reload())}
          onClose={() => (setEditing(null), setCreating(false))}
        />
      )}
    </>
  )
}

function TaskModal({ task, all, projectId, phases, rooms, suppliers, onDone, onClose }: {
  task: BoardTask | null
  all: BoardTask[]
  projectId: string
  phases: Phase[]
  rooms: Room[]
  suppliers: Supplier[]
  onDone: () => void
  onClose: () => void
}) {
  const ZERO = '00000000-0000-0000-0000-000000000000'
  const [title, setTitle] = useState(task?.title ?? '')
  const [description, setDescription] = useState(task?.description ?? '')
  const [status, setStatus] = useState<TaskStatus>(task?.status ?? 'todo')
  const [phaseId, setPhaseId] = useState(task?.phaseId ?? '')
  const [roomId, setRoomId] = useState(task?.roomId ?? '')
  const [supplierId, setSupplierId] = useState(task?.responsibleSupplierId ?? '')
  const [newDep, setNewDep] = useState('')
  const [error, setError] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    const body = {
      title,
      description,
      status,
      phaseId: phaseId || ZERO,
      roomId: roomId || ZERO,
      responsibleSupplierId: supplierId || ZERO,
    }
    try {
      if (task) await api.patch(`/tasks/${task.id}`, body)
      else await api.post(`/projects/${projectId}/tasks`, body)
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function addDependency() {
    if (!task || !newDep) return
    try {
      await api.post(`/tasks/${task.id}/dependencies`, { dependsOnTaskId: newDep })
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function removeDependency(depId: string) {
    if (!task) return
    try {
      await api.del(`/tasks/${task.id}/dependencies/${depId}`)
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function remove() {
    if (!task || !confirmDelete(`opgaven "${task.title}"`)) return
    try {
      await api.del(`/tasks/${task.id}`)
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  const candidates = all.filter((t) => t.id !== task?.id && !task?.dependsOn.includes(t.id))

  return (
    <Modal title={task ? 'Redigér opgave' : 'Ny opgave'} onClose={onClose}>
      <form onSubmit={submit}>
        <div className="field" style={{ marginBottom: 10 }}>
          <label>Titel *</label>
          <input value={title} onChange={(e) => setTitle(e.target.value)} required autoFocus={!task} />
        </div>
        <div className="form-row">
          <div className="field">
            <label>Status</label>
            <select value={status} onChange={(e) => setStatus(e.target.value as TaskStatus)}>
              <option value="todo">Klar</option>
              <option value="blocked">Blokeret</option>
              <option value="in_progress">I gang</option>
              <option value="done">Færdig</option>
              <option value="cancelled">Annulleret</option>
            </select>
          </div>
          <div className="field">
            <label>Fase</label>
            <select value={phaseId} onChange={(e) => setPhaseId(e.target.value)}>
              <option value="">(ingen)</option>
              {phases.map((p) => (
                <option key={p.id} value={p.id}>{p.name}</option>
              ))}
            </select>
          </div>
          <div className="field">
            <label>Rum/zone</label>
            <select value={roomId} onChange={(e) => setRoomId(e.target.value)}>
              <option value="">(ingen)</option>
              {rooms.map((r) => (
                <option key={r.id} value={r.id}>{r.name}</option>
              ))}
            </select>
          </div>
        </div>
        <div className="form-row">
          <div className="field" style={{ flex: 1 }}>
            <label>Ansvarlig håndværker</label>
            <select value={supplierId} onChange={(e) => setSupplierId(e.target.value)}>
              <option value="">Mig selv</option>
              {suppliers.map((s) => (
                <option key={s.id} value={s.id}>{s.companyName}</option>
              ))}
            </select>
          </div>
        </div>
        <div className="field" style={{ marginBottom: 10 }}>
          <label>Beskrivelse</label>
          <textarea value={description} onChange={(e) => setDescription(e.target.value)} />
        </div>

        {task && (
          <>
            <h3>Afhængigheder — kan først starte når disse er færdige</h3>
            {task.dependsOn.length === 0 && <p className="hint">Ingen afhængigheder.</p>}
            {task.dependsOn.map((depId) => (
              <div key={depId} className="form-row" style={{ alignItems: 'center' }}>
                <span style={{ flex: 1 }}>{all.find((t) => t.id === depId)?.title ?? depId}</span>
                <button type="button" className="btn small secondary" onClick={() => removeDependency(depId)}>
                  Fjern
                </button>
              </div>
            ))}
            <div className="form-row">
              <div className="field" style={{ flex: 1 }}>
                <select value={newDep} onChange={(e) => setNewDep(e.target.value)}>
                  <option value="">Vælg opgave…</option>
                  {candidates.map((t) => (
                    <option key={t.id} value={t.id}>{t.title}</option>
                  ))}
                </select>
              </div>
              <button type="button" className="btn small secondary" onClick={addDependency} disabled={!newDep}>
                + Tilføj afhængighed
              </button>
            </div>
          </>
        )}

        <ErrorText error={error} />
        <div className="form-row" style={{ justifyContent: 'space-between', marginTop: 12 }}>
          <div>{task && <button type="button" className="btn danger" onClick={remove}>Slet</button>}</div>
          <div className="row-actions">
            <button type="button" className="btn secondary" onClick={onClose}>Annullér</button>
            <button className="btn">Gem</button>
          </div>
        </div>
      </form>
    </Modal>
  )
}
