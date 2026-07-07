import { NavLink, Outlet, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Project } from '../api/types'
import { useLoad } from '../components'
import { useAuth } from '../auth'

export interface ProjectContext {
  project: Project
  reloadProject: () => void
}

export default function ProjectShell() {
  const { projectId } = useParams()
  const { user, logout } = useAuth()
  const { data: project, error, reload } = useLoad(
    () => api.get<Project>(`/projects/${projectId}`),
    [projectId],
  )

  if (error) return <div style={{ padding: 40 }} className="error-text">{error}</div>
  if (!project) return <div style={{ padding: 40 }}>Indlæser…</div>

  const nav = ({ isActive }: { isActive: boolean }) => (isActive ? 'active' : '')

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
          <NavLink to="" end className={nav}>Overblik</NavLink>
          <div className="section">Projekt & proces</div>
          <NavLink to="phases" className={nav}>Faser</NavLink>
          <NavLink to="tasks" className={nav}>Opgaver</NavLink>
          <NavLink to="budget" className={nav}>Budget & økonomi</NavLink>
          <NavLink to="materials" className={nav}>Materialer</NavLink>
          <NavLink to="suppliers" className={nav}>Leverandører</NavLink>
          <NavLink to="rooms" className={nav}>Rum & zoner</NavLink>
          <NavLink to="documents" className={nav}>Dokumenter</NavLink>
          <div className="section">Byggesag</div>
          <NavLink to="cases" className={nav}>Byggesager</NavLink>
          <NavLink to="drawings" className={nav}>Tegninger</NavLink>
          <NavLink to="sources" className={nav}>Kildemateriale</NavLink>
          <NavLink to="assistant" className={nav}>AI-assistent</NavLink>
          <div className="section">Statik</div>
          <NavLink to="structural" className={nav}>Statiker-forberedelse</NavLink>
        </nav>
        <div className="foot">
          <NavLink to="/">← Alle projekter</NavLink>
          <div style={{ marginTop: 6 }}>
            {user?.displayName} · <button onClick={logout}>Log ud</button>
          </div>
        </div>
      </aside>
      <main className="main">
        <Outlet context={{ project, reloadProject: reload } satisfies ProjectContext} />
      </main>
    </div>
  )
}
