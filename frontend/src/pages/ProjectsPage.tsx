import { useEffect, useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { AddressDetails, AddressSuggestion, Project } from '../api/types'
import { Empty, ErrorText, Modal, StatusBadge, statusLabel, useLoad } from '../components'
import { useAuth } from '../auth'

export default function ProjectsPage() {
  const { user, logout } = useAuth()
  const { data: projects, error, reload } = useLoad(() => api.get<Project[]>('/projects'), [])
  const [showCreate, setShowCreate] = useState(false)

  return (
    <div style={{ maxWidth: 860, margin: '0 auto', padding: '40px 20px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}>
        <h1>
          Hef<span style={{ color: 'var(--accent)' }}>ai</span>
        </h1>
        <div style={{ fontSize: 13, color: 'var(--text-dim)' }}>
          {user?.displayName} ·{' '}
          <a href="#" onClick={(e) => (e.preventDefault(), logout())}>
            Log ud
          </a>
        </div>
      </div>
      <p className="page-sub">Dine byggeprojekter</p>
      <ErrorText error={error} />

      <div className="grid cols-2">
        {projects?.map((p) => (
          <Link key={p.id} to={`/projects/${p.id}`} style={{ textDecoration: 'none', color: 'inherit' }}>
            <div className="card" style={{ marginBottom: 0, height: '100%' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', gap: 8 }}>
                <strong style={{ fontSize: 16 }}>{p.name}</strong>
                <StatusBadge status={p.status} />
              </div>
              <div style={{ color: 'var(--text-dim)', fontSize: 13, marginTop: 6 }}>
                {statusLabel(p.kind)}
                {p.address && <> · {p.address}</>}
                {p.municipality && <> · {p.municipality} Kommune</>}
              </div>
              {p.description && <p style={{ marginBottom: 0 }}>{p.description}</p>}
            </div>
          </Link>
        ))}
      </div>
      {projects && projects.length === 0 && <Empty>Ingen projekter endnu — opret dit første.</Empty>}

      <div style={{ marginTop: 20 }}>
        <button className="btn" onClick={() => setShowCreate(true)}>
          + Nyt projekt
        </button>
      </div>

      {showCreate && <CreateProjectModal onDone={() => (setShowCreate(false), reload())} onClose={() => setShowCreate(false)} />}
    </div>
  )
}

// Anbefalede dokumenter der spørges efter helt fra start. Manglende
// dokumenter bliver automatisk til opgaver.
const startDocuments = [
  { key: 'skoede', label: 'Skøde / købsaftale', hint: 'ejerforhold og matrikel' },
  { key: 'bbr', label: 'BBR-meddelelse', hint: 'registrerede bygninger og arealer' },
  { key: 'lokalplan', label: 'Lokalplan', hint: 'Hefai kan finde den ud fra adressen' },
  { key: 'tegninger', label: 'Eksisterende tegninger', hint: 'ved renovering/tilbygning' },
  { key: 'forsikring', label: 'Forsikringspolice', hint: 'husk entrepriseforsikring ved større arbejder' },
]

// Projektguiden: fire små, venlige trin i stedet for én stor formular.
function CreateProjectModal({ onDone, onClose }: { onDone: () => void; onClose: () => void }) {
  const [step, setStep] = useState(0)
  const [name, setName] = useState('')
  const [kind, setKind] = useState('new_build')
  const [address, setAddress] = useState('')
  const [municipality, setMunicipality] = useState('')
  const [cadastralId, setCadastralId] = useState('')
  const [plotArea, setPlotArea] = useState('')
  const [geo, setGeo] = useState<AddressDetails | null>(null)
  const [description, setDescription] = useState('')
  const [haveDocs, setHaveDocs] = useState<Record<string, boolean>>({})
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  // Adressesøgning mod DAWA (debounced). pickedText stopper gen-søgning
  // efter et valg.
  const [addressQuery, setAddressQuery] = useState('')
  const [pickedText, setPickedText] = useState('')
  const [suggestions, setSuggestions] = useState<AddressSuggestion[]>([])
  const [lookupBusy, setLookupBusy] = useState(false)
  useEffect(() => {
    if (addressQuery.trim().length < 3 || addressQuery === pickedText) {
      setSuggestions([])
      return
    }
    const timer = setTimeout(() => {
      api.get<AddressSuggestion[]>(`/lookup/address?q=${encodeURIComponent(addressQuery)}`)
        .then(setSuggestions)
        .catch(() => setSuggestions([]))
    }, 250)
    return () => clearTimeout(timer)
  }, [addressQuery, pickedText])

  async function pickAddress(s: AddressSuggestion) {
    setSuggestions([])
    setPickedText(s.text)
    setAddressQuery(s.text)
    setAddress(s.text)
    setLookupBusy(true)
    setError(null)
    try {
      const d = await api.get<AddressDetails>(`/lookup/address/${s.id}`)
      setGeo(d)
      setAddress(d.address || s.text)
      setMunicipality(d.municipality)
      setCadastralId(d.cadastralId)
      if (d.plotAreaM2 != null) setPlotArea(String(d.plotAreaM2))
    } catch (err) {
      setError('Detaljeopslag fejlede (' + (err as Error).message + ') — udfyld felterne manuelt.')
    } finally {
      setLookupBusy(false)
    }
  }

  const kinds = [
    { key: 'new_build', emoji: '🏠', name: 'Nybyggeri', desc: 'Vi bygger et nyt hus eller sommerhus' },
    { key: 'renovation', emoji: '🔨', name: 'Renovering', desc: 'Vi sætter noget eksisterende i stand' },
    { key: 'extension', emoji: '📐', name: 'Tilbygning', desc: 'Vi bygger til eller ud' },
    { key: 'other', emoji: '✨', name: 'Andet', desc: 'Noget helt fjerde' },
  ]

  async function create() {
    setBusy(true)
    setError(null)
    try {
      const project = await api.post<Project>('/projects', {
        name, kind, address, municipality, cadastralId, description,
        plotAreaM2: plotArea ? Number(plotArea) : null,
        latitude: geo?.lat ?? null,
        longitude: geo?.lon ?? null,
        utmX: geo?.utmX ?? null,
        utmY: geo?.utmY ?? null,
      })
      // Manglende dokumenter bliver til konkrete opgaver fra dag ét.
      for (const doc of startDocuments) {
        if (!haveDocs[doc.key]) {
          await api.post(`/projects/${project.id}/tasks`, {
            title: `Skaf og upload: ${doc.label}`,
            description: doc.hint,
          })
        }
      }
      onDone()
    } catch (err) {
      setError((err as Error).message)
      setBusy(false)
    }
  }

  function next(e: FormEvent) {
    e.preventDefault()
    if (step < 3) setStep(step + 1)
    else void create()
  }

  return (
    <Modal title={['Hvad skal der bygges?', 'Hvor bygger I?', 'Hvilke dokumenter har I?', 'Klar til at gå i gang!'][step]} onClose={onClose}>
      <p className="hint" style={{ marginTop: -6 }}>Trin {step + 1} af 4</p>
      <form onSubmit={next}>
        {step === 0 && (
          <>
            <div className="wizard-kinds">
              {kinds.map((k) => (
                <div key={k.key} className={`wizard-kind ${kind === k.key ? 'active' : ''}`} onClick={() => setKind(k.key)}>
                  <div className="emoji">{k.emoji}</div>
                  <div className="name">{k.name}</div>
                  <div className="desc">{k.desc}</div>
                </div>
              ))}
            </div>
            <div className="field" style={{ marginTop: 10 }}>
              <label>Hvad skal projektet hedde? *</label>
              <input value={name} onChange={(e) => setName(e.target.value)} required autoFocus
                placeholder="Fx Sommerhuset i Marielyst" />
            </div>
          </>
        )}
        {step === 1 && (
          <>
            <p className="hint">
              Søg adressen frem, så henter Hefai selv kommune, matrikelnummer, grundareal og
              koordinater fra de officielle registre (gratis og automatisk).
            </p>
            <div className="field" style={{ position: 'relative', marginBottom: 10 }}>
              <label>Søg adressen</label>
              <input value={addressQuery} onChange={(e) => setAddressQuery(e.target.value)} autoFocus
                placeholder="Fx Strandvejen 12, Væggerløse" />
              {suggestions.length > 0 && (
                <div style={{
                  position: 'absolute', top: '100%', left: 0, right: 0, zIndex: 10,
                  background: '#fff', border: '1px solid var(--border)', borderRadius: 6,
                  boxShadow: 'var(--shadow)', maxHeight: 220, overflowY: 'auto',
                }}>
                  {suggestions.map((s) => (
                    <div key={s.id + s.text} className="address-suggestion"
                      style={{ padding: '7px 10px', cursor: 'pointer' }}
                      onClick={() => pickAddress(s)}>
                      {s.text}
                    </div>
                  ))}
                </div>
              )}
            </div>
            {lookupBusy && <p className="hint">Henter matrikel, grundareal og koordinater…</p>}
            {geo && !lookupBusy && (
              <div className="notice" style={{ background: 'var(--green-soft)', color: 'var(--green)', borderColor: '#bfe0c6' }}>
                ✓ Fundet: {geo.municipality} Kommune · matrikel {geo.cadastralId || 'ukendt'}
                {geo.plotAreaM2 != null && <> · grund {geo.plotAreaM2} m²</>} · koordinater hentet til luftfoto
              </div>
            )}
            <div className="form-row">
              <div className="field" style={{ flex: 2 }}>
                <label>Adresse</label>
                <input value={address} onChange={(e) => setAddress(e.target.value)} />
              </div>
              <div className="field">
                <label>Kommune</label>
                <input value={municipality} onChange={(e) => setMunicipality(e.target.value)} />
              </div>
            </div>
            <div className="form-row">
              <div className="field">
                <label>Grundens størrelse (m²)</label>
                <input type="number" step="0.1" min="0" value={plotArea} onChange={(e) => setPlotArea(e.target.value)}
                  placeholder="fx 1200" />
              </div>
              <div className="field">
                <label>Matrikel</label>
                <input value={cadastralId} onChange={(e) => setCadastralId(e.target.value)} />
              </div>
            </div>
          </>
        )}
        {step === 2 && (
          <>
            <p className="hint">
              Sæt kryds ved det I allerede har liggende. Det der mangler, bliver automatisk til
              opgaver, så intet glemmes.
            </p>
            {startDocuments.map((doc) => (
              <label key={doc.key} style={{ display: 'block', padding: '8px 10px', border: '1px solid var(--border)', borderRadius: 8, marginBottom: 6, cursor: 'pointer' }}>
                <input type="checkbox" checked={!!haveDocs[doc.key]}
                  onChange={(e) => setHaveDocs((h) => ({ ...h, [doc.key]: e.target.checked }))} />{' '}
                <strong>{doc.label}</strong>
                <span style={{ color: 'var(--text-dim)' }}> — {doc.hint}</span>
              </label>
            ))}
          </>
        )}
        {step === 3 && (
          <>
            <div className="field">
              <label>Beskriv kort hvad I drømmer om (valgfrit)</label>
              <textarea rows={4} value={description} onChange={(e) => setDescription(e.target.value)}
                placeholder="Fx: Et sommerhus på ca. 70 m² med stort køkken-alrum og terrasse mod vest…" autoFocus />
            </div>
            <div className="notice">
              Når projektet er oprettet, får du en kort rundvisning, og på Overblik venter en
              "Kom godt i gang"-liste der guider jer igennem de første skridt.
            </div>
          </>
        )}
        <ErrorText error={error} />
        <div className="form-row" style={{ marginTop: 12, justifyContent: 'space-between' }}>
          <div>
            {step > 0 && (
              <button type="button" className="btn secondary" onClick={() => setStep(step - 1)}>Tilbage</button>
            )}
          </div>
          <div className="row-actions">
            <button type="button" className="btn secondary" onClick={onClose}>Annullér</button>
            <button className="btn" disabled={busy || (step === 0 && !name.trim())}>
              {step < 3 ? 'Næste' : busy ? 'Opretter…' : 'Opret projekt 🎉'}
            </button>
          </div>
        </div>
      </form>
    </Modal>
  )
}
