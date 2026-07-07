import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { api, kr, parseKr } from '../api/client'
import type { Supplier } from '../api/types'
import { confirmDelete, Empty, ErrorText, Modal, useLoad } from '../components'

export default function SuppliersPage() {
  const { projectId } = useParams()
  const { data: suppliers, error, reload } = useLoad(
    () => api.get<Supplier[]>(`/projects/${projectId}/suppliers`), [projectId])
  const [editing, setEditing] = useState<Supplier | null>(null)
  const [creating, setCreating] = useState(false)

  return (
    <>
      <h1>Leverandører & håndværkere</h1>
      <p className="page-sub">Kontaktregister med fag og noter.</p>
      <ErrorText error={error} />

      <div className="grid cols-2">
        {suppliers?.map((s) => (
          <div key={s.id} className="card" style={{ marginBottom: 0 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <strong>{s.companyName}</strong>
              <button className="btn small secondary" onClick={() => setEditing(s)}>Redigér</button>
            </div>
            <div style={{ color: 'var(--text-dim)', fontSize: 13, margin: '4px 0' }}>
              {s.trade}{s.contactPerson && <> · {s.contactPerson}</>}
            </div>
            <div style={{ fontSize: 13 }}>
              {s.phone && <div>☎ {s.phone}</div>}
              {s.email && <div>✉ {s.email}</div>}
              {s.hourlyRateOre != null && <div>⏱ {kr(s.hourlyRateOre)}/time</div>}
            </div>
            {s.notes && <p style={{ fontSize: 13, marginBottom: 0 }}>{s.notes}</p>}
          </div>
        ))}
      </div>
      {suppliers?.length === 0 && <Empty>Ingen leverandører endnu.</Empty>}

      <div style={{ marginTop: 16 }}>
        <button className="btn" onClick={() => setCreating(true)}>+ Leverandør</button>
      </div>

      {(editing || creating) && (
        <SupplierModal
          supplier={editing} projectId={projectId!}
          onDone={() => (setEditing(null), setCreating(false), reload())}
          onClose={() => (setEditing(null), setCreating(false))}
        />
      )}
    </>
  )
}

function SupplierModal({ supplier, projectId, onDone, onClose }: {
  supplier: Supplier | null
  projectId: string
  onDone: () => void
  onClose: () => void
}) {
  const [companyName, setCompanyName] = useState(supplier?.companyName ?? '')
  const [contactPerson, setContactPerson] = useState(supplier?.contactPerson ?? '')
  const [trade, setTrade] = useState(supplier?.trade ?? '')
  const [phone, setPhone] = useState(supplier?.phone ?? '')
  const [email, setEmail] = useState(supplier?.email ?? '')
  const [hourlyRate, setHourlyRate] = useState(
    supplier?.hourlyRateOre != null ? (supplier.hourlyRateOre / 100).toFixed(2).replace('.', ',') : '')
  const [notes, setNotes] = useState(supplier?.notes ?? '')
  const [error, setError] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    const body: Record<string, unknown> = { companyName, contactPerson, trade, phone, email, notes }
    if (hourlyRate.trim()) {
      const ore = parseKr(hourlyRate)
      if (ore === null) return setError('Ugyldig timepris — brug fx 585,00')
      body.hourlyRateOre = ore
    } else if (supplier?.hourlyRateOre != null) {
      body.hourlyRateOre = -1 // ryd feltet
    }
    try {
      if (supplier) await api.patch(`/suppliers/${supplier.id}`, body)
      else await api.post(`/projects/${projectId}/suppliers`, body)
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function remove() {
    if (!supplier || !confirmDelete(supplier.companyName)) return
    await api.del(`/suppliers/${supplier.id}`)
    onDone()
  }

  return (
    <Modal title={supplier ? 'Redigér leverandør' : 'Ny leverandør'} onClose={onClose}>
      <form onSubmit={submit}>
        <div className="form-row">
          <div className="field" style={{ flex: 2 }}>
            <label>Firma *</label>
            <input value={companyName} onChange={(e) => setCompanyName(e.target.value)} required autoFocus={!supplier} />
          </div>
          <div className="field">
            <label>Fag</label>
            <input value={trade} onChange={(e) => setTrade(e.target.value)} placeholder="Tømrer, VVS…" />
          </div>
        </div>
        <div className="form-row">
          <div className="field">
            <label>Kontaktperson</label>
            <input value={contactPerson} onChange={(e) => setContactPerson(e.target.value)} />
          </div>
          <div className="field">
            <label>Telefon</label>
            <input value={phone} onChange={(e) => setPhone(e.target.value)} />
          </div>
          <div className="field">
            <label>E-mail</label>
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          </div>
          <div className="field">
            <label>Timepris (kr.)</label>
            <input value={hourlyRate} onChange={(e) => setHourlyRate(e.target.value)} placeholder="585,00" />
          </div>
        </div>
        <div className="field">
          <label>Noter</label>
          <textarea value={notes} onChange={(e) => setNotes(e.target.value)} />
        </div>
        <ErrorText error={error} />
        <div className="form-row" style={{ justifyContent: 'space-between', marginTop: 12 }}>
          <div>{supplier && <button type="button" className="btn danger" onClick={remove}>Slet</button>}</div>
          <div className="row-actions">
            <button type="button" className="btn secondary" onClick={onClose}>Annullér</button>
            <button className="btn">Gem</button>
          </div>
        </div>
      </form>
    </Modal>
  )
}
