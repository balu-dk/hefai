import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { AskResult, CaseFile } from '../api/types'
import { ErrorText, useLoad } from '../components'

interface ChatEntry {
  role: 'user' | 'assistant'
  text: string
  result?: AskResult
}

export default function AssistantPage() {
  const { projectId } = useParams()
  const { data: cases } = useLoad(() => api.get<CaseFile[]>(`/projects/${projectId}/case-files`), [projectId])
  const [caseFileId, setCaseFileId] = useState('')
  const [question, setQuestion] = useState('')
  const [log, setLog] = useState<ChatEntry[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function ask(e: React.FormEvent) {
    e.preventDefault()
    const q = question.trim()
    if (!q) return
    setLog((l) => [...l, { role: 'user', text: q }])
    setQuestion('')
    setBusy(true)
    setError(null)
    try {
      const result = await api.post<AskResult>(`/projects/${projectId}/assistant/ask`, {
        question: q,
        caseFileId: caseFileId || undefined,
      })
      const text = result.answered ? result.answer : (result.notice ?? 'Intet svar.')
      setLog((l) => [...l, { role: 'assistant', text, result }])
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <h1>AI-assistent</h1>
      <p className="page-sub">
        Svarer kun ud fra dit <Link to={`/projects/${projectId}/sources`}>kildemateriale</Link> — aldrig gæt.
        Alt der kræver bekræftelse fra kommune eller rådgiver markeres tydeligt.
      </p>

      <div className="form-row">
        <div className="field">
          <label>Byggesag som kontekst</label>
          <select value={caseFileId} onChange={(e) => setCaseFileId(e.target.value)}>
            <option value="">(ingen)</option>
            {cases?.map((c) => <option key={c.id} value={c.id}>{c.title}</option>)}
          </select>
        </div>
      </div>

      <div className="card">
        <div className="chat-log">
          {log.length === 0 && (
            <p className="hint">
              Stil et spørgsmål, fx: "Hvor tæt på skel må jeg bygge?" eller "Hvilke bilag kræver kommunen typisk
              til en byggetilladelse?"
            </p>
          )}
          {log.map((entry, i) => (
            <div key={i} className={`chat-msg ${entry.role}`}>
              {entry.text}
              {entry.result && entry.result.citations.length > 0 && (
                <div>
                  {entry.result.citations.map((c, j) => (
                    <div key={c.chunkId} className="cite">
                      <span className="ref">[{j + 1}] {c.sourceTitle}{c.sectionRef && <> — {c.sectionRef}</>}</span>
                      <div style={{ marginTop: 4 }}>{c.content.length > 400 ? c.content.slice(0, 400) + '…' : c.content}</div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))}
          {busy && <div className="chat-msg assistant">Søger i kilderne…</div>}
        </div>
        <ErrorText error={error} />
        <form onSubmit={ask} className="form-row" style={{ marginBottom: 0 }}>
          <div className="field" style={{ flex: 1 }}>
            <input value={question} onChange={(e) => setQuestion(e.target.value)} placeholder="Stil et spørgsmål…" disabled={busy} />
          </div>
          <button className="btn" disabled={busy || !question.trim()}>Spørg</button>
        </form>
      </div>

      <div className="notice">
        Assistenten træffer ingen myndighedsafgørelser og garanterer aldrig godkendelse. Den opfinder ikke
        paragraffer: findes svaret ikke i dine kilder, siger den det og henviser til kommunen eller en rådgiver.
      </div>
    </>
  )
}
