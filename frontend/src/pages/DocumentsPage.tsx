import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { api, fetchDocumentObjectURL, formatBytes, formatDate } from '../api/client'
import type { Doc, DocumentKind, DocumentLink, LinkTargetType } from '../api/types'
import { confirmDelete, Empty, ErrorText, Modal, useLoad } from '../components'

const kindLabels: Record<DocumentKind, string> = {
  architect_drawing: 'Arkitekttegning',
  construction_drawing: 'Konstruktionstegning',
  receipt: 'Kvittering',
  photo: 'Billede',
  warranty: 'Garantibevis',
  datasheet: 'Datablad',
  permit: 'Tilladelse',
  correspondence: 'Korrespondance',
  generated: 'Genereret',
  other: 'Andet',
}

export default function DocumentsPage() {
  const { projectId } = useParams()
  const [query, setQuery] = useState('')
  const [kind, setKind] = useState('')
  const params = new URLSearchParams()
  if (query) params.set('q', query)
  if (kind) params.set('kind', kind)

  const { data: docs, error, reload } = useLoad(
    () => api.get<Doc[]>(`/projects/${projectId}/documents?${params.toString()}`),
    [projectId, query, kind],
  )
  const [uploading, setUploading] = useState(false)
  const [selected, setSelected] = useState<Doc | null>(null)

  return (
    <>
      <h1>Dokumenter & arkiv</h1>
      <p className="page-sub">Tegninger, kvitteringer, billeder, garantier og datablade — søgbart og tagget.</p>
      <ErrorText error={error} />

      <div className="form-row">
        <div className="field" style={{ flex: 2 }}>
          <label>Søg (titel, beskrivelse, filnavn)</label>
          <input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="fx varmepumpe garanti" />
        </div>
        <div className="field">
          <label>Type</label>
          <select value={kind} onChange={(e) => setKind(e.target.value)}>
            <option value="">Alle</option>
            {Object.entries(kindLabels).map(([k, label]) => (
              <option key={k} value={k}>{label}</option>
            ))}
          </select>
        </div>
        <button className="btn" onClick={() => setUploading(true)}>+ Upload</button>
      </div>

      <div className="doc-grid">
        {docs?.map((d) => (
          <div key={d.id} className="doc-card" onClick={() => setSelected(d)}>
            <div className="name" title={d.title}>{d.title}</div>
            <div className="meta">
              {kindLabels[d.kind]} · {formatBytes(d.sizeBytes)} · {formatDate(d.createdAt)}
            </div>
            <div>
              {d.tags.map((t) => <span key={t} className="tag-chip">{t}</span>)}
            </div>
          </div>
        ))}
      </div>
      {docs?.length === 0 && <Empty>Ingen dokumenter matcher.</Empty>}

      {uploading && (
        <UploadModal projectId={projectId!} onDone={() => (setUploading(false), reload())} onClose={() => setUploading(false)} />
      )}
      {selected && (
        <DocumentViewer doc={selected} projectId={projectId!} onChanged={reload} onClose={() => (setSelected(null), reload())} />
      )}
    </>
  )
}

function UploadModal({ projectId, onDone, onClose }: {
  projectId: string
  onDone: () => void
  onClose: () => void
}) {
  const [file, setFile] = useState<File | null>(null)
  const [title, setTitle] = useState('')
  const [kind, setKind] = useState<DocumentKind>('other')
  const [description, setDescription] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!file) return
    setBusy(true)
    const form = new FormData()
    form.append('file', file)
    form.append('title', title)
    form.append('kind', kind)
    form.append('description', description)
    try {
      await api.post(`/projects/${projectId}/documents`, form)
      onDone()
    } catch (err) {
      setError((err as Error).message)
      setBusy(false)
    }
  }

  return (
    <Modal title="Upload dokument" onClose={onClose}>
      <form onSubmit={submit}>
        <div className="field" style={{ marginBottom: 10 }}>
          <label>Fil * (PDF, billeder, tegninger — maks. 100 MB)</label>
          <input
            type="file"
            onChange={(e) => {
              const f = e.target.files?.[0] ?? null
              setFile(f)
              if (f && !title) setTitle(f.name.replace(/\.[^.]+$/, ''))
            }}
            required
          />
        </div>
        <div className="form-row">
          <div className="field" style={{ flex: 2 }}>
            <label>Titel</label>
            <input value={title} onChange={(e) => setTitle(e.target.value)} />
          </div>
          <div className="field">
            <label>Type</label>
            <select value={kind} onChange={(e) => setKind(e.target.value as DocumentKind)}>
              {Object.entries(kindLabels).map(([k, label]) => (
                <option key={k} value={k}>{label}</option>
              ))}
            </select>
          </div>
        </div>
        <div className="field">
          <label>Beskrivelse</label>
          <textarea value={description} onChange={(e) => setDescription(e.target.value)} />
        </div>
        <ErrorText error={error} />
        <div className="form-row" style={{ justifyContent: 'flex-end', marginTop: 12 }}>
          <button type="button" className="btn secondary" onClick={onClose}>Annullér</button>
          <button className="btn" disabled={!file || busy}>{busy ? 'Uploader…' : 'Upload'}</button>
        </div>
      </form>
    </Modal>
  )
}

const linkTargetLabels: Record<LinkTargetType, string> = {
  phase: 'Fase', task: 'Opgave', room: 'Rum', expense: 'Udgift',
  material: 'Materiale', supplier: 'Leverandør', case_file: 'Byggesag',
  structural_element: 'Konstruktionselement',
}

function DocumentViewer({ doc, projectId, onChanged, onClose }: {
  doc: Doc
  projectId: string
  onChanged: () => void
  onClose: () => void
}) {
  const [objectURL, setObjectURL] = useState<string | null>(null)
  const [tags, setTags] = useState(doc.tags.join(', '))
  const [error, setError] = useState<string | null>(null)
  const { data: links, reload: reloadLinks } = useLoad(
    () => api.get<DocumentLink[]>(`/documents/${doc.id}/links`), [doc.id])
  const [linkType, setLinkType] = useState<LinkTargetType>('room')
  const [linkTargets, setLinkTargets] = useState<{ id: string; label: string }[]>([])
  const [linkTarget, setLinkTarget] = useState('')

  const viewable = doc.mimeType === 'application/pdf' || doc.mimeType.startsWith('image/')

  useEffect(() => {
    if (!viewable) return
    let url: string | null = null
    fetchDocumentObjectURL(doc.id)
      .then((u) => {
        url = u
        setObjectURL(u)
      })
      .catch((e: Error) => setError(e.message))
    return () => {
      if (url) URL.revokeObjectURL(url)
    }
  }, [doc.id, viewable])

  useEffect(() => {
    // Load candidates for the chosen link target type.
    const endpoints: Record<LinkTargetType, { path: string; label: (x: any) => string }> = {
      phase: { path: 'phases', label: (x) => x.name },
      task: { path: 'tasks', label: (x) => x.title },
      room: { path: 'rooms', label: (x) => x.name },
      expense: { path: 'expenses', label: (x) => x.description },
      material: { path: 'materials', label: (x) => x.name },
      supplier: { path: 'suppliers', label: (x) => x.companyName },
      case_file: { path: 'case-files', label: (x) => x.title },
      structural_element: { path: 'structural-elements', label: (x) => x.name },
    }
    const ep = endpoints[linkType]
    api.get<any[]>(`/projects/${projectId}/${ep.path}`).then((items) =>
      setLinkTargets(items.map((x) => ({ id: x.id, label: ep.label(x) }))),
    ).catch(() => setLinkTargets([]))
  }, [linkType, projectId])

  async function saveTags() {
    try {
      await api.put(`/documents/${doc.id}/tags`, { tags: tags.split(',').map((t) => t.trim()).filter(Boolean) })
      onChanged()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function addLink() {
    if (!linkTarget) return
    try {
      await api.post(`/documents/${doc.id}/links`, { targetType: linkType, targetId: linkTarget })
      reloadLinks()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function removeLink(linkId: string) {
    await api.del(`/documents/${doc.id}/links/${linkId}`)
    reloadLinks()
  }

  async function remove() {
    if (!confirmDelete(`dokumentet "${doc.title}"`)) return
    await api.del(`/documents/${doc.id}`)
    onClose()
  }

  return (
    <Modal title={doc.title} wide onClose={onClose}>
      <p className="hint">
        {doc.filename} · {doc.mimeType} · {formatBytes(doc.sizeBytes)}
        {doc.description && <> — {doc.description}</>}
      </p>
      {viewable && objectURL && (
        doc.mimeType === 'application/pdf' ? (
          <iframe className="viewer-frame" src={objectURL} title={doc.title} />
        ) : (
          <img src={objectURL} alt={doc.title} style={{ maxWidth: '100%', borderRadius: 6 }} />
        )
      )}
      {!viewable && objectURL == null && (
        <p className="hint">Filtypen kan ikke vises i browseren.</p>
      )}
      {objectURL && (
        <p style={{ marginTop: 6 }}>
          <a href={objectURL} download={doc.filename}>Download {doc.filename}</a>
        </p>
      )}

      <div className="form-row" style={{ marginTop: 12 }}>
        <div className="field" style={{ flex: 1 }}>
          <label>Tags (kommasepareret)</label>
          <input value={tags} onChange={(e) => setTags(e.target.value)} placeholder="varmepumpe, garanti" />
        </div>
        <button className="btn small secondary" onClick={saveTags}>Gem tags</button>
      </div>

      <h3>Knyttet til</h3>
      {links?.map((l) => (
        <div key={l.id} className="form-row" style={{ alignItems: 'center' }}>
          <span style={{ flex: 1 }}>
            {linkTargetLabels[l.targetType]} <span style={{ color: 'var(--text-dim)', fontSize: 12 }}>({l.targetId.slice(0, 8)}…)</span>
          </span>
          <button className="btn small secondary" onClick={() => removeLink(l.id)}>Fjern</button>
        </div>
      ))}
      {links?.length === 0 && <p className="hint">Ikke knyttet til noget endnu.</p>}
      <div className="form-row">
        <div className="field">
          <label>Type</label>
          <select value={linkType} onChange={(e) => (setLinkType(e.target.value as LinkTargetType), setLinkTarget(''))}>
            {Object.entries(linkTargetLabels).map(([k, label]) => (
              <option key={k} value={k}>{label}</option>
            ))}
          </select>
        </div>
        <div className="field" style={{ flex: 1 }}>
          <label>Mål</label>
          <select value={linkTarget} onChange={(e) => setLinkTarget(e.target.value)}>
            <option value="">Vælg…</option>
            {linkTargets.map((t) => (
              <option key={t.id} value={t.id}>{t.label}</option>
            ))}
          </select>
        </div>
        <button className="btn small secondary" onClick={addLink} disabled={!linkTarget}>+ Knyt</button>
      </div>

      <ErrorText error={error} />
      <div className="form-row" style={{ justifyContent: 'space-between', marginTop: 12 }}>
        <button className="btn danger" onClick={remove}>Slet dokument</button>
        <button className="btn" onClick={onClose}>Luk</button>
      </div>
    </Modal>
  )
}
