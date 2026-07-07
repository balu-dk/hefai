import { lazy, Suspense, useEffect, useMemo, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Drawing, DrawingData, DrawingVersion, Opening, Point, Wall } from '../api/types'
import { ErrorText, useLoad } from '../components'

// three.js er tung — hent den først når 3D-fanen åbnes.
const Drawing3DView = lazy(() => import('./Drawing3DView'))

// The drawing canvas works in world millimetres. SVG viewBox handles
// zoom/pan; snapping is 100 mm.

const SNAP = 100
const emptyData: DrawingData = { walls: [], rooms: [], openings: [], plot: null, trees: [] }

type Tool = 'select' | 'wall' | 'room' | 'door' | 'window' | 'plot' | 'tree' | 'pan'

interface ViewBox { x: number; y: number; w: number; h: number }

export default function DrawingEditorPage() {
  const { drawingId } = useParams()
  const { data: drawing } = useLoad(() => api.get<Drawing>(`/drawings/${drawingId}`), [drawingId])
  const { data: versions, reload: reloadVersions } = useLoad(
    () => api.get<DrawingVersion[]>(`/drawings/${drawingId}/versions`), [drawingId])

  const [data, setData] = useState<DrawingData>(emptyData)
  const [loadedVersion, setLoadedVersion] = useState<number | null>(null)
  const [tool, setTool] = useState<Tool>('wall')
  const [thickness, setThickness] = useState(300)
  const [loadBearing, setLoadBearing] = useState(true)
  const [openingWidth, setOpeningWidth] = useState(900)
  const [pending, setPending] = useState<Point[]>([])
  const [cursor, setCursor] = useState<Point | null>(null)
  const [selected, setSelected] = useState<{ kind: 'wall' | 'opening' | 'room' | 'tree'; index: number } | null>(null)
  const [view, setView] = useState<'2d' | '3d'>('2d')
  const [treeHeight, setTreeHeight] = useState(6000)
  const [treeCrown, setTreeCrown] = useState(4000)
  const [viewBox, setViewBox] = useState<ViewBox>({ x: -2000, y: -2000, w: 16000, h: 12000 })
  const [error, setError] = useState<string | null>(null)
  const [saveNote, setSaveNote] = useState('')
  const [saving, setSaving] = useState(false)
  const svgRef = useRef<SVGSVGElement>(null)
  const panState = useRef<{ start: Point; box: ViewBox } | null>(null)

  // Load the newest version once versions arrive.
  useEffect(() => {
    if (versions && versions.length > 0 && loadedVersion === null) {
      setData({ ...emptyData, ...versions[0].data })
      setLoadedVersion(versions[0].versionNo)
    }
  }, [versions, loadedVersion])

  const toWorld = (e: React.MouseEvent): Point => {
    const svg = svgRef.current!
    const pt = svg.createSVGPoint()
    pt.x = e.clientX
    pt.y = e.clientY
    const p = pt.matrixTransform(svg.getScreenCTM()!.inverse())
    return { x: Math.round(p.x / SNAP) * SNAP, y: Math.round(p.y / SNAP) * SNAP }
  }

  const rawWorld = (e: React.MouseEvent): Point => {
    const svg = svgRef.current!
    const pt = svg.createSVGPoint()
    pt.x = e.clientX
    pt.y = e.clientY
    const p = pt.matrixTransform(svg.getScreenCTM()!.inverse())
    return { x: p.x, y: p.y }
  }

  function nearestWall(p: Point): { wall: Wall; index: number; offset: number } | null {
    type Hit = { wall: Wall; index: number; offset: number; dist: number }
    const best = data.walls.reduce<Hit | null>((acc, w, index) => {
      const dx = w.to.x - w.from.x
      const dy = w.to.y - w.from.y
      const len2 = dx * dx + dy * dy
      if (len2 === 0) return acc
      const t = Math.max(0, Math.min(1, ((p.x - w.from.x) * dx + (p.y - w.from.y) * dy) / len2))
      const dist = Math.hypot(p.x - (w.from.x + t * dx), p.y - (w.from.y + t * dy))
      if (acc && acc.dist <= dist) return acc
      return { wall: w, index, offset: t * Math.sqrt(len2), dist }
    }, null)
    return best && best.dist < 500 ? best : null
  }

  function handleClick(e: React.MouseEvent) {
    if (tool === 'pan') return
    const p = toWorld(e)

    if (tool === 'wall') {
      if (pending.length === 0) {
        setPending([p])
      } else {
        const from = pending[0]
        if (from.x !== p.x || from.y !== p.y) {
          const wall: Wall = {
            id: `w${Date.now()}${data.walls.length}`,
            from, to: p, thicknessMm: thickness, isLoadBearing: loadBearing,
          }
          setData((d) => ({ ...d, walls: [...d.walls, wall] }))
        }
        setPending([p]) // chain: next wall starts here
      }
    } else if (tool === 'room' || tool === 'plot') {
      setPending((prev) => [...prev, p])
    } else if (tool === 'door' || tool === 'window') {
      const hit = nearestWall(rawWorld(e))
      if (!hit) return setError('Klik tættere på en væg for at placere åbningen.')
      setError(null)
      const opening: Opening = {
        wallId: hit.wall.id,
        type: tool,
        offsetMm: Math.max(0, Math.round((hit.offset - openingWidth / 2) / SNAP) * SNAP),
        widthMm: openingWidth,
        heightMm: tool === 'door' ? 2100 : 1200,
      }
      setData((d) => ({ ...d, openings: [...d.openings, opening] }))
    } else if (tool === 'tree') {
      setData((d) => ({
        ...d,
        trees: [...(d.trees ?? []), { position: p, heightMm: treeHeight, crownDiameterMm: treeCrown }],
      }))
    } else if (tool === 'select') {
      const world = rawWorld(e)
      // Trees first (small targets), then walls.
      const treeIndex = (data.trees ?? []).findIndex(
        (t) => Math.hypot(world.x - t.position.x, world.y - t.position.y) < Math.max(t.crownDiameterMm / 2, 600),
      )
      if (treeIndex >= 0) {
        setSelected({ kind: 'tree', index: treeIndex })
        return
      }
      const hit = nearestWall(world)
      setSelected(hit ? { kind: 'wall', index: hit.index } : null)
    }
  }

  function handleDoubleClick() {
    if (tool === 'room' && pending.length >= 3) {
      const name = window.prompt('Rummets navn:')
      if (name) {
        setData((d) => ({ ...d, rooms: [...d.rooms, { name, polygon: pending }] }))
      }
      setPending([])
    } else if (tool === 'plot' && pending.length >= 3) {
      setData((d) => ({
        ...d,
        plot: { boundary: pending, offset: d.plot?.offset ?? { x: 0, y: 0 }, rotationDeg: d.plot?.rotationDeg ?? 0 },
      }))
      setPending([])
    } else if (tool === 'wall') {
      setPending([])
    }
  }

  function handleMouseDown(e: React.MouseEvent) {
    if (tool === 'pan' || e.button === 1) {
      panState.current = { start: { x: e.clientX, y: e.clientY }, box: viewBox }
      e.preventDefault()
    }
  }

  function handleMouseMove(e: React.MouseEvent) {
    if (panState.current) {
      const scale = viewBox.w / (svgRef.current?.clientWidth ?? 1)
      const dx = (e.clientX - panState.current.start.x) * scale
      const dy = (e.clientY - panState.current.start.y) * scale
      setViewBox({ ...panState.current.box, x: panState.current.box.x - dx, y: panState.current.box.y - dy })
      return
    }
    setCursor(toWorld(e))
  }

  function handleWheel(e: React.WheelEvent) {
    const factor = e.deltaY > 0 ? 1.15 : 1 / 1.15
    const p = rawWorld(e as unknown as React.MouseEvent)
    setViewBox((vb) => ({
      x: p.x - (p.x - vb.x) * factor,
      y: p.y - (p.y - vb.y) * factor,
      w: vb.w * factor,
      h: vb.h * factor,
    }))
  }

  function deleteSelected() {
    if (!selected) return
    if (selected.kind === 'wall') {
      const wall = data.walls[selected.index]
      setData((d) => ({
        ...d,
        walls: d.walls.filter((_, i) => i !== selected.index),
        openings: d.openings.filter((o) => o.wallId !== wall.id),
      }))
    } else if (selected.kind === 'tree') {
      setData((d) => ({ ...d, trees: (d.trees ?? []).filter((_, i) => i !== selected.index) }))
    }
    setSelected(null)
  }

  async function save() {
    setSaving(true)
    setError(null)
    try {
      await api.post(`/drawings/${drawingId}/versions`, { data, scale: '1:100', note: saveNote })
      setSaveNote('')
      reloadVersions()
      setLoadedVersion(null) // reload newest
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setSaving(false)
    }
  }

  const totalArea = useMemo(
    () => data.rooms.reduce((sum, r) => sum + polygonAreaM2(r.polygon), 0),
    [data.rooms],
  )

  const tools: { key: Tool; label: string }[] = [
    { key: 'select', label: 'Vælg' },
    { key: 'wall', label: 'Væg' },
    { key: 'room', label: 'Rum' },
    { key: 'door', label: 'Dør' },
    { key: 'window', label: 'Vindue' },
    { key: 'plot', label: 'Grund/skel' },
    { key: 'tree', label: 'Træ' },
    { key: 'pan', label: 'Panorér' },
  ]

  return (
    <>
      <h1>{drawing?.title ?? 'Tegning'}</h1>
      <p className="page-sub">
        Mål i mm, snap {SNAP} mm. Version: {loadedVersion ?? 'ny'} · Samlet rumareal: {totalArea.toFixed(1)} m²
      </p>

      <div className="canvas-toolbar">
        <button className={`tool ${view === '2d' ? 'active' : ''}`} onClick={() => setView('2d')}>2D-tegning</button>
        <button className={`tool ${view === '3d' ? 'active' : ''}`} onClick={() => setView('3d')}>3D-model</button>
        <span style={{ width: 12 }} />
        {view === '2d' && tools.map((t) => (
          <button
            key={t.key}
            className={`tool ${tool === t.key ? 'active' : ''}`}
            onClick={() => (setTool(t.key), setPending([]), setSelected(null))}
          >
            {t.label}
          </button>
        ))}
        {tool === 'wall' && (
          <>
            <label style={{ fontSize: 12.5 }}>
              Tykkelse{' '}
              <input type="number" value={thickness} step={50} min={50} style={{ width: 70 }}
                onChange={(e) => setThickness(Number(e.target.value))} /> mm
            </label>
            <label style={{ fontSize: 12.5 }}>
              <input type="checkbox" checked={loadBearing} onChange={(e) => setLoadBearing(e.target.checked)} /> bærende
            </label>
          </>
        )}
        {(tool === 'door' || tool === 'window') && (
          <label style={{ fontSize: 12.5 }}>
            Bredde{' '}
            <input type="number" value={openingWidth} step={100} min={100} style={{ width: 70 }}
              onChange={(e) => setOpeningWidth(Number(e.target.value))} /> mm
          </label>
        )}
        {view === '2d' && tool === 'tree' && (
          <>
            <label style={{ fontSize: 12.5 }}>
              Højde{' '}
              <input type="number" value={treeHeight} step={500} min={500} style={{ width: 70 }}
                onChange={(e) => setTreeHeight(Number(e.target.value))} /> mm
            </label>
            <label style={{ fontSize: 12.5 }}>
              Krone{' '}
              <input type="number" value={treeCrown} step={500} min={500} style={{ width: 70 }}
                onChange={(e) => setTreeCrown(Number(e.target.value))} /> mm
            </label>
          </>
        )}
        {view === '2d' && selected && <button className="tool" onClick={deleteSelected}>Slet valgte</button>}
        <span style={{ flex: 1 }} />
        <label style={{ fontSize: 12.5 }}>
          Væghøjde{' '}
          <input type="number" value={data.wallHeightMm || 2500} step={100} min={2000} style={{ width: 70 }}
            onChange={(e) => setData((d) => ({ ...d, wallHeightMm: Number(e.target.value) }))} /> mm
        </label>
        <label style={{ fontSize: 12.5 }}>
          Taghældning{' '}
          <input type="number" value={data.roofAngleDeg ?? 0} step={5} min={0} max={60} style={{ width: 55 }}
            onChange={(e) => setData((d) => ({ ...d, roofAngleDeg: Number(e.target.value) }))} /> °
        </label>
      </div>

      {view === '3d' && (
        <>
          <Suspense fallback={<div className="card">Indlæser 3D-visning…</div>}>
            <Drawing3DView data={data} />
          </Suspense>
          <p className="hint">
            Træk for at rotere, scroll for at zoome, højreklik-træk for at flytte. Modellen bygges direkte af
            din 2D-tegning: vægge får væghøjden, taget følger taghældningen, og træer/grund vises som på
            situationsplanen. Gem stadig via 2D-fanen.
          </p>
        </>
      )}
      {view === '2d' && (
      <svg
        ref={svgRef}
        className={`drawing-svg ${tool === 'pan' ? 'pan' : ''}`}
        viewBox={`${viewBox.x} ${viewBox.y} ${viewBox.w} ${viewBox.h}`}
        style={{ height: '58vh' }}
        onClick={handleClick}
        onDoubleClick={handleDoubleClick}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={() => (panState.current = null)}
        onMouseLeave={() => (panState.current = null)}
        onWheel={handleWheel}
      >
        <Grid viewBox={viewBox} />
        {/* plot boundary */}
        {data.plot && data.plot.boundary.length >= 3 && (
          <polygon
            points={data.plot.boundary.map((p) => `${p.x},${p.y}`).join(' ')}
            fill="#e9f3e9" stroke="#2e7d43" strokeWidth={viewBox.w / 500} strokeDasharray={`${viewBox.w / 100},${viewBox.w / 200}`}
          />
        )}
        {/* rooms */}
        {data.rooms.map((r, i) => (
          <g key={i}>
            <polygon points={r.polygon.map((p) => `${p.x},${p.y}`).join(' ')} fill="#f2ede4" stroke="#c9b899" strokeWidth={viewBox.w / 800} />
            <text x={centroid(r.polygon).x} y={centroid(r.polygon).y} textAnchor="middle" fontSize={viewBox.w / 55} fill="#7d766b">
              {r.name} ({polygonAreaM2(r.polygon).toFixed(1)} m²)
            </text>
          </g>
        ))}
        {/* walls */}
        {data.walls.map((w, i) => (
          <g key={w.id}>
            <line
              x1={w.from.x} y1={w.from.y} x2={w.to.x} y2={w.to.y}
              stroke={selected?.kind === 'wall' && selected.index === i ? '#b4551e' : w.isLoadBearing ? '#26221c' : '#8d867a'}
              strokeWidth={w.thicknessMm}
              strokeLinecap="square"
            />
            <text
              x={(w.from.x + w.to.x) / 2} y={(w.from.y + w.to.y) / 2 - w.thicknessMm}
              textAnchor="middle" fontSize={viewBox.w / 70} fill="#b4551e"
            >
              {Math.round(Math.hypot(w.to.x - w.from.x, w.to.y - w.from.y))}
            </text>
          </g>
        ))}
        {/* openings */}
        {data.openings.map((o, i) => {
          const wall = data.walls.find((w) => w.id === o.wallId)
          if (!wall) return null
          const len = Math.hypot(wall.to.x - wall.from.x, wall.to.y - wall.from.y)
          if (len === 0) return null
          const ux = (wall.to.x - wall.from.x) / len
          const uy = (wall.to.y - wall.from.y) / len
          const cx = wall.from.x + ux * (o.offsetMm + o.widthMm / 2)
          const cy = wall.from.y + uy * (o.offsetMm + o.widthMm / 2)
          return (
            <line
              key={i}
              x1={cx - (ux * o.widthMm) / 2} y1={cy - (uy * o.widthMm) / 2}
              x2={cx + (ux * o.widthMm) / 2} y2={cy + (uy * o.widthMm) / 2}
              stroke={o.type === 'window' ? '#2a5f9e' : '#a06a2c'}
              strokeWidth={wall.thicknessMm * 1.2}
            />
          )
        })}
        {/* pending preview */}
        {pending.length > 0 && cursor && tool === 'wall' && (
          <line x1={pending[0].x} y1={pending[0].y} x2={cursor.x} y2={cursor.y} stroke="#b4551e" strokeWidth={thickness} opacity={0.4} />
        )}
        {(tool === 'room' || tool === 'plot') && pending.length > 0 && (
          <polyline
            points={[...pending, cursor ?? pending[pending.length - 1]].map((p) => `${p.x},${p.y}`).join(' ')}
            fill="none" stroke={tool === 'plot' ? '#2e7d43' : '#c9b899'} strokeWidth={viewBox.w / 400}
          />
        )}
        {/* trees */}
        {(data.trees ?? []).map((t, i) => (
          <g key={`tree${i}`} opacity={0.9}>
            <circle cx={t.position.x} cy={t.position.y} r={t.crownDiameterMm / 2}
              fill="#cde5c8" stroke={selected?.kind === 'tree' && selected.index === i ? '#b4551e' : '#3f7d3a'}
              strokeWidth={viewBox.w / 600} />
            <circle cx={t.position.x} cy={t.position.y} r={150} fill="#6d4c33" />
          </g>
        ))}
        {cursor && tool !== 'select' && tool !== 'pan' && (
          <circle cx={cursor.x} cy={cursor.y} r={viewBox.w / 250} fill="#b4551e" />
        )}
      </svg>
      )}

      {view === '2d' && <p className="hint">
        {tool === 'wall' && 'Klik for at starte en væg, klik igen for at afslutte den (kæder videre) — dobbeltklik for at stoppe kæden.'}
        {tool === 'room' && 'Klik rummets hjørner, dobbeltklik for at lukke polygonen og navngive rummet.'}
        {tool === 'plot' && 'Klik skellets hjørner, dobbeltklik for at lukke. Grunden vises med grøn stiplet linje.'}
        {(tool === 'door' || tool === 'window') && 'Klik på en væg for at placere åbningen.'}
        {tool === 'tree' && 'Klik på grunden for at plante et træ — højde og kronediameter sættes i værktøjslinjen.'}
        {tool === 'select' && 'Klik på en væg eller et træ for at vælge — brug "Slet valgte" i værktøjslinjen.'}
        {tool === 'pan' && 'Træk for at panorere. Scroll for at zoome.'}
      </p>}

      {view === '2d' && data.plot && (
        <div className="form-row">
          <div className="field">
            <label>Bygningens placering på grunden — X (mm)</label>
            <input type="number" value={data.plot.offset.x}
              onChange={(e) => setData((d) => ({ ...d, plot: { ...d.plot!, offset: { ...d.plot!.offset, x: Number(e.target.value) } } }))} />
          </div>
          <div className="field">
            <label>Y (mm)</label>
            <input type="number" value={data.plot.offset.y}
              onChange={(e) => setData((d) => ({ ...d, plot: { ...d.plot!, offset: { ...d.plot!.offset, y: Number(e.target.value) } } }))} />
          </div>
          <div className="field">
            <label>Rotation (grader)</label>
            <input type="number" value={data.plot.rotationDeg}
              onChange={(e) => setData((d) => ({ ...d, plot: { ...d.plot!, rotationDeg: Number(e.target.value) } }))} />
          </div>
        </div>
      )}

      <ErrorText error={error} />
      <div className="form-row">
        <div className="field" style={{ flex: 1 }}>
          <label>Versionsnote</label>
          <input value={saveNote} onChange={(e) => setSaveNote(e.target.value)} placeholder="Hvad er ændret?" />
        </div>
        <button className="btn" onClick={save} disabled={saving}>
          {saving ? 'Gemmer…' : `Gem som version ${(versions?.[0]?.versionNo ?? 0) + 1}`}
        </button>
      </div>

      {versions && versions.length > 0 && (
        <div className="card" style={{ marginTop: 14 }}>
          <h3>Versioner</h3>
          <table className="tbl">
            <tbody>
              {versions.map((v) => (
                <tr key={v.id}>
                  <td>v{v.versionNo}</td>
                  <td>{v.note || '–'}</td>
                  <td>{new Date(v.createdAt).toLocaleString('da-DK')}</td>
                  <td>
                    <button className="btn small secondary"
                      onClick={() => (setData({ ...emptyData, ...v.data }), setLoadedVersion(v.versionNo))}>
                      Indlæs
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  )
}

function Grid({ viewBox }: { viewBox: ViewBox }) {
  // 1 m grid lines across the visible area.
  const step = 1000
  const lines: React.ReactElement[] = []
  const x0 = Math.floor(viewBox.x / step) * step
  const y0 = Math.floor(viewBox.y / step) * step
  for (let x = x0; x <= viewBox.x + viewBox.w; x += step) {
    lines.push(<line key={`v${x}`} x1={x} y1={viewBox.y} x2={x} y2={viewBox.y + viewBox.h} stroke="#f0ede7" strokeWidth={viewBox.w / 1200} />)
  }
  for (let y = y0; y <= viewBox.y + viewBox.h; y += step) {
    lines.push(<line key={`h${y}`} x1={viewBox.x} y1={y} x2={viewBox.x + viewBox.w} y2={y} stroke="#f0ede7" strokeWidth={viewBox.w / 1200} />)
  }
  return <g>{lines}</g>
}

function centroid(polygon: Point[]): Point {
  const n = polygon.length || 1
  return {
    x: polygon.reduce((s, p) => s + p.x, 0) / n,
    y: polygon.reduce((s, p) => s + p.y, 0) / n,
  }
}

function polygonAreaM2(polygon: Point[]): number {
  if (polygon.length < 3) return 0
  let sum = 0
  for (let i = 0; i < polygon.length; i++) {
    const j = (i + 1) % polygon.length
    sum += polygon[i].x * polygon[j].y - polygon[j].x * polygon[i].y
  }
  return Math.abs(sum) / 2 / 1e6
}
