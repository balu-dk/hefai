import { useCallback, useEffect, useState, type ReactNode } from 'react'

/** useLoad fetches on mount (and when deps change) with a manual reload. */
export function useLoad<T>(fn: () => Promise<T>, deps: unknown[]): {
  data: T | null
  error: string | null
  loading: boolean
  reload: () => void
} {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [tick, setTick] = useState(0)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    fn()
      .then((d) => !cancelled && (setData(d), setError(null)))
      .catch((e: Error) => !cancelled && setError(e.message))
      .finally(() => !cancelled && setLoading(false))
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, tick])

  const reload = useCallback(() => setTick((t) => t + 1), [])
  return { data, error, loading, reload }
}

export function Modal({ title, wide, onClose, children }: {
  title: string
  wide?: boolean
  onClose: () => void
  children: ReactNode
}) {
  return (
    <div className="modal-backdrop" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className={wide ? 'modal wide' : 'modal'}>
        <h2>{title}</h2>
        {children}
      </div>
    </div>
  )
}

export function Badge({ tone, children }: { tone: string; children: ReactNode }) {
  return <span className={`badge ${tone}`}>{children}</span>
}

const statusTones: Record<string, string> = {
  // tasks
  todo: 'gray', blocked: 'amber', in_progress: 'blue', done: 'green', cancelled: 'gray',
  // phases / projects
  not_started: 'gray', completed: 'green', planning: 'blue', on_hold: 'amber', archived: 'gray',
  // materials
  needed: 'amber', ordered: 'blue', delivered: 'green', in_stock: 'green', used: 'gray',
  // case
  draft: 'gray', ready_for_submission: 'blue', submitted: 'blue', awaiting_response: 'amber',
  questions_from_municipality: 'amber', approved: 'green', rejected: 'red', closed: 'gray',
  // compliance
  not_checked: 'gray', ok: 'green', attention: 'amber', needs_confirmation: 'red', confirmed: 'green',
  // structural
  assumed: 'amber', engineer_confirmed: 'green', engineer_changed: 'red',
  advisory: 'amber', verified: 'green', superseded: 'gray',
  sent: 'blue', reviewed: 'green',
  approved_with_changes: 'amber', partial: 'amber',
}

const statusLabels: Record<string, string> = {
  todo: 'Klar', blocked: 'Blokeret', in_progress: 'I gang', done: 'Færdig', cancelled: 'Annulleret',
  not_started: 'Ikke startet', completed: 'Afsluttet', planning: 'Planlægning', on_hold: 'På pause',
  archived: 'Arkiveret',
  needed: 'Skal bruges', ordered: 'Bestilt', delivered: 'Leveret', in_stock: 'På lager', used: 'Brugt',
  draft: 'Kladde', ready_for_submission: 'Klar til indsendelse', submitted: 'Indsendt',
  awaiting_response: 'Afventer svar', questions_from_municipality: 'Spørgsmål fra kommunen',
  approved: 'Godkendt', rejected: 'Afvist', closed: 'Lukket',
  not_checked: 'Ikke tjekket', ok: 'OK', attention: 'OBS', needs_confirmation: 'Kræver bekræftelse',
  confirmed: 'Bekræftet',
  assumed: 'Antaget', engineer_confirmed: 'Bekræftet af statiker', engineer_changed: 'Ændret af statiker',
  advisory: 'Vejledende', verified: 'Verificeret', superseded: 'Forældet',
  sent: 'Sendt', reviewed: 'Gennemgået',
  approved_with_changes: 'Godkendt med ændringer', partial: 'Delvis',
  unknown: 'Ikke afklaret', notification: 'Anmeldelse', building_permit: 'Byggetilladelse',
  new_build: 'Nybyggeri', renovation: 'Renovering', extension: 'Tilbygning', other: 'Andet',
}

export function StatusBadge({ status }: { status: string }) {
  return <Badge tone={statusTones[status] ?? 'gray'}>{statusLabels[status] ?? status}</Badge>
}

export function statusLabel(status: string): string {
  return statusLabels[status] ?? status
}

export function ErrorText({ error }: { error: string | null }) {
  return error ? <p className="error-text">{error}</p> : null
}

export function Empty({ children }: { children: ReactNode }) {
  return <p className="empty">{children}</p>
}

/** Small confirm helper for destructive actions. */
export function confirmDelete(what: string): boolean {
  return window.confirm(`Slet ${what}? Dette kan ikke fortrydes.`)
}
