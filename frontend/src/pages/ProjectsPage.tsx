import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { Project } from '../api/types'
import { Empty, ErrorText, Modal, StatusBadge, statusLabel, useLoad } from '../components'
import { useAuth } from '../auth'

export default function ProjectsPage() {
  const { user, logout } = useAuth()
  const { data: projects, error, reload } = useLoad(() => api.get<Project[]>('/projects'), [])
  const [showCreate, setShowCreate] = useState(false)

  return (
    <div style={{ maxWidth: 860, margin: '0 auto', padding: '40px 20px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}>
        <h1>
          Hef<span style={{ color: 'var(--accent)' }}>ai</span>
        </h1>
        <div style={{ fontSize: 13, color: 'var(--text-dim)' }}>
          {user?.displayName} ·{' '}
          <a href="#" onClick={(e) => (e.preventDefault(), logout())}>
            Log ud
          </a>
        </div>
      </div>
      <p className="page-sub">Dine byggeprojekter</p>
      <ErrorText error={error} />

      <div className="grid cols-2">
        {projects?.map((p) => (
          <Link key={p.id} to={`/projects/${p.id}`} style={{ textDecoration: 'none', color: 'inherit' }}>
            <div className="card" style={{ marginBottom: 0, height: '100%' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', gap: 8 }}>
                <strong style={{ fontSize: 16 }}>{p.name}</strong>
                <StatusBadge status={p.status} />
              </div>
              <div style={{ color: 'var(--text-dim)', fontSize: 13, marginTop: 6 }}>
                {statusLabel(p.kind)}
                {p.address && <> · {p.address}</>}
                {p.municipality && <> · {p.municipality} Kommune</>}
              </div>
              {p.description && <p style={{ marginBottom: 0 }}>{p.description}</p>}
            </div>
          </Link>
        ))}
      </div>
      {projects && projects.length === 0 && <Empty>Ingen projekter endnu — opret dit første.</Empty>}

      <div style={{ marginTop: 20 }}>
        <button className="btn" onClick={() => setShowCreate(true)}>
          + Nyt projekt
        </button>
      </div>

      {showCreate && <CreateProjectModal onDone={() => (setShowCreate(false), reload())} onClose={() => setShowCreate(false)} />}
    </div>
  )
}

function CreateProjectModal({ onDone, onClose }: { onDone: () => void; onClose: () => void }) {
  const [name, setName] = useState('')
  const [kind, setKind] = useState('new_build')
  const [address, setAddress] = useState('')
  const [municipality, setMunicipality] = useState('')
  const [cadastralId, setCadastralId] = useState('')
  const [plotArea, setPlotArea] = useState('')
  const [description, setDescription] = useState('')
  const [error, setError] = useState<string | null>(null)

  async function submit(e: FormEvent) {
    e.preventDefault()
    try {
      await api.post('/projects', {
        name,
        kind,
        address,
        municipality,
        cadastralId,
        description,
        plotAreaM2: plotArea ? Number(plotArea) : null,
      })
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  return (
    <Modal title="Nyt projekt" onClose={onClose}>
      <form onSubmit={submit}>
        <div className="form-row">
          <div className="field" style={{ flex: 2 }}>
            <label>Navn *</label>
            <input value={name} onChange={(e) => setName(e.target.value)} required autoFocus />
          </div>
          <div className="field">
            <label>Type</label>
            <select value={kind} onChange={(e) => setKind(e.target.value)}>
              <option value="new_build">Nybyggeri</option>
              <option value="renovation">Renovering</option>
              <option value="extension">Tilbygning</option>
              <option value="other">Andet</option>
            </select>
          </div>
        </div>
        <div className="form-row">
          <div className="field" style={{ flex: 2 }}>
            <label>Adresse</label>
            <input value={address} onChange={(e) => setAddress(e.target.value)} />
          </div>
          <div className="field">
            <label>Kommune</label>
            <input value={municipality} onChange={(e) => setMunicipality(e.target.value)} />
          </div>
        </div>
        <div className="form-row">
          <div className="field">
            <label>Matrikelnr.</label>
            <input value={cadastralId} onChange={(e) => setCadastralId(e.target.value)} />
          </div>
          <div className="field">
            <label>Grundareal (m²)</label>
            <input type="number" step="0.1" min="0" value={plotArea} onChange={(e) => setPlotArea(e.target.value)} />
          </div>
        </div>
        <div className="field">
          <label>Beskrivelse</label>
          <textarea value={description} onChange={(e) => setDescription(e.target.value)} />
        </div>
        <ErrorText error={error} />
        <div className="form-row" style={{ marginTop: 12, justifyContent: 'flex-end' }}>
          <button type="button" className="btn secondary" onClick={onClose}>
            Annullér
          </button>
          <button className="btn">Opret projekt</button>
        </div>
      </form>
    </Modal>
  )
}
