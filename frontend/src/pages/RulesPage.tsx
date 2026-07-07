import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import { Empty, ErrorText, Modal, useLoad } from '../components'

// Lovtjek: strukturerede grænseværdier med kildehenvisning, evalueret
// automatisk mod den nyeste tegning. Hjertet i "systemet ved selv om du
// bryder reglerne".

interface RuleParam {
  id: string
  label: string
  unit: string
  direction: 'max' | 'min'
}

interface Rule {
  id: string
  parameter: string
  value: number
  sourceChunkId: string | null
  sourceRef: string
  quote: string
  status: 'suggested' | 'confirmed' | 'rejected'
  note: string
}

interface Evaluation {
  rule: Rule
  parameter: RuleParam
  factValue: number | null
  status: 'ok' | 'violation' | 'unknown'
  margin: number | null
}

interface EvaluationResult {
  facts: {
    footprintM2: number | null
    roomAreaM2: number | null
    plotAreaM2: number | null
    bebyggelsesprocentPct: number | null
    minSkelafstandM: number | null
    bygningshoejdeM: number | null
    taghaeldningDeg: number | null
    assumptions: string[]
  }
  drawingTitle?: string
  evaluations: Evaluation[]
  violations: number
  notice: string
}

export default function RulesPage() {
  const { projectId } = useParams()
  const { data: rules, reload: reloadRules } = useLoad(
    () => api.get<Rule[]>(`/projects/${projectId}/rules`), [projectId])
  const { data: evaluation, reload: reloadEval } = useLoad(
    () => api.get<EvaluationResult>(`/projects/${projectId}/rules/evaluation`), [projectId])
  const { data: catalog } = useLoad(() => api.get<RuleParam[]>(`/rules/catalog`), [])
  const [adding, setAdding] = useState(false)
  const [extracting, setExtracting] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const reloadAll = () => (reloadRules(), reloadEval())

  async function extract() {
    setExtracting(true)
    setError(null)
    setNotice(null)
    try {
      const result = await api.post<{ suggested: Rule[]; notice?: string }>(
        `/projects/${projectId}/rules/extract`, {})
      setNotice(result.notice ?? `${result.suggested.length} regel-forslag fundet i kildematerialet — bekræft dem mod kilden nedenfor.`)
      reloadAll()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setExtracting(false)
    }
  }

  async function setStatus(rule: Rule, status: string) {
    await api.patch(`/rules/${rule.id}`, { status })
    reloadAll()
  }

  async function removeRule(rule: Rule) {
    await api.del(`/rules/${rule.id}`)
    reloadAll()
  }

  const paramLabel = (id: string) => catalog?.find((p) => p.id === id)?.label ?? id
  const paramUnit = (id: string) => catalog?.find((p) => p.id === id)?.unit ?? ''

  const fmt = (v: number | null | undefined, digits = 2) =>
    v == null ? '–' : v.toLocaleString('da-DK', { maximumFractionDigits: digits })

  return (
    <>
      <h1>Lovtjek</h1>
      <p className="page-sub">
        Automatisk egenkontrol: grænseværdier fra dit kildemateriale sammenlignes løbende med den
        nyeste tegning. Vejledende hjælp — ikke en myndighedsafgørelse.
      </p>

      {evaluation && (
        <div className="card" style={{ borderLeft: `3px solid ${evaluation.violations > 0 ? 'var(--red)' : 'var(--green)'}` }}>
          <h3 style={{ marginTop: 0 }}>
            {evaluation.violations > 0
              ? `⚠️ ${evaluation.violations} regel(er) overskredet`
              : evaluation.evaluations.length > 0
                ? '✅ Alle regler overholdes'
                : 'Ingen regler at tjekke endnu'}
            {evaluation.drawingTitle && (
              <span style={{ fontWeight: 400, fontSize: 13, color: 'var(--text-dim)' }}>
                {' '}· målt på "{evaluation.drawingTitle}"
              </span>
            )}
          </h3>
          <table className="tbl">
            <thead>
              <tr><th>Krav</th><th className="num">Grænse</th><th className="num">Målt på tegningen</th><th className="num">Margin</th><th>Kilde</th><th>Status</th></tr>
            </thead>
            <tbody>
              {evaluation.evaluations.map((e) => (
                <tr key={e.rule.id}>
                  <td>
                    {paramLabel(e.rule.parameter)}
                    {e.rule.status === 'suggested' && <span className="badge amber" style={{ marginLeft: 6 }}>afventer bekræftelse</span>}
                  </td>
                  <td className="num">{fmt(e.rule.value)} {paramUnit(e.rule.parameter)}</td>
                  <td className="num">{fmt(e.factValue)} {e.factValue != null ? paramUnit(e.rule.parameter) : ''}</td>
                  <td className="num" style={{ color: e.margin != null && e.margin < 0 ? 'var(--red)' : 'var(--green)' }}>
                    {e.margin != null ? `${e.margin >= 0 ? '+' : ''}${fmt(e.margin)}` : '–'}
                  </td>
                  <td>{e.rule.sourceRef ? <span className="badge blue">{e.rule.sourceRef}</span> : <span className="badge red">uden kilde</span>}</td>
                  <td>
                    {e.status === 'ok' && <span className="badge green">Overholdt</span>}
                    {e.status === 'violation' && <span className="badge red">OVERSKREDET</span>}
                    {e.status === 'unknown' && <span className="badge gray" title="Tegningen mangler data til at måle dette">Kan ikke måles</span>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {evaluation.evaluations.length === 0 && (
            <Empty>Tilføj regler nedenfor — eller lad AI'en foreslå dem ud fra dit kildemateriale.</Empty>
          )}

          <h3>Sådan er tallene målt</h3>
          <table className="tbl">
            <tbody>
              <tr><td>Bebygget areal (fodaftryk)</td><td className="num">{fmt(evaluation.facts.footprintM2)} m²</td></tr>
              <tr><td>Indvendigt areal</td><td className="num">{fmt(evaluation.facts.roomAreaM2)} m²</td></tr>
              <tr><td>Grundareal</td><td className="num">{fmt(evaluation.facts.plotAreaM2)} m²</td></tr>
              <tr><td>Bebyggelsesprocent</td><td className="num">{fmt(evaluation.facts.bebyggelsesprocentPct)} %</td></tr>
              <tr><td>Min. afstand til skel</td><td className="num">{fmt(evaluation.facts.minSkelafstandM)} m</td></tr>
              <tr><td>Bygningshøjde</td><td className="num">{fmt(evaluation.facts.bygningshoejdeM)} m</td></tr>
            </tbody>
          </table>
          {evaluation.facts.assumptions.map((a, i) => (
            <p key={i} className="hint" style={{ margin: '4px 0' }}>• {a}</p>
          ))}
          <p className="hint">{evaluation.notice}</p>
        </div>
      )}

      <h2>Regler</h2>
      {notice && <div className="notice">{notice}</div>}
      <ErrorText error={error} />
      <div className="card">
        <table className="tbl">
          <thead>
            <tr><th>Krav</th><th className="num">Grænse</th><th>Kilde & citat</th><th>Status</th><th></th></tr>
          </thead>
          <tbody>
            {rules?.map((rule) => (
              <tr key={rule.id}>
                <td>{paramLabel(rule.parameter)}</td>
                <td className="num">{fmt(rule.value)} {paramUnit(rule.parameter)}</td>
                <td style={{ maxWidth: 380 }}>
                  {rule.sourceRef && <span className="badge blue">{rule.sourceRef}</span>}
                  {rule.quote && <div className="hint" style={{ margin: '4px 0 0' }}>"{rule.quote}"</div>}
                  {rule.note && <div className="hint" style={{ margin: '2px 0 0' }}>{rule.note}</div>}
                </td>
                <td>
                  {rule.status === 'confirmed' && <span className="badge green">Bekræftet</span>}
                  {rule.status === 'suggested' && <span className="badge amber">Forslag</span>}
                  {rule.status === 'rejected' && <span className="badge gray">Afvist</span>}
                </td>
                <td className="row-actions">
                  {rule.status !== 'confirmed' && (
                    <button className="btn small" onClick={() => setStatus(rule, 'confirmed')}>Bekræft</button>
                  )}
                  {rule.status !== 'rejected' && (
                    <button className="btn small secondary" onClick={() => setStatus(rule, 'rejected')}>Afvis</button>
                  )}
                  <button className="btn small secondary" onClick={() => removeRule(rule)}>Slet</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {rules?.length === 0 && <Empty>Ingen regler endnu.</Empty>}
      </div>

      <div className="row-actions">
        <button className="btn" disabled={extracting} onClick={extract}>
          {extracting ? 'Læser kildematerialet…' : '✨ Foreslå regler fra kildematerialet'}
        </button>
        <button className="btn secondary" onClick={() => setAdding(true)}>+ Tilføj regel manuelt</button>
      </div>
      <p className="hint">
        AI'en foreslår kun regler med ordret citat fra dit <Link to={`/projects/${projectId}/sources`}>kildemateriale</Link> —
        du bekræfter hver enkelt mod kilden, før den tæller som aktiv.
      </p>

      {adding && catalog && (
        <AddRuleModal projectId={projectId!} catalog={catalog}
          onDone={() => (setAdding(false), reloadAll())} onClose={() => setAdding(false)} />
      )}
    </>
  )
}

function AddRuleModal({ projectId, catalog, onDone, onClose }: {
  projectId: string
  catalog: RuleParam[]
  onDone: () => void
  onClose: () => void
}) {
  const [parameter, setParameter] = useState(catalog[0]?.id ?? '')
  const [value, setValue] = useState('')
  const [note, setNote] = useState('')
  const [error, setError] = useState<string | null>(null)
  const selected = catalog.find((p) => p.id === parameter)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    try {
      await api.post(`/projects/${projectId}/rules`, {
        parameter,
        value: Number(value.replace(',', '.')),
        note,
      })
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  return (
    <Modal title="Tilføj regel" onClose={onClose}>
      <form onSubmit={submit}>
        <div className="form-row">
          <div className="field" style={{ flex: 1 }}>
            <label>Krav</label>
            <select value={parameter} onChange={(e) => setParameter(e.target.value)}>
              {catalog.map((p) => <option key={p.id} value={p.id}>{p.label}</option>)}
            </select>
          </div>
          <div className="field">
            <label>Grænseværdi ({selected?.unit})</label>
            <input value={value} onChange={(e) => setValue(e.target.value)} required placeholder="5,0" autoFocus />
          </div>
        </div>
        <div className="field">
          <label>Note (fx hvor værdien kommer fra)</label>
          <input value={note} onChange={(e) => setNote(e.target.value)} placeholder="Bekræftet telefonisk med kommunen 12/8" />
        </div>
        <p className="hint">Manuelt tilføjede regler markeres som bekræftede — du står selv inde for værdien.</p>
        <ErrorText error={error} />
        <div className="form-row" style={{ justifyContent: 'flex-end', marginTop: 12 }}>
          <button type="button" className="btn secondary" onClick={onClose}>Annullér</button>
          <button className="btn">Gem regel</button>
        </div>
      </form>
    </Modal>
  )
}
