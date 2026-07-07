import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { api, fetchDocumentObjectURL, formatDate } from '../api/client'
import type {
  CalcMethod, CalculationEstimate, ElementType, EngineerReview, LoadType,
  StructuralElement, StructuralLoad, StructuralMaterialType, StructuralPackage,
} from '../api/types'
import { confirmDelete, Empty, ErrorText, Modal, StatusBadge, useLoad } from '../components'

export default function StructuralPage() {
  const { projectId } = useParams()
  const { data: elements, reload: reloadElements } = useLoad(
    () => api.get<StructuralElement[]>(`/projects/${projectId}/structural-elements`), [projectId])
  const { data: loads, reload: reloadLoads } = useLoad(
    () => api.get<StructuralLoad[]>(`/projects/${projectId}/loads`), [projectId])
  const { data: estimates, reload: reloadEstimates } = useLoad(
    () => api.get<CalculationEstimate[]>(`/projects/${projectId}/estimates`), [projectId])
  const { data: packages, reload: reloadPackages } = useLoad(
    () => api.get<StructuralPackage[]>(`/projects/${projectId}/structural-packages`), [projectId])
  const { data: methods } = useLoad(() => api.get<CalcMethod[]>(`/calc/methods`), [])

  const [editElement, setEditElement] = useState<StructuralElement | null>(null)
  const [newElement, setNewElement] = useState(false)
  const [newLoad, setNewLoad] = useState(false)
  const [runCalc, setRunCalc] = useState(false)
  const [showEstimate, setShowEstimate] = useState<CalculationEstimate | null>(null)
  const [reviewPackage, setReviewPackage] = useState<StructuralPackage | null>(null)
  const [showReviews, setShowReviews] = useState<StructuralPackage | null>(null)
  const [error, setError] = useState<string | null>(null)

  const elementName = (id: string | null) => elements?.find((e) => e.id === id)?.name ?? 'Projekt-niveau'

  async function createPackage() {
    setError(null)
    try {
      await api.post(`/projects/${projectId}/structural-packages`, { title: '' })
      reloadPackages()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function openPackage(p: StructuralPackage) {
    if (!p.documentId) return
    window.open(await fetchDocumentObjectURL(p.documentId), '_blank')
  }

  async function markSent(p: StructuralPackage) {
    await api.patch(`/structural-packages/${p.id}/status`, { status: 'sent' })
    reloadPackages()
  }

  return (
    <>
      <h1>Statiker-forberedelse</h1>
      <div className="draft-banner">
        KLADDE — alt her er et vejledende grundlag. Endelig statisk dokumentation kræver kontrol og
        underskrift fra en autoriseret/anerkendt statiker.
      </div>

      <h2>Konstruktionselementer</h2>
      <div className="card">
        <table className="tbl">
          <thead>
            <tr><th>Navn</th><th>Type</th><th>Materiale</th><th>Bærende</th><th>Geometri</th><th></th></tr>
          </thead>
          <tbody>
            {elements?.map((e) => (
              <tr key={e.id}>
                <td><strong>{e.name}</strong></td>
                <td>{e.elementType}</td>
                <td>{e.material} {e.materialSpec}</td>
                <td>{e.isLoadBearing ? 'Ja' : 'Nej'}</td>
                <td style={{ fontSize: 12 }}>{JSON.stringify(e.geometry)}</td>
                <td><button className="btn small secondary" onClick={() => setEditElement(e)}>Redigér</button></td>
              </tr>
            ))}
          </tbody>
        </table>
        {elements?.length === 0 && <Empty>Ingen elementer endnu.</Empty>}
      </div>
      <button className="btn" onClick={() => setNewElement(true)}>+ Element</button>

      <h2>Laster</h2>
      <div className="card">
        <table className="tbl">
          <thead>
            <tr><th>Type</th><th className="num">Værdi</th><th>Omfang</th><th>Reference</th><th>Status</th><th></th></tr>
          </thead>
          <tbody>
            {loads?.map((l) => (
              <tr key={l.id}>
                <td>{l.loadType}</td>
                <td className="num">{l.value} {l.unit}</td>
                <td>{elementName(l.structuralElementId)}</td>
                <td style={{ fontSize: 12.5 }}>{l.standardReference || '–'}</td>
                <td><StatusBadge status={l.status} /></td>
                <td>
                  <button className="btn small secondary" onClick={async () => {
                    if (!confirmDelete('lasten')) return
                    await api.del(`/loads/${l.id}`)
                    reloadLoads()
                  }}>Slet</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {loads?.length === 0 && <Empty>Ingen laster endnu — kør fx snelast-beregningen og registrér resultatet.</Empty>}
      </div>
      <button className="btn" onClick={() => setNewLoad(true)}>+ Last</button>

      <h2>Vejledende beregninger</h2>
      <div className="card">
        <table className="tbl">
          <thead>
            <tr><th>Metode</th><th>Element</th><th>Standard</th><th>Status</th><th>Kørt</th><th></th></tr>
          </thead>
          <tbody>
            {estimates?.filter((e) => e.status !== 'superseded').map((e) => (
              <tr key={e.id}>
                <td>{e.method}</td>
                <td>{elementName(e.structuralElementId)}</td>
                <td style={{ fontSize: 12.5 }}>{e.standardReference}</td>
                <td><StatusBadge status={e.status} /></td>
                <td>{formatDate(e.createdAt)}</td>
                <td><button className="btn small secondary" onClick={() => setShowEstimate(e)}>Vis</button></td>
              </tr>
            ))}
          </tbody>
        </table>
        {estimates?.length === 0 && <Empty>Ingen beregninger kørt endnu.</Empty>}
      </div>
      <button className="btn" onClick={() => setRunCalc(true)}>+ Kør beregning</button>

      <h2>Statiker-pakker & feedback</h2>
      <ErrorText error={error} />
      <div className="card">
        <table className="tbl">
          <thead>
            <tr><th>Version</th><th>Titel</th><th>Status</th><th>Sendt</th><th></th></tr>
          </thead>
          <tbody>
            {packages?.map((p) => (
              <tr key={p.id}>
                <td>v{p.versionNo}</td>
                <td>{p.title}</td>
                <td><StatusBadge status={p.status} /></td>
                <td>{formatDate(p.sentAt)}</td>
                <td className="row-actions">
                  <button className="btn small secondary" onClick={() => openPackage(p)}>PDF</button>
                  {p.status === 'draft' && <button className="btn small secondary" onClick={() => markSent(p)}>Markér sendt</button>}
                  <button className="btn small secondary" onClick={() => setReviewPackage(p)}>Registrér svar</button>
                  <button className="btn small secondary" onClick={() => setShowReviews(p)}>Svar</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {packages?.length === 0 && <Empty>Ingen pakker endnu.</Empty>}
      </div>
      <button className="btn" onClick={createPackage}>+ Generér statiker-pakke</button>

      {(editElement || newElement) && (
        <ElementModal element={editElement} projectId={projectId!}
          onDone={() => (setEditElement(null), setNewElement(false), reloadElements())}
          onClose={() => (setEditElement(null), setNewElement(false))} />
      )}
      {newLoad && (
        <LoadModal projectId={projectId!} elements={elements ?? []}
          onDone={() => (setNewLoad(false), reloadLoads())} onClose={() => setNewLoad(false)} />
      )}
      {runCalc && (
        <CalcModal projectId={projectId!} elements={elements ?? []} methods={methods ?? []}
          onDone={() => (setRunCalc(false), reloadEstimates())} onClose={() => setRunCalc(false)} />
      )}
      {showEstimate && <EstimateModal estimate={showEstimate} onClose={() => setShowEstimate(null)} />}
      {reviewPackage && (
        <ReviewModal pkg={reviewPackage} elements={elements ?? []} loads={loads ?? []}
          estimates={(estimates ?? []).filter((e) => e.status !== 'superseded')}
          onDone={() => (setReviewPackage(null), reloadPackages(), reloadLoads(), reloadEstimates())}
          onClose={() => setReviewPackage(null)} />
      )}
      {showReviews && <ReviewsListModal pkg={showReviews} onClose={() => setShowReviews(null)} />}
    </>
  )
}

function ElementModal({ element, projectId, onDone, onClose }: {
  element: StructuralElement | null
  projectId: string
  onDone: () => void
  onClose: () => void
}) {
  const [name, setName] = useState(element?.name ?? '')
  const [elementType, setElementType] = useState<ElementType>(element?.elementType ?? 'beam')
  const [material, setMaterial] = useState<StructuralMaterialType>(element?.material ?? 'timber')
  const [materialSpec, setMaterialSpec] = useState(element?.materialSpec ?? '')
  const [isLoadBearing, setIsLoadBearing] = useState(element?.isLoadBearing ?? true)
  const [geometry, setGeometry] = useState(JSON.stringify(element?.geometry ?? {}, null, 0))
  const [notes, setNotes] = useState(element?.notes ?? '')
  const [error, setError] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    let geo: unknown
    try {
      geo = JSON.parse(geometry || '{}')
    } catch {
      return setError('Geometri skal være gyldig JSON, fx {"spanM": 4.0}')
    }
    const body = { name, elementType, material, materialSpec, isLoadBearing, geometry: geo, notes }
    try {
      if (element) await api.patch(`/structural-elements/${element.id}`, body)
      else await api.post(`/projects/${projectId}/structural-elements`, body)
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function remove() {
    if (!element || !confirmDelete(`elementet "${element.name}"`)) return
    await api.del(`/structural-elements/${element.id}`)
    onDone()
  }

  return (
    <Modal title={element ? 'Redigér element' : 'Nyt konstruktionselement'} onClose={onClose}>
      <form onSubmit={submit}>
        <div className="form-row">
          <div className="field" style={{ flex: 2 }}>
            <label>Navn *</label>
            <input value={name} onChange={(e) => setName(e.target.value)} required autoFocus={!element} placeholder="Bjælke over stue" />
          </div>
          <div className="field">
            <label>Type</label>
            <select value={elementType} onChange={(e) => setElementType(e.target.value as ElementType)}>
              <option value="beam">Bjælke</option>
              <option value="column">Søjle</option>
              <option value="wall">Væg</option>
              <option value="foundation">Fundament</option>
              <option value="roof">Tagkonstruktion</option>
              <option value="slab">Dæk</option>
              <option value="other">Andet</option>
            </select>
          </div>
        </div>
        <div className="form-row">
          <div className="field">
            <label>Materiale</label>
            <select value={material} onChange={(e) => setMaterial(e.target.value as StructuralMaterialType)}>
              <option value="timber">Træ</option>
              <option value="steel">Stål</option>
              <option value="concrete">Beton</option>
              <option value="masonry">Murværk</option>
              <option value="other">Andet</option>
            </select>
          </div>
          <div className="field">
            <label>Spec</label>
            <input value={materialSpec} onChange={(e) => setMaterialSpec(e.target.value)} placeholder="C24, S235…" />
          </div>
          <div className="field">
            <label style={{ marginTop: 18 }}>
              <input type="checkbox" checked={isLoadBearing} onChange={(e) => setIsLoadBearing(e.target.checked)} /> Bærende
            </label>
          </div>
        </div>
        <div className="field" style={{ marginBottom: 10 }}>
          <label>Geometri (JSON — fx spændvidde, tværsnit)</label>
          <textarea rows={3} value={geometry} onChange={(e) => setGeometry(e.target.value)}
            placeholder='{"spanM": 4.0, "spacingM": 0.6, "widthMm": 45, "heightMm": 195}' />
        </div>
        <div className="field">
          <label>Noter</label>
          <textarea rows={2} value={notes} onChange={(e) => setNotes(e.target.value)} />
        </div>
        <ErrorText error={error} />
        <div className="form-row" style={{ justifyContent: 'space-between', marginTop: 12 }}>
          <div>{element && <button type="button" className="btn danger" onClick={remove}>Slet</button>}</div>
          <div className="row-actions">
            <button type="button" className="btn secondary" onClick={onClose}>Annullér</button>
            <button className="btn">Gem</button>
          </div>
        </div>
      </form>
    </Modal>
  )
}

function LoadModal({ projectId, elements, onDone, onClose }: {
  projectId: string
  elements: StructuralElement[]
  onDone: () => void
  onClose: () => void
}) {
  const [loadType, setLoadType] = useState<LoadType>('snow')
  const [value, setValue] = useState('')
  const [unit, setUnit] = useState('kN/m²')
  const [elementId, setElementId] = useState('')
  const [reference, setReference] = useState('')
  const [notes, setNotes] = useState('')
  const [error, setError] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    try {
      await api.post(`/projects/${projectId}/loads`, {
        loadType, value: Number(value), unit,
        structuralElementId: elementId || undefined,
        standardReference: reference, notes,
      })
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  return (
    <Modal title="Ny last" onClose={onClose}>
      <form onSubmit={submit}>
        <div className="form-row">
          <div className="field">
            <label>Type</label>
            <select value={loadType} onChange={(e) => setLoadType(e.target.value as LoadType)}>
              <option value="dead">Egenlast</option>
              <option value="live">Nyttelast</option>
              <option value="snow">Snelast</option>
              <option value="wind">Vindlast</option>
              <option value="point">Punktlast</option>
              <option value="line">Linjelast</option>
              <option value="other">Andet</option>
            </select>
          </div>
          <div className="field">
            <label>Værdi *</label>
            <input type="number" step="0.001" value={value} onChange={(e) => setValue(e.target.value)} required />
          </div>
          <div className="field">
            <label>Enhed</label>
            <input value={unit} onChange={(e) => setUnit(e.target.value)} style={{ width: 90 }} />
          </div>
        </div>
        <div className="form-row">
          <div className="field" style={{ flex: 1 }}>
            <label>Element (tom = hele projektet)</label>
            <select value={elementId} onChange={(e) => setElementId(e.target.value)}>
              <option value="">Projekt-niveau</option>
              {elements.map((el) => <option key={el.id} value={el.id}>{el.name}</option>)}
            </select>
          </div>
          <div className="field" style={{ flex: 1 }}>
            <label>Standardreference</label>
            <input value={reference} onChange={(e) => setReference(e.target.value)} placeholder="DS/EN 1991-1-3 + DK NA" />
          </div>
        </div>
        <div className="field">
          <label>Noter</label>
          <textarea rows={2} value={notes} onChange={(e) => setNotes(e.target.value)} />
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

function CalcModal({ projectId, elements, methods, onDone, onClose }: {
  projectId: string
  elements: StructuralElement[]
  methods: CalcMethod[]
  onDone: () => void
  onClose: () => void
}) {
  const [method, setMethod] = useState(methods[0]?.name ?? '')
  const [elementId, setElementId] = useState('')
  const [inputs, setInputs] = useState('')
  const [error, setError] = useState<string | null>(null)
  const selected = methods.find((m) => m.name === method)

  function useExample() {
    if (selected) setInputs(JSON.stringify(selected.inputsExample, null, 2))
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    let parsed: unknown
    try {
      parsed = JSON.parse(inputs)
    } catch {
      return setError('Inputs skal være gyldig JSON — brug "Indsæt eksempel" som skabelon.')
    }
    try {
      await api.post(`/projects/${projectId}/estimates`, {
        method,
        structuralElementId: elementId || undefined,
        inputs: parsed,
      })
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  return (
    <Modal title="Kør vejledende beregning" wide onClose={onClose}>
      <form onSubmit={submit}>
        <div className="form-row">
          <div className="field" style={{ flex: 1 }}>
            <label>Metode</label>
            <select value={method} onChange={(e) => setMethod(e.target.value)}>
              {methods.map((m) => <option key={m.name} value={m.name}>{m.name}</option>)}
            </select>
          </div>
          <div className="field" style={{ flex: 1 }}>
            <label>Element</label>
            <select value={elementId} onChange={(e) => setElementId(e.target.value)}>
              <option value="">(intet — projekt-niveau)</option>
              {elements.map((el) => <option key={el.id} value={el.id}>{el.name}</option>)}
            </select>
          </div>
        </div>
        {selected && (
          <p className="hint">{selected.description} · {selected.standardReference}</p>
        )}
        <div className="field">
          <label>Inputs (JSON) — <a href="#" onClick={(e) => (e.preventDefault(), useExample())}>Indsæt eksempel</a></label>
          <textarea rows={8} value={inputs} onChange={(e) => setInputs(e.target.value)} style={{ fontFamily: 'monospace' }} />
        </div>
        <ErrorText error={error} />
        <div className="form-row" style={{ justifyContent: 'flex-end', marginTop: 12 }}>
          <button type="button" className="btn secondary" onClick={onClose}>Annullér</button>
          <button className="btn">Beregn</button>
        </div>
      </form>
    </Modal>
  )
}

function EstimateModal({ estimate, onClose }: { estimate: CalculationEstimate; onClose: () => void }) {
  return (
    <Modal title={`${estimate.method} — resultat`} wide onClose={onClose}>
      <div className="draft-banner">{estimate.results.notice}</div>
      <div className="form-row">
        <StatusBadge status={estimate.status} />
        <span className="badge blue">{estimate.standardReference}</span>
      </div>
      <h3>Inputs</h3>
      <pre style={{ background: '#f6f4ef', padding: 10, borderRadius: 6, fontSize: 12.5, overflowX: 'auto' }}>
        {JSON.stringify(estimate.inputs, null, 2)}
      </pre>
      <h3>Antagelser — skal be- eller afkræftes af statiker</h3>
      <ul>
        {estimate.assumptions.map((a, i) => (
          <li key={i} style={{ marginBottom: 6 }}>
            {a.text} <span className="badge gray">{a.reference}</span>
          </li>
        ))}
      </ul>
      <h3>Resultater</h3>
      <pre style={{ background: '#f6f4ef', padding: 10, borderRadius: 6, fontSize: 12.5, overflowX: 'auto' }}>
        {JSON.stringify(estimate.results.results, null, 2)}
      </pre>
      <div className="form-row" style={{ justifyContent: 'flex-end' }}>
        <button className="btn" onClick={onClose}>Luk</button>
      </div>
    </Modal>
  )
}

function ReviewModal({ pkg, elements, loads, estimates, onDone, onClose }: {
  pkg: StructuralPackage
  elements: StructuralElement[]
  loads: StructuralLoad[]
  estimates: CalculationEstimate[]
  onDone: () => void
  onClose: () => void
}) {
  const [reviewerName, setReviewerName] = useState('')
  const [reviewerCompany, setReviewerCompany] = useState('')
  const [reviewerCredentials, setReviewerCredentials] = useState('')
  const [overallStatus, setOverallStatus] = useState('approved_with_changes')
  const [summary, setSummary] = useState('')
  const [items, setItems] = useState<{ targetKind: string; targetId: string; verdict: string; comment: string }[]>([])
  const [error, setError] = useState<string | null>(null)

  function addItem() {
    setItems((arr) => [...arr, { targetKind: 'load', targetId: '', verdict: 'approved', comment: '' }])
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    try {
      await api.post(`/structural-packages/${pkg.id}/reviews`, {
        reviewerName, reviewerCompany, reviewerCredentials, overallStatus, summary,
        items: items.filter((i) => i.targetId || i.verdict === 'comment').map((i) => ({
          structuralElementId: i.targetKind === 'element' && i.targetId ? i.targetId : undefined,
          loadId: i.targetKind === 'load' && i.targetId ? i.targetId : undefined,
          calculationEstimateId: i.targetKind === 'estimate' && i.targetId ? i.targetId : undefined,
          verdict: i.verdict,
          comment: i.comment,
        })),
      })
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  const targetOptions = (kind: string) =>
    kind === 'element' ? elements.map((e) => ({ id: e.id, label: e.name }))
    : kind === 'load' ? loads.map((l) => ({ id: l.id, label: `${l.loadType} ${l.value} ${l.unit}` }))
    : estimates.map((e) => ({ id: e.id, label: e.method }))

  return (
    <Modal title={`Registrér statikerens svar — pakke v${pkg.versionNo}`} wide onClose={onClose}>
      <p className="hint">
        Offline-loop: statikeren har fået PDF-pakken og svaret pr. mail/telefon. Registrér svaret her, så
        systemet kan spore hvilke antagelser der er bekræftet og hvilke der er ændret.
      </p>
      <form onSubmit={submit}>
        <div className="form-row">
          <div className="field">
            <label>Statikerens navn *</label>
            <input value={reviewerName} onChange={(e) => setReviewerName(e.target.value)} required />
          </div>
          <div className="field">
            <label>Firma</label>
            <input value={reviewerCompany} onChange={(e) => setReviewerCompany(e.target.value)} />
          </div>
          <div className="field">
            <label>Autorisation</label>
            <input value={reviewerCredentials} onChange={(e) => setReviewerCredentials(e.target.value)} placeholder="Anerkendt statiker" />
          </div>
          <div className="field">
            <label>Samlet vurdering</label>
            <select value={overallStatus} onChange={(e) => setOverallStatus(e.target.value)}>
              <option value="approved">Godkendt</option>
              <option value="approved_with_changes">Godkendt med ændringer</option>
              <option value="rejected">Afvist</option>
              <option value="partial">Delvis</option>
            </select>
          </div>
        </div>
        <div className="field" style={{ marginBottom: 10 }}>
          <label>Sammenfatning</label>
          <textarea rows={2} value={summary} onChange={(e) => setSummary(e.target.value)} />
        </div>

        <h3>Punkt-for-punkt</h3>
        {items.map((item, i) => (
          <div key={i} className="form-row" style={{ alignItems: 'center' }}>
            <select value={item.targetKind} onChange={(e) => setItems((arr) => arr.map((x, j) => j === i ? { ...x, targetKind: e.target.value, targetId: '' } : x))}>
              <option value="load">Last</option>
              <option value="estimate">Beregning</option>
              <option value="element">Element</option>
            </select>
            <select value={item.targetId} onChange={(e) => setItems((arr) => arr.map((x, j) => j === i ? { ...x, targetId: e.target.value } : x))}>
              <option value="">Vælg…</option>
              {targetOptions(item.targetKind).map((t) => <option key={t.id} value={t.id}>{t.label}</option>)}
            </select>
            <select value={item.verdict} onChange={(e) => setItems((arr) => arr.map((x, j) => j === i ? { ...x, verdict: e.target.value } : x))}>
              <option value="approved">Godkendt</option>
              <option value="changed">Ændret</option>
              <option value="rejected">Afvist</option>
              <option value="comment">Kommentar</option>
            </select>
            <input style={{ flex: 1 }} placeholder="Kommentar/rettelse" value={item.comment}
              onChange={(e) => setItems((arr) => arr.map((x, j) => j === i ? { ...x, comment: e.target.value } : x))} />
          </div>
        ))}
        <button type="button" className="btn small secondary" onClick={addItem}>+ Punkt</button>

        <ErrorText error={error} />
        <div className="form-row" style={{ justifyContent: 'flex-end', marginTop: 12 }}>
          <button type="button" className="btn secondary" onClick={onClose}>Annullér</button>
          <button className="btn">Registrér svar</button>
        </div>
      </form>
    </Modal>
  )
}

function ReviewsListModal({ pkg, onClose }: { pkg: StructuralPackage; onClose: () => void }) {
  const { data: reviews } = useLoad(
    () => api.get<EngineerReview[]>(`/structural-packages/${pkg.id}/reviews`), [pkg.id])

  return (
    <Modal title={`Statiker-svar — pakke v${pkg.versionNo}`} wide onClose={onClose}>
      {reviews?.length === 0 && <Empty>Ingen svar registreret endnu.</Empty>}
      {reviews?.map((r) => (
        <div key={r.id} className="card">
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <strong>{r.reviewerName}{r.reviewerCompany && <>, {r.reviewerCompany}</>}</strong>
            <StatusBadge status={r.overallStatus} />
          </div>
          <div className="hint">{r.reviewerCredentials} · modtaget {formatDate(r.receivedAt)}</div>
          {r.summary && <p>{r.summary}</p>}
          {r.items && r.items.length > 0 && (
            <table className="tbl">
              <tbody>
                {r.items.map((item) => (
                  <tr key={item.id}>
                    <td><StatusBadge status={item.verdict === 'approved' ? 'ok' : item.verdict === 'changed' ? 'attention' : item.verdict === 'rejected' ? 'rejected' : 'not_checked'} /></td>
                    <td>{item.comment || '–'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      ))}
      <div className="form-row" style={{ justifyContent: 'flex-end' }}>
        <button className="btn" onClick={onClose}>Luk</button>
      </div>
    </Modal>
  )
}
