import type { DrawingData, Point, RoomShape } from '../api/types'

// Automatisk rum-detektion: væggene rasteriseres på et 100 mm-gitter,
// ydersiden flood-filles, og de tiloversblevne lukkede lommer er rum.
// Deterministisk og hurtig for huse i normal størrelse.

const CELL = 100 // mm

export function detectRooms(data: DrawingData): Point[][] {
  if (data.walls.length < 3) return []

  const xs = data.walls.flatMap((w) => [w.from.x, w.to.x])
  const ys = data.walls.flatMap((w) => [w.from.y, w.to.y])
  const maxThickness = Math.max(...data.walls.map((w) => w.thicknessMm), CELL)
  const pad = maxThickness + 2 * CELL
  const minX = Math.min(...xs) - pad
  const minY = Math.min(...ys) - pad
  const cols = Math.ceil((Math.max(...xs) + pad - minX) / CELL)
  const rows = Math.ceil((Math.max(...ys) + pad - minY) / CELL)
  if (cols * rows > 4_000_000) return [] // urealistisk stor tegning

  const blocked = new Uint8Array(cols * rows)
  const idx = (cx: number, cy: number) => cy * cols + cx

  // Rasterisér vægge med deres tykkelse.
  for (const w of data.walls) {
    const len = Math.hypot(w.to.x - w.from.x, w.to.y - w.from.y)
    const steps = Math.max(1, Math.ceil(len / (CELL / 2)))
    const radius = Math.max(1, Math.round(w.thicknessMm / 2 / CELL))
    for (let s = 0; s <= steps; s++) {
      const px = w.from.x + ((w.to.x - w.from.x) * s) / steps
      const py = w.from.y + ((w.to.y - w.from.y) * s) / steps
      const cx = Math.round((px - minX) / CELL)
      const cy = Math.round((py - minY) / CELL)
      for (let dy = -radius; dy <= radius; dy++) {
        for (let dx = -radius; dx <= radius; dx++) {
          const nx = cx + dx
          const ny = cy + dy
          if (nx >= 0 && ny >= 0 && nx < cols && ny < rows) blocked[idx(nx, ny)] = 1
        }
      }
    }
  }

  // Flood fill fra kanten: alt der kan nås udefra er "ude".
  const outside = new Uint8Array(cols * rows)
  const queue: number[] = []
  for (let cx = 0; cx < cols; cx++) {
    queue.push(idx(cx, 0), idx(cx, rows - 1))
  }
  for (let cy = 0; cy < rows; cy++) {
    queue.push(idx(0, cy), idx(cols - 1, cy))
  }
  while (queue.length) {
    const i = queue.pop()!
    if (outside[i] || blocked[i]) continue
    outside[i] = 1
    const cx = i % cols
    const cy = (i / cols) | 0
    if (cx > 0) queue.push(i - 1)
    if (cx < cols - 1) queue.push(i + 1)
    if (cy > 0) queue.push(i - cols)
    if (cy < rows - 1) queue.push(i + cols)
  }

  // Indvendige regioner grupperes.
  const region = new Int32Array(cols * rows).fill(-1)
  const polygons: Point[][] = []
  let regionCount = 0
  for (let i = 0; i < cols * rows; i++) {
    if (blocked[i] || outside[i] || region[i] >= 0) continue
    const cells: number[] = []
    const stack = [i]
    region[i] = regionCount
    while (stack.length) {
      const c = stack.pop()!
      cells.push(c)
      const cx = c % cols
      const cy = (c / cols) | 0
      for (const n of [c - 1, c + 1, c - cols, c + cols]) {
        const nx = n % cols
        const ny = (n / cols) | 0
        if (n < 0 || n >= cols * rows || Math.abs(nx - cx) + Math.abs(ny - cy) !== 1) continue
        if (!blocked[n] && !outside[n] && region[n] < 0) {
          region[n] = regionCount
          stack.push(n)
        }
      }
    }
    // Mindst 1 m² (100 celler à 0,01 m²) for at tælle som rum.
    if (cells.length >= 100) {
      const polygon = traceBoundary(cells, cols, minX, minY)
      if (polygon.length >= 4) polygons.push(polygon)
    }
    regionCount++
  }
  return polygons
}

// traceBoundary følger regionens yderkant (Moore-tracering) og forenkler
// kollineære punkter, så polygonen bliver pæn og rektilineær.
function traceBoundary(cells: number[], cols: number, minX: number, minY: number): Point[] {
  const inRegion = new Set(cells)
  // Startcelle: øverste venstre.
  let start = cells[0]
  for (const c of cells) {
    const cx = c % cols
    const cy = (c / cols) | 0
    const sx = start % cols
    const sy = (start / cols) | 0
    if (cy < sy || (cy === sy && cx < sx)) start = c
  }

  // Moore-tracering med 8 retninger, med uret fra "op".
  const dirs = [
    [0, -1], [1, -1], [1, 0], [1, 1], [0, 1], [-1, 1], [-1, 0], [-1, -1],
  ]
  const path: Point[] = []
  let current = start
  let dir = 6 // kom "fra venstre"
  const maxSteps = cells.length * 8
  for (let step = 0; step < maxSteps; step++) {
    const cx = current % cols
    const cy = (current / cols) | 0
    path.push({ x: cx * CELL + minX, y: cy * CELL + minY })
    let found = false
    for (let k = 0; k < 8; k++) {
      const d = (dir + 6 + k) % 8 // start søgning bagud-venstre for at følge kanten
      const nx = cx + dirs[d][0]
      const ny = cy + dirs[d][1]
      const n = ny * cols + nx
      if (inRegion.has(n)) {
        current = n
        dir = d
        found = true
        break
      }
    }
    if (!found) break // isoleret celle
    if (current === start && path.length > 2) break
  }
  return simplifyCollinear(path)
}

function simplifyCollinear(path: Point[]): Point[] {
  if (path.length < 3) return path
  const out: Point[] = []
  for (let i = 0; i < path.length; i++) {
    const prev = path[(i - 1 + path.length) % path.length]
    const cur = path[i]
    const next = path[(i + 1) % path.length]
    const cross = (cur.x - prev.x) * (next.y - prev.y) - (cur.y - prev.y) * (next.x - prev.x)
    if (Math.abs(cross) > 1e-6) out.push(cur)
  }
  return out.length >= 3 ? out : path
}

export function pointInPolygon(p: Point, polygon: Point[]): boolean {
  let inside = false
  for (let i = 0, j = polygon.length - 1; i < polygon.length; j = i++) {
    const a = polygon[i]
    const b = polygon[j]
    if (a.y > p.y !== b.y > p.y && p.x < ((b.x - a.x) * (p.y - a.y)) / (b.y - a.y) + a.x) {
      inside = !inside
    }
  }
  return inside
}

export function polygonCenter(polygon: Point[]): Point {
  const n = polygon.length || 1
  return {
    x: polygon.reduce((s, p) => s + p.x, 0) / n,
    y: polygon.reduce((s, p) => s + p.y, 0) / n,
  }
}

/** Nye rum = detekterede lommer hvis centrum ikke allerede ligger i et rum. */
export function newDetectedRooms(data: DrawingData): Point[][] {
  const existing: RoomShape[] = data.rooms
  return detectRooms(data).filter((polygon) => {
    const c = polygonCenter(polygon)
    return !existing.some((r) => pointInPolygon(c, r.polygon))
  })
}
