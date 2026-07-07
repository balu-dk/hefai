// Thin fetch wrapper: JSON in/out, bearer token, Danish error messages
// surfaced from the backend's {error} envelope.

const TOKEN_KEY = 'hefai.token'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string | null) {
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {}
  const token = getToken()
  if (token) headers['Authorization'] = `Bearer ${token}`

  let payload: BodyInit | undefined
  if (body instanceof FormData) {
    payload = body
  } else if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    payload = JSON.stringify(body)
  }

  const res = await fetch(`/api/v1${path}`, { method, headers, body: payload })
  if (res.status === 401 && getToken()) {
    // Session expired: drop the token so the app returns to login.
    setToken(null)
    window.location.reload()
  }
  if (res.status === 204) return undefined as T
  const data = await res.json().catch(() => null)
  if (!res.ok) {
    throw new ApiError(res.status, data?.error ?? `Fejl ${res.status}`)
  }
  return data as T
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  patch: <T>(path: string, body?: unknown) => request<T>('PATCH', path, body),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
  del: (path: string) => request<void>('DELETE', path),
}

/** URL for streaming a document's content (PDF/image viewing). */
export function documentContentURL(documentId: string): string {
  return `/api/v1/documents/${documentId}/content`
}

/** fetch document content with auth and return an object URL for <img>/<iframe>. */
export async function fetchDocumentObjectURL(documentId: string): Promise<string> {
  const res = await fetch(documentContentURL(documentId), {
    headers: { Authorization: `Bearer ${getToken()}` },
  })
  if (!res.ok) throw new ApiError(res.status, 'Kunne ikke hente filen')
  return URL.createObjectURL(await res.blob())
}

// --- formatting helpers --------------------------------------------------------

/** Format øre as "12.345,67 kr." */
export function kr(ore: number): string {
  const negative = ore < 0
  const abs = Math.abs(ore)
  const kroner = Math.floor(abs / 100)
  const rest = String(abs % 100).padStart(2, '0')
  const grouped = kroner.toString().replace(/\B(?=(\d{3})+(?!\d))/g, '.')
  return `${negative ? '−' : ''}${grouped},${rest} kr.`
}

/** Parse "12.345,67" (or "12345.67") into øre. */
export function parseKr(input: string): number | null {
  const cleaned = input.trim().replace(/\s|kr\.?/gi, '')
  if (!cleaned) return null
  // Danish format: dots are thousand separators, comma is decimal.
  const normalized = cleaned.includes(',')
    ? cleaned.replace(/\./g, '').replace(',', '.')
    : cleaned
  const value = Number(normalized)
  if (Number.isNaN(value)) return null
  return Math.round(value * 100)
}

export function formatDate(iso: string | null | undefined): string {
  if (!iso) return '–'
  const d = new Date(iso)
  return d.toLocaleDateString('da-DK', { day: '2-digit', month: '2-digit', year: 'numeric' })
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}
