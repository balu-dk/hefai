import { NavLink, Outlet, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Project } from '../api/types'
import { useLoad } from '../components'
import { Tour, useStoredState, type TourStep } from '../onboarding'
import { useAuth } from '../auth'

export interface ProjectContext {
  project: Project
  reloadProject: () => void
  simpleMode: boolean
}

const tourSteps: TourStep[] = [
  {
    selector: '[data-tour="overview"]',
    title: 'Overblik',
    text: 'Her ser du hele projektet på ét blik: hvor langt du er, hvad der er brugt af budgettet, og hvad du kan gå i gang med lige nu.',
  },
  {
    selector: '[data-tour="tasks"]',
    title: 'Opgaver',
    text: 'Alle opgaver i byggeriet. Hefai holder styr på rækkefølgen — "Klar til start" viser hvad du kan gøre nu, og hvad der venter på andre.',
  },
  {
    selector: '[data-tour="budget"]',
    title: 'Budget',
    text: 'Skriv dit budget ind og registrér udgifter løbende. Så ved du altid præcis hvad der er brugt, og hvad der er tilbage.',
  },
  {
    selector: '[data-tour="materials"]',
    title: 'Indkøb & materialer',
    text: 'Lav en liste over alt der skal købes. Indkøbslisten samler det pr. byggemarked, så du aldrig kører forgæves.',
  },
  {
    selector: '[data-tour="documents"]',
    title: 'Dokumenter & billeder',
    text: 'Gem kvitteringer, garantier, tegninger og billeder her. Alt kan søges frem igen — fx "hvor er garantien på varmepumpen?"',
  },
  {
    selector: '[data-tour="drawings"]',
    title: 'Tegninger & 3D',
    text: 'Tegn din grundplan med vægge, rum, døre og vinduer — og se den som 3D-model med tag, grund og træer.',
  },
  {
    selector: '[data-tour="mode"]',
    title: 'Enkel eller avanceret?',
    text: 'Enkel visning holder menuen kort. Skift til Avanceret når du skal bruge byggesag, AI-assistent eller statiker-forberedelse.',
  },
]

export default function ProjectShell() {
  const { projectId } = useParams()
  const { user, logout } = useAuth()
  const [mode, setMode] = useStoredState<'simple' | 'advanced'>('hefai.mode', 'simple')
  const [tourDone, setTourDone] = useStoredState<'yes' | 'no'>('hefai.tourDone', 'no')
  const { data: project, error, reload } = useLoad(
    () => api.get<Project>(`/projects/${projectId}`),
    [projectId],
  )

  if (error) return <div style={{ padding: 40 }} className="error-text">{error}</div>
  if (!project) return <div style={{ padding: 40 }}>Indlæser…</div>

  const nav = ({ isActive }: { isActive: boolean }) => (isActive ? 'active' : '')
  const simple = mode === 'simple'

  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="brand">
          Hef<span>ai</span>
        </div>
        <div className="project-name" title={project.name}>
          {project.name}
        </div>
        <nav>
          <NavLink to="" end className={nav} data-tour="overview">Overblik</NavLink>
          <div className="section">Projekt & proces</div>
          {!simple && <NavLink to="phases" className={nav}>Faser</NavLink>}
          <NavLink to="tasks" className={nav} data-tour="tasks">Opgaver</NavLink>
          <NavLink to="budget" className={nav} data-tour="budget">Budget & økonomi</NavLink>
          <NavLink to="materials" className={nav} data-tour="materials">
            {simple ? 'Indkøb & materialer' : 'Materialer'}
          </NavLink>
          {!simple && <NavLink to="suppliers" className={nav}>Leverandører</NavLink>}
          {!simple && <NavLink to="rooms" className={nav}>Rum & zoner</NavLink>}
          <NavLink to="documents" className={nav} data-tour="documents">
            {simple ? 'Dokumenter & billeder' : 'Dokumenter'}
          </NavLink>
          <div className="section">{simple ? 'Tegning' : 'Byggesag'}</div>
          <NavLink to="drawings" className={nav} data-tour="drawings">Tegninger & 3D</NavLink>
          <NavLink to="rules" className={nav}>Lovtjek</NavLink>
          {!simple && (
            <>
              <NavLink to="cases" className={nav}>Byggesager</NavLink>
              <NavLink to="sources" className={nav}>Kildemateriale</NavLink>
              <NavLink to="assistant" className={nav}>AI-assistent</NavLink>
              <div className="section">Statik</div>
              <NavLink to="structural" className={nav}>Statiker-forberedelse</NavLink>
            </>
          )}
        </nav>
        <div className="mode-toggle" data-tour="mode">
          Visning:
          <button className={simple ? 'on' : ''} onClick={() => setMode('simple')}>Enkel</button>
          <button className={!simple ? 'on' : ''} onClick={() => setMode('advanced')}>Avanceret</button>
        </div>
        <div className="foot">
          <NavLink to="/">← Alle projekter</NavLink>
          <div style={{ marginTop: 6 }}>
            {user?.displayName} · <button onClick={logout}>Log ud</button>
          </div>
          <div style={{ marginTop: 6 }}>
            <button onClick={() => setTourDone('no')}>Vis rundvisning igen</button>
          </div>
        </div>
      </aside>
      <main className="main">
        <Outlet context={{ project, reloadProject: reload, simpleMode: simple } satisfies ProjectContext} />
      </main>
      {tourDone === 'no' && <Tour steps={tourSteps} onDone={() => setTourDone('yes')} />}
    </div>
  )
}
