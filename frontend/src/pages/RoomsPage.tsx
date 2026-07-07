import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Doc, Material, Room, RoomKind, Task } from '../api/types'
import { confirmDelete, Empty, ErrorText, Modal, StatusBadge, useLoad } from '../components'

export default function RoomsPage() {
  const { projectId } = useParams()
  const { data: rooms, error, reload } = useLoad(() => api.get<Room[]>(`/projects/${projectId}/rooms`), [projectId])
  const [editing, setEditing] = useState<Room | null>(null)
  const [creating, setCreating] = useState(false)
  const [selected, setSelected] = useState<Room | null>(null)

  return (
    <>
      <h1>Rum & zoner</h1>
      <p className="page-sub">Alt om ét rum samlet ét sted — opgaver, materialer og dokumenter.</p>
      <ErrorText error={error} />

      <div className="grid cols-3">
        {rooms?.map((r) => (
          <div key={r.id} className="card" style={{ marginBottom: 0, cursor: 'pointer' }} onClick={() => setSelected(r)}>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <strong>{r.name}</strong>
              <StatusBadge status={r.kind === 'room' ? 'in_progress' : r.kind} />
            </div>
            <div style={{ fontSize: 13, color: 'var(--text-dim)', marginTop: 4 }}>
              {r.kind === 'room' ? 'Rum' : r.kind === 'zone' ? 'Zone' : 'Udendørs'}
              {r.areaM2 != null && <> · {r.areaM2} m²</>}
            </div>
          </div>
        ))}
      </div>
      {rooms?.length === 0 && <Empty>Ingen rum endnu.</Empty>}

      <div style={{ marginTop: 16 }}>
        <button className="btn" onClick={() => setCreating(true)}>+ Rum/zone</button>
      </div>

      {selected && (
        <RoomDetailModal
          room={selected} projectId={projectId!}
          onEdit={() => (setEditing(selected), setSelected(null))}
          onClose={() => setSelected(null)}
        />
      )}
      {(editing || creating) && (
        <RoomModal
          room={editing} projectId={projectId!}
          onDone={() => (setEditing(null), setCreating(false), reload())}
          onClose={() => (setEditing(null), setCreating(false))}
        />
      )}
    </>
  )
}

function RoomDetailModal({ room, projectId, onEdit, onClose }: {
  room: Room
  projectId: string
  onEdit: () => void
  onClose: () => void
}) {
  const { data: tasks } = useLoad(() => api.get<Task[]>(`/projects/${projectId}/tasks`), [projectId])
  const { data: materials } = useLoad(() => api.get<Material[]>(`/projects/${projectId}/materials`), [projectId])
  const { data: docs } = useLoad(
    () => api.get<Doc[]>(`/projects/${projectId}/documents?targetType=room&targetId=${room.id}`),
    [projectId, room.id],
  )

  const roomTasks = tasks?.filter((t) => t.roomId === room.id) ?? []
  const roomMaterials = materials?.filter((m) => m.roomId === room.id) ?? []

  return (
    <Modal title={room.name} wide onClose={onClose}>
      {room.description && <p>{room.description}</p>}
      <div className="grid cols-3">
        <div>
          <h3>Opgaver ({roomTasks.length})</h3>
          {roomTasks.map((t) => (
            <div key={t.id} className="task-card">
              <div className="title">{t.title}</div>
              <StatusBadge status={t.status} />
            </div>
          ))}
          {roomTasks.length === 0 && <Empty>Ingen.</Empty>}
        </div>
        <div>
          <h3>Materialer ({roomMaterials.length})</h3>
          {roomMaterials.map((m) => (
            <div key={m.id} className="task-card">
              <div className="title">{m.name}</div>
              <div className="meta">{m.quantity} {m.unit} · <StatusBadge status={m.status} /></div>
            </div>
          ))}
          {roomMaterials.length === 0 && <Empty>Ingen.</Empty>}
        </div>
        <div>
          <h3>Dokumenter ({docs?.length ?? 0})</h3>
          {docs?.map((d) => (
            <div key={d.id} className="task-card">
              <div className="title">{d.title}</div>
              <div className="meta">{d.kind}</div>
            </div>
          ))}
          {docs?.length === 0 && <Empty>Ingen.</Empty>}
          <p className="hint">
            Knyt dokumenter til rummet under <Link to={`/projects/${projectId}/documents`}>Dokumenter</Link>.
          </p>
        </div>
      </div>
      <div className="form-row" style={{ justifyContent: 'flex-end', marginTop: 10 }}>
        <button className="btn secondary" onClick={onEdit}>Redigér rum</button>
        <button className="btn" onClick={onClose}>Luk</button>
      </div>
    </Modal>
  )
}

function RoomModal({ room, projectId, onDone, onClose }: {
  room: Room | null
  projectId: string
  onDone: () => void
  onClose: () => void
}) {
  const [name, setName] = useState(room?.name ?? '')
  const [kind, setKind] = useState<RoomKind>(room?.kind ?? 'room')
  const [area, setArea] = useState(room?.areaM2 != null ? String(room.areaM2) : '')
  const [description, setDescription] = useState(room?.description ?? '')
  const [error, setError] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    const body = { name, kind, description, areaM2: area ? Number(area) : undefined }
    try {
      if (room) await api.patch(`/rooms/${room.id}`, body)
      else await api.post(`/projects/${projectId}/rooms`, body)
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function remove() {
    if (!room || !confirmDelete(`rummet "${room.name}"`)) return
    await api.del(`/rooms/${room.id}`)
    onDone()
  }

  return (
    <Modal title={room ? 'Redigér rum' : 'Nyt rum/zone'} onClose={onClose}>
      <form onSubmit={submit}>
        <div className="form-row">
          <div className="field" style={{ flex: 2 }}>
            <label>Navn *</label>
            <input value={name} onChange={(e) => setName(e.target.value)} required autoFocus={!room} />
          </div>
          <div className="field">
            <label>Type</label>
            <select value={kind} onChange={(e) => setKind(e.target.value as RoomKind)}>
              <option value="room">Rum</option>
              <option value="zone">Zone</option>
              <option value="outdoor">Udendørs</option>
            </select>
          </div>
          <div className="field">
            <label>Areal (m²)</label>
            <input type="number" step="0.1" min="0" value={area} onChange={(e) => setArea(e.target.value)} />
          </div>
        </div>
        <div className="field">
          <label>Beskrivelse</label>
          <textarea value={description} onChange={(e) => setDescription(e.target.value)} />
        </div>
        <ErrorText error={error} />
        <div className="form-row" style={{ justifyContent: 'space-between', marginTop: 12 }}>
          <div>{room && <button type="button" className="btn danger" onClick={remove}>Slet</button>}</div>
          <div className="row-actions">
            <button type="button" className="btn secondary" onClick={onClose}>Annullér</button>
            <button className="btn">Gem</button>
          </div>
        </div>
      </form>
    </Modal>
  )
}
