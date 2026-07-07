import { Navigate, Route, Routes } from 'react-router-dom'
import { useAuth } from './auth'
import LoginPage from './pages/LoginPage'
import ProjectsPage from './pages/ProjectsPage'
import ProjectShell from './pages/ProjectShell'
import OverviewPage from './pages/OverviewPage'
import PhasesPage from './pages/PhasesPage'
import TasksPage from './pages/TasksPage'
import BudgetPage from './pages/BudgetPage'
import MaterialsPage from './pages/MaterialsPage'
import SuppliersPage from './pages/SuppliersPage'
import RoomsPage from './pages/RoomsPage'
import DocumentsPage from './pages/DocumentsPage'
import CasesPage from './pages/CasesPage'
import CaseDetailPage from './pages/CaseDetailPage'
import DrawingsPage from './pages/DrawingsPage'
import DrawingEditorPage from './pages/DrawingEditorPage'
import SourcesPage from './pages/SourcesPage'
import AssistantPage from './pages/AssistantPage'
import StructuralPage from './pages/StructuralPage'
import SetupWizardPage from './pages/SetupWizardPage'

export default function App() {
  const { user, loading } = useAuth()

  if (loading) return <div style={{ padding: 40 }}>Indlæser…</div>
  if (!user) return <LoginPage />

  return (
    <Routes>
      <Route path="/" element={<ProjectsPage />} />
      <Route path="/projects/:projectId" element={<ProjectShell />}>
        <Route index element={<OverviewPage />} />
        <Route path="setup" element={<SetupWizardPage />} />
        <Route path="phases" element={<PhasesPage />} />
        <Route path="tasks" element={<TasksPage />} />
        <Route path="budget" element={<BudgetPage />} />
        <Route path="materials" element={<MaterialsPage />} />
        <Route path="suppliers" element={<SuppliersPage />} />
        <Route path="rooms" element={<RoomsPage />} />
        <Route path="documents" element={<DocumentsPage />} />
        <Route path="cases" element={<CasesPage />} />
        <Route path="cases/:caseFileId" element={<CaseDetailPage />} />
        <Route path="drawings" element={<DrawingsPage />} />
        <Route path="drawings/:drawingId" element={<DrawingEditorPage />} />
        <Route path="sources" element={<SourcesPage />} />
        <Route path="assistant" element={<AssistantPage />} />
        <Route path="structural" element={<StructuralPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
