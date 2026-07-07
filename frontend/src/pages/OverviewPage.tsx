import { Link, useOutletContext, useParams } from 'react-router-dom'
import { api, kr, formatDate } from '../api/client'
import type { BoardTask, BudgetSummary, CaseFile, Doc, Drawing, Phase } from '../api/types'
import { Empty, StatusBadge, statusLabel, useLoad } from '../components'
import { useStoredState } from '../onboarding'
import type { ProjectContext } from './ProjectShell'

export default function OverviewPage() {
  const { projectId } = useParams()
  const { project } = useOutletContext<ProjectContext>()
  const { data: phases } = useLoad(() => api.get<Phase[]>(`/projects/${projectId}/phases`), [projectId])
  const { data: board } = useLoad(() => api.get<BoardTask[]>(`/projects/${projectId}/tasks/board`), [projectId])
  const { data: budget } = useLoad(() => api.get<BudgetSummary>(`/projects/${projectId}/budget/summary`), [projectId])
  const { data: cases } = useLoad(() => api.get<CaseFile[]>(`/projects/${projectId}/case-files`), [projectId])
  const { data: drawings } = useLoad(() => api.get<Drawing[]>(`/projects/${projectId}/drawings`), [projectId])
  const { data: docs } = useLoad(() => api.get<Doc[]>(`/projects/${projectId}/documents`), [projectId])

  const actionable = board?.filter((t) => t.actionable) ?? []
  const inProgress = board?.filter((t) => t.status === 'in_progress') ?? []
  const donePhases = phases?.filter((p) => p.status === 'completed').length ?? 0

  return (
    <>
      <h1>{project.name}</h1>
      <GettingStarted
        projectId={projectId!}
        hasDrawing={(drawings?.length ?? 0) > 0}
        hasTask={(board?.length ?? 0) > 0}
        hasBudget={(budget?.estimatedOre ?? 0) > 0}
        hasDocument={(docs?.length ?? 0) > 0}
        loaded={!!(drawings && board && budget && docs)}
      />
      <p className="page-sub">
        {statusLabel(project.kind)}
        {project.address && <> · {project.address}</>}
        {project.municipality && <> · {project.municipality} Kommune</>}
        {project.plotAreaM2 != null && <> · grund {project.plotAreaM2} m²</>}
      </p>

      <div className="grid cols-3">
        <div className="card stat">
          <div className="label">Fremdrift, faser</div>
          <div className="value">
            {donePhases} / {phases?.length ?? 0}
          </div>
          <div className="progress-track" style={{ marginTop: 8 }}>
            <div
              className="progress-fill"
              style={{ width: phases?.length ? `${(donePhases / phases.length) * 100}%` : '0%' }}
            />
          </div>
        </div>
        <div className="card stat">
          <div className="label">Budget brugt / tilbage</div>
          <div className="value" style={{ fontSize: 17 }}>
            {budget ? (
              <>
                {kr(budget.spentOre)}
                <span style={{ color: 'var(--text-dim)' }}> / {kr(budget.remainingOre)}</span>
              </>
            ) : (
              '–'
            )}
          </div>
          {budget && budget.estimatedOre > 0 && (
            <div className="progress-track" style={{ marginTop: 8 }}>
              <div
                className={budget.spentOre > budget.estimatedOre ? 'progress-fill over' : 'progress-fill'}
                style={{ width: `${Math.min(100, (budget.spentOre / budget.estimatedOre) * 100)}%` }}
              />
            </div>
          )}
        </div>
        <div className="card stat">
          <div className="label">Byggesag</div>
          <div className="value" style={{ fontSize: 15 }}>
            {cases && cases.length > 0 ? <StatusBadge status={cases[0].status} /> : 'Ingen sag endnu'}
          </div>
          {cases && cases.length > 0 && (
            <div style={{ fontSize: 12.5, color: 'var(--text-dim)', marginTop: 6 }}>
              <Link to="cases">{cases[0].title}</Link>
            </div>
          )}
        </div>
      </div>

      <h2>Hvad kan jeg gøre nu?</h2>
      <div className="grid cols-2">
        <div className="card">
          <h3>Klar til start ({actionable.length})</h3>
          {actionable.slice(0, 6).map((t) => (
            <div key={t.id} className="task-card actionable">
              <div className="title">{t.title}</div>
              <div className="meta">{t.plannedEnd && <>frist {formatDate(t.plannedEnd)}</>}</div>
            </div>
          ))}
          {actionable.length === 0 && <Empty>Ingen opgaver venter på dig lige nu.</Empty>}
        </div>
        <div className="card">
          <h3>I gang ({inProgress.length})</h3>
          {inProgress.slice(0, 6).map((t) => (
            <div key={t.id} className="task-card">
              <div className="title">{t.title}</div>
            </div>
          ))}
          {inProgress.length === 0 && <Empty>Intet er i gang.</Empty>}
        </div>
      </div>

      <h2>Tidslinje</h2>
      <div className="card">
        <table className="tbl">
          <thead>
            <tr>
              <th>Fase</th>
              <th>Status</th>
              <th>Planlagt</th>
              <th>Faktisk</th>
            </tr>
          </thead>
          <tbody>
            {phases?.map((p) => (
              <tr key={p.id}>
                <td>{p.name}</td>
                <td>
                  <StatusBadge status={p.status} />
                </td>
                <td>
                  {formatDate(p.plannedStart)} – {formatDate(p.plannedEnd)}
                </td>
                <td>
                  {formatDate(p.actualStart)} – {formatDate(p.actualEnd)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  )
}

// "Kom godt i gang" — guider de første skridt og vinker farvel når alt er
// gjort (eller når man lukker den).
function GettingStarted({ projectId, hasDrawing, hasTask, hasBudget, hasDocument, loaded }: {
  projectId: string
  hasDrawing: boolean
  hasTask: boolean
  hasBudget: boolean
  hasDocument: boolean
  loaded: boolean
}) {
  const [dismissed, setDismissed] = useStoredState<'yes' | 'no'>(`hefai.gettingStarted.${projectId}`, 'no')

  const items = [
    {
      done: hasDrawing,
      to: 'drawings',
      title: 'Tegn jeres grundplan',
      desc: 'Vægge, rum, døre og vinduer — og se den i 3D bagefter.',
    },
    {
      done: hasTask,
      to: 'tasks',
      title: 'Opret den første opgave',
      desc: 'Fx "Ring til kommunen" eller "Indhent tilbud fra tømrer".',
    },
    {
      done: hasBudget,
      to: 'budget',
      title: 'Læg et budget',
      desc: 'Bare et groft overslag — Hefai holder styr på brugt/tilbage.',
    },
    {
      done: hasDocument,
      to: 'documents',
      title: 'Gem det første dokument',
      desc: 'Skøde, tilbud eller et billede af grunden.',
    },
  ]
  const allDone = items.every((i) => i.done)

  if (!loaded || dismissed === 'yes' || allDone) return null

  return (
    <div className="card getting-started" style={{ borderLeft: '3px solid var(--accent)' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
        <h3 style={{ margin: 0 }}>Kom godt i gang ({items.filter((i) => i.done).length}/{items.length})</h3>
        <button className="btn small secondary" onClick={() => setDismissed('yes')}>Skjul</button>
      </div>
      {!hasTask && (
        <p style={{ margin: '10px 0 0' }}>
          🪄 <Link to="setup"><strong>Lad AI-projektstarten interviewe dig</strong></Link> og bygge hele
          planen — opgaver, budget, rum og materialeliste — som et udkast du godkender.
        </p>
      )}
      <ul style={{ listStyle: 'none', padding: 0, margin: '10px 0 0' }}>
        {items.map((item) => (
          <li key={item.to} className={item.done ? 'done' : ''}>
            <span className="check">{item.done ? '✅' : '⬜'}</span>
            <Link to={item.to}><strong>{item.title}</strong></Link>
            {!item.done && <span style={{ color: 'var(--text-dim)' }}> — {item.desc}</span>}
          </li>
        ))}
      </ul>
    </div>
  )
}
