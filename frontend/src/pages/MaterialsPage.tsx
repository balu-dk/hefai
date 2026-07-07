import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { api, kr, parseKr } from '../api/client'
import type { Material, MaterialStatus, Phase, Room, ShoppingListGroup, Supplier } from '../api/types'
import { confirmDelete, Empty, ErrorText, Modal, StatusBadge, useLoad } from '../components'

const ZERO = '00000000-0000-0000-0000-000000000000'

export default function MaterialsPage() {
  const { projectId } = useParams()
  const { data: materials, error, reload } = useLoad(
    () => api.get<Material[]>(`/projects/${projectId}/materials`), [projectId])
  const { data: shopping, reload: reloadShopping } = useLoad(
    () => api.get<ShoppingListGroup[]>(`/projects/${projectId}/materials/shopping-list`), [projectId])
  const { data: suppliers } = useLoad(() => api.get<Supplier[]>(`/projects/${projectId}/suppliers`), [projectId])
  const { data: phases } = useLoad(() => api.get<Phase[]>(`/projects/${projectId}/phases`), [projectId])
  const { data: rooms } = useLoad(() => api.get<Room[]>(`/projects/${projectId}/rooms`), [projectId])

  const [editing, setEditing] = useState<Material | null>(null)
  const [creating, setCreating] = useState(false)
  const [showShopping, setShowShopping] = useState(false)

  const reloadAll = () => (reload(), reloadShopping())

  async function advanceStatus(m: Material) {
    const next: Record<MaterialStatus, MaterialStatus> = {
      needed: 'ordered', ordered: 'delivered', delivered: 'in_stock', in_stock: 'used', used: 'used',
    }
    await api.patch(`/materials/${m.id}`, { status: next[m.status] })
    reloadAll()
  }

  return (
    <>
      <h1>Materialer & indkøb</h1>
      <p className="page-sub">Skal bruges → bestilt → leveret → på lager → brugt.</p>
      <ErrorText error={error} />

      <div className="card">
        <table className="tbl">
          <thead>
            <tr>
              <th>Materiale</th><th>Antal</th><th>Status</th><th>Leverandør</th>
              <th>Fase / rum</th><th className="num">Pris</th><th></th>
            </tr>
          </thead>
          <tbody>
            {materials?.map((m) => (
              <tr key={m.id}>
                <td>
                  <strong>{m.name}</strong>
                  {m.spec && <div style={{ fontSize: 12, color: 'var(--text-dim)' }}>{m.spec}</div>}
                </td>
                <td>{m.quantity} {m.unit}</td>
                <td><StatusBadge status={m.status} /></td>
                <td>{suppliers?.find((s) => s.id === m.supplierId)?.companyName ?? '–'}</td>
                <td style={{ fontSize: 12.5 }}>
                  {phases?.find((p) => p.id === m.phaseId)?.name ?? ''}
                  {m.roomId && <> · {rooms?.find((r) => r.id === m.roomId)?.name}</>}
                </td>
                <td className="num">{m.unitPriceOre != null ? kr(Math.round(m.quantity * m.unitPriceOre)) : '–'}</td>
                <td className="row-actions">
                  {m.status !== 'used' && (
                    <button className="btn small secondary" onClick={() => advanceStatus(m)} title="Ryk til næste status">→</button>
                  )}
                  <button className="btn small secondary" onClick={() => setEditing(m)}>Redigér</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {materials?.length === 0 && <Empty>Ingen materialer endnu.</Empty>}
      </div>

      <div className="row-actions">
        <button className="btn" onClick={() => setCreating(true)}>+ Materiale</button>
        <button className="btn secondary" onClick={() => setShowShopping(true)}>Indkøbsliste</button>
      </div>

      {showShopping && (
        <Modal title="Indkøbsliste — det der mangler at blive købt" wide onClose={() => setShowShopping(false)}>
          {shopping?.length === 0 && <Empty>Alt er købt ind!</Empty>}
          {shopping?.map((g) => (
            <div key={g.supplierName} className="card">
              <h3>
                {g.supplierName}
                {g.totalOre > 0 && <span style={{ float: 'right' }}>{kr(g.totalOre)}</span>}
              </h3>
              <table className="tbl">
                <tbody>
                  {g.materials.map((m) => (
                    <tr key={m.id}>
                      <td>{m.name} {m.spec && <span style={{ color: 'var(--text-dim)' }}>({m.spec})</span>}</td>
                      <td>{m.quantity} {m.unit}</td>
                      <td className="num">{m.unitPriceOre != null ? kr(Math.round(m.quantity * m.unitPriceOre)) : ''}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ))}
        </Modal>
      )}

      {(editing || creating) && (
        <MaterialModal
          material={editing} projectId={projectId!}
          suppliers={suppliers ?? []} phases={phases ?? []} rooms={rooms ?? []}
          onDone={() => (setEditing(null), setCreating(false), reloadAll())}
          onClose={() => (setEditing(null), setCreating(false))}
        />
      )}
    </>
  )
}

function MaterialModal({ material, projectId, suppliers, phases, rooms, onDone, onClose }: {
  material: Material | null
  projectId: string
  suppliers: Supplier[]
  phases: Phase[]
  rooms: Room[]
  onDone: () => void
  onClose: () => void
}) {
  const [name, setName] = useState(material?.name ?? '')
  const [spec, setSpec] = useState(material?.spec ?? '')
  const [quantity, setQuantity] = useState(String(material?.quantity ?? 1))
  const [unit, setUnit] = useState(material?.unit ?? 'stk')
  const [price, setPrice] = useState(
    material?.unitPriceOre != null ? (material.unitPriceOre / 100).toFixed(2).replace('.', ',') : '')
  const [status, setStatus] = useState<MaterialStatus>(material?.status ?? 'needed')
  const [supplierId, setSupplierId] = useState(material?.supplierId ?? '')
  const [phaseId, setPhaseId] = useState(material?.phaseId ?? '')
  const [roomId, setRoomId] = useState(material?.roomId ?? '')
  const [error, setError] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    const body: Record<string, unknown> = {
      name, spec, quantity: Number(quantity), unit, status,
      supplierId: supplierId || ZERO, phaseId: phaseId || ZERO, roomId: roomId || ZERO,
    }
    if (price.trim()) {
      const ore = parseKr(price)
      if (ore === null) return setError('Ugyldig pris')
      body.unitPriceOre = ore
    }
    try {
      if (material) await api.patch(`/materials/${material.id}`, body)
      else await api.post(`/projects/${projectId}/materials`, body)
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function remove() {
    if (!material || !confirmDelete(`materialet "${material.name}"`)) return
    await api.del(`/materials/${material.id}`)
    onDone()
  }

  return (
    <Modal title={material ? 'Redigér materiale' : 'Nyt materiale'} onClose={onClose}>
      <form onSubmit={submit}>
        <div className="form-row">
          <div className="field" style={{ flex: 2 }}>
            <label>Navn *</label>
            <input value={name} onChange={(e) => setName(e.target.value)} required autoFocus={!material} />
          </div>
          <div className="field">
            <label>Spec/dimension</label>
            <input value={spec} onChange={(e) => setSpec(e.target.value)} placeholder="45x195 C24" />
          </div>
        </div>
        <div className="form-row">
          <div className="field">
            <label>Antal</label>
            <input type="number" step="0.001" min="0" value={quantity} onChange={(e) => setQuantity(e.target.value)} />
          </div>
          <div className="field">
            <label>Enhed</label>
            <input value={unit} onChange={(e) => setUnit(e.target.value)} style={{ width: 80 }} />
          </div>
          <div className="field">
            <label>Stykpris (kr.)</label>
            <input value={price} onChange={(e) => setPrice(e.target.value)} placeholder="185,00" />
          </div>
          <div className="field">
            <label>Status</label>
            <select value={status} onChange={(e) => setStatus(e.target.value as MaterialStatus)}>
              <option value="needed">Skal bruges</option>
              <option value="ordered">Bestilt</option>
              <option value="delivered">Leveret</option>
              <option value="in_stock">På lager</option>
              <option value="used">Brugt</option>
            </select>
          </div>
        </div>
        <div className="form-row">
          <div className="field">
            <label>Leverandør</label>
            <select value={supplierId} onChange={(e) => setSupplierId(e.target.value)}>
              <option value="">(ingen)</option>
              {suppliers.map((s) => <option key={s.id} value={s.id}>{s.companyName}</option>)}
            </select>
          </div>
          <div className="field">
            <label>Fase</label>
            <select value={phaseId} onChange={(e) => setPhaseId(e.target.value)}>
              <option value="">(ingen)</option>
              {phases.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
            </select>
          </div>
          <div className="field">
            <label>Rum</label>
            <select value={roomId} onChange={(e) => setRoomId(e.target.value)}>
              <option value="">(ingen)</option>
              {rooms.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}
            </select>
          </div>
        </div>
        <ErrorText error={error} />
        <div className="form-row" style={{ justifyContent: 'space-between', marginTop: 12 }}>
          <div>{material && <button type="button" className="btn danger" onClick={remove}>Slet</button>}</div>
          <div className="row-actions">
            <button type="button" className="btn secondary" onClick={onClose}>Annullér</button>
            <button className="btn">Gem</button>
          </div>
        </div>
      </form>
    </Modal>
  )
}
