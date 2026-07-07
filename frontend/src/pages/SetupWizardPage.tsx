import { useState } from 'react'
import { useNavigate, useOutletContext, useParams } from 'react-router-dom'
import { api, kr, parseKr } from '../api/client'
import type { ApplyResult, Blueprint, BlueprintResult, Interview } from '../api/types'
import { Empty, ErrorText } from '../components'
import type { ProjectContext } from './ProjectShell'

// AI-projektstart: et "grill mig"-interview → blueprint-udkast til
// gennemsyn → oprettelse. Intet oprettes før brugeren godkender.

const propertyTypes = ['Sommerhus', 'Villa/parcelhus', 'Rækkehus', 'Lejlighed', 'Andet']
const featureOptions = [
  'Terrasse', 'Carport/garage', 'Nyt køkken', 'Nyt badeværelse', 'Brændeovn',
  'Varmepumpe', 'Solceller', 'Udestue', 'Anneks/skur',
]

export default function SetupWizardPage() {
  const { projectId } = useParams()
  const { project } = useOutletContext<ProjectContext>()
  const navigate = useNavigate()

  const [step, setStep] = useState(0)
  const [goal, setGoal] = useState(project.description ?? '')
  const [propertyType, setPropertyType] = useState('Sommerhus')
  const [sizeM2, setSizeM2] = useState('')
  const [roomsText, setRoomsText] = useState('')
  const [features, setFeatures] = useState<string[]>([])
  const [budget, setBudget] = useState('')
  const [selfBuild, setSelfBuild] = useState('mixed')
  const [timeline, setTimeline] = useState('')
  const [notes, setNotes] = useState('')

  const [result, setResult] = useState<BlueprintResult | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [applied, setApplied] = useState<ApplyResult | null>(null)

  function toggleFeature(f: string) {
    setFeatures((arr) => (arr.includes(f) ? arr.filter((x) => x !== f) : [...arr, f]))
  }

  async function generate() {
    setBusy(true)
    setError(null)
    const interview: Interview = {
      goal,
      propertyType,
      sizeM2: sizeM2 ? Number(sizeM2) : 0,
      rooms: roomsText.split(/[,\n]/).map((r) => r.trim()).filter(Boolean),
      features,
      budgetOre: parseKr(budget) ?? 0,
      selfBuild,
      timeline,
      notes,
    }
    try {
      const r = await api.post<BlueprintResult>(`/projects/${projectId}/setup/generate`, interview)
      setResult(r)
      setStep(4)
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  async function apply() {
    if (!result) return
    setBusy(true)
    setError(null)
    try {
      const summary = await api.post<ApplyResult>(`/projects/${projectId}/setup/apply`, result.blueprint)
      setApplied(summary)
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  if (applied) {
    return (
      <>
        <h1>Projektet er sat op! 🎉</h1>
        <div className="card">
          <ul>
            <li>{applied.tasksCreated} opgaver oprettet ({applied.dependenciesLinked} afhængigheder)</li>
            <li>{applied.roomsCreated} rum/zoner</li>
            <li>{applied.budgetItemsCreated} budgetposter</li>
            <li>{applied.materialsCreated} materialer</li>
            {applied.caseFileId && <li>1 byggesag oprettet som kladde</li>}
          </ul>
          <p className="hint">Alt kan redigeres og slettes som normalt — planen er et udgangspunkt, ikke en facitliste.</p>
          <button className="btn" onClick={() => navigate(`/projects/${projectId}`)}>Gå til overblikket</button>
        </div>
      </>
    )
  }

  const questions = [
    <div key="q0">
      <h2>Fortæl om projektet</h2>
      <div className="field" style={{ marginBottom: 12 }}>
        <label>Hvad skal der ske? Skriv løs — jo mere, jo bedre. *</label>
        <textarea rows={6} value={goal} onChange={(e) => setGoal(e.target.value)} autoFocus
          placeholder="Fx: Vi vil bygge et sommerhus på ca. 70 m² med saddeltag... eller: Vi renoverer badeværelset og fjerner måske væggen ind til gangen…" />
      </div>
      <div className="form-row">
        <div className="field">
          <label>Hvad er det for en type bolig?</label>
          <select value={propertyType} onChange={(e) => setPropertyType(e.target.value)}>
            {propertyTypes.map((t) => <option key={t}>{t}</option>)}
          </select>
        </div>
        <div className="field">
          <label>Hvor mange m² handler det om (ca.)?</label>
          <input type="number" min="0" value={sizeM2} onChange={(e) => setSizeM2(e.target.value)} placeholder="70" />
        </div>
      </div>
    </div>,
    <div key="q1">
      <h2>Rum og ønsker</h2>
      <div className="field" style={{ marginBottom: 12 }}>
        <label>Hvilke rum indgår? (ét pr. linje eller kommasepareret)</label>
        <textarea rows={4} value={roomsText} onChange={(e) => setRoomsText(e.target.value)} autoFocus
          placeholder={'Stue/køkken\nSoveværelse\nBadeværelse\nTerrasse'} />
      </div>
      <label style={{ fontSize: 12.5, color: 'var(--text-dim)' }}>Skal noget af dette med?</label>
      <div className="wizard-kinds" style={{ gridTemplateColumns: 'repeat(3, 1fr)' }}>
        {featureOptions.map((f) => (
          <div key={f} className={`wizard-kind ${features.includes(f) ? 'active' : ''}`} onClick={() => toggleFeature(f)}>
            <div className="name" style={{ fontSize: 13 }}>{f}</div>
          </div>
        ))}
      </div>
    </div>,
    <div key="q2">
      <h2>Penge og hænder</h2>
      <div className="form-row">
        <div className="field">
          <label>Hvad er budgettet (ca.)?</label>
          <input value={budget} onChange={(e) => setBudget(e.target.value)} placeholder="850.000" autoFocus />
        </div>
        <div className="field" style={{ flex: 1 }}>
          <label>Hvem skal bygge?</label>
          <select value={selfBuild} onChange={(e) => setSelfBuild(e.target.value)}>
            <option value="self">Vi laver det meste selv</option>
            <option value="mixed">Vi gør noget selv og hyrer håndværkere til resten</option>
            <option value="contractors">Håndværkere laver det hele</option>
          </select>
        </div>
      </div>
      <p className="hint">
        Budgettet bruges kun til at fordele dine egne tal på poster — Hefai opfinder aldrig priser.
      </p>
    </div>,
    <div key="q3">
      <h2>Tid og andet</h2>
      <div className="field" style={{ marginBottom: 12 }}>
        <label>Hvornår vil I gerne være færdige?</label>
        <input value={timeline} onChange={(e) => setTimeline(e.target.value)} placeholder="Fx inden sommeren 2027" autoFocus />
      </div>
      <div className="field">
        <label>Andet planen skal tage højde for?</label>
        <textarea rows={3} value={notes} onChange={(e) => setNotes(e.target.value)}
          placeholder="Fx: grunden skråner, vi kan kun arbejde i weekender, naboen er nærig med skellet…" />
      </div>
    </div>,
  ]

  return (
    <>
      <h1>AI-projektstart</h1>
      <p className="page-sub">
        Svar på spørgsmålene, så bygger Hefai et udkast til hele projektplanen — opgaver i rigtig
        rækkefølge, rum, budgetposter og materialeliste. Du godkender før noget oprettes.
      </p>

      {step < 4 && (
        <div className="card">
          <p className="hint">Spørgsmål {step + 1} af 4</p>
          {questions[step]}
          <ErrorText error={error} />
          <div className="form-row" style={{ justifyContent: 'space-between', marginTop: 14 }}>
            <div>
              {step > 0 && <button className="btn secondary" onClick={() => setStep(step - 1)}>Tilbage</button>}
            </div>
            {step < 3 ? (
              <button className="btn" disabled={step === 0 && !goal.trim()} onClick={() => setStep(step + 1)}>Næste</button>
            ) : (
              <button className="btn" disabled={busy} onClick={generate}>
                {busy ? 'Bygger planen…' : 'Byg mit projektudkast'}
              </button>
            )}
          </div>
        </div>
      )}

      {step === 4 && result && (
        <BlueprintReview
          result={result}
          busy={busy}
          error={error}
          onBack={() => setStep(3)}
          onApply={apply}
        />
      )}
    </>
  )
}

function BlueprintReview({ result, busy, error, onBack, onApply }: {
  result: BlueprintResult
  busy: boolean
  error: string | null
  onBack: () => void
  onApply: () => void
}) {
  const raw = result.blueprint
  // Defensivt: garantér arrays selv hvis serveren skulle sende null.
  const b: Blueprint = {
    ...raw,
    rooms: raw.rooms ?? [],
    tasks: (raw.tasks ?? []).map((t) => ({ ...t, dependsOn: t.dependsOn ?? [] })),
    budgetItems: raw.budgetItems ?? [],
    materials: raw.materials ?? [],
  }
  return (
    <>
      {result.notice && <div className="notice">{result.notice}</div>}
      <div className="card">
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <h2 style={{ margin: 0 }}>Udkast til projektplan</h2>
          <span className={`badge ${result.source === 'llm' ? 'blue' : 'gray'}`}>
            {result.source === 'llm' ? `Genereret af ${result.provider}` : 'Hefais standardskabelon'}
          </span>
        </div>
        {b.projectDescription && <p>{b.projectDescription}</p>}
        {b.notes && <div className="notice">{b.notes}</div>}
      </div>

      <div className="grid cols-2">
        <div className="card">
          <h3>Opgaver ({b.tasks.length})</h3>
          <ol style={{ paddingLeft: 20, margin: 0 }}>
            {b.tasks.map((t, i) => (
              <li key={i} style={{ marginBottom: 6 }}>
                <strong>{t.title}</strong>
                {t.phase && <span className="badge gray" style={{ marginLeft: 6 }}>{t.phase}</span>}
                {t.dependsOn.length > 0 && (
                  <div className="hint" style={{ margin: 0 }}>
                    venter på: {t.dependsOn.map((d) => b.tasks[d]?.title).join(', ')}
                  </div>
                )}
              </li>
            ))}
          </ol>
        </div>
        <div>
          <div className="card">
            <h3>Rum ({b.rooms.length})</h3>
            {b.rooms.length === 0 && <Empty>Ingen.</Empty>}
            {b.rooms.map((r, i) => (
              <span key={i} className="tag-chip">{r.name}</span>
            ))}
          </div>
          <div className="card">
            <h3>Budgetposter ({b.budgetItems.length})</h3>
            <table className="tbl">
              <tbody>
                {b.budgetItems.map((item, i) => (
                  <tr key={i}>
                    <td>{item.description}</td>
                    <td className="num">{kr(item.estimatedAmountOre)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {b.materials.length > 0 && (
            <div className="card">
              <h3>Materialer ({b.materials.length})</h3>
              {b.materials.map((m, i) => (
                <div key={i} style={{ fontSize: 13 }}>{m.quantity} {m.unit} {m.name} {m.spec}</div>
              ))}
              <p className="hint">Uden priser — de udfyldes når du har tilbud i hånden.</p>
            </div>
          )}
          {b.needsBuildingCase && (
            <div className="notice">Planen opretter også en byggesag som kladde — projektet ser ud til at kræve kontakt med kommunen.</div>
          )}
        </div>
      </div>

      <ErrorText error={error} />
      <div className="form-row" style={{ justifyContent: 'space-between' }}>
        <button className="btn secondary" onClick={onBack}>Tilbage til spørgsmålene</button>
        <button className="btn" disabled={busy} onClick={onApply}>
          {busy ? 'Opretter…' : 'Godkend og opret det hele'}
        </button>
      </div>
    </>
  )
}
