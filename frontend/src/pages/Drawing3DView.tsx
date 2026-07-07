import { useEffect, useRef } from 'react'
import * as THREE from 'three'
import { OrbitControls } from 'three/addons/controls/OrbitControls.js'
import type { DrawingData, Point } from '../api/types'

// Renders the measured 2D model as a walk-around 3D scene. Units: the model
// is millimetres; the scene is metres (mm × 0.001). The building (walls,
// rooms, roof) is placed on the plot via plot.offset/rotationDeg — the same
// transform the generated site plan uses.

const MM = 0.001

export default function Drawing3DView({ data }: { data: DrawingData }) {
  const mountRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const mount = mountRef.current!
    const width = mount.clientWidth
    const height = Math.max(mount.clientHeight, 420)

    const scene = new THREE.Scene()
    scene.background = new THREE.Color(0xdcebf5)

    const renderer = new THREE.WebGLRenderer({ antialias: true })
    renderer.setSize(width, height)
    renderer.setPixelRatio(window.devicePixelRatio)
    renderer.shadowMap.enabled = true
    mount.appendChild(renderer.domElement)

    // Lys: himmel + sol med skygger.
    scene.add(new THREE.HemisphereLight(0xffffff, 0x8899aa, 0.9))
    const sun = new THREE.DirectionalLight(0xfff4e0, 1.6)
    sun.position.set(30, 40, 20)
    sun.castShadow = true
    sun.shadow.mapSize.set(2048, 2048)
    sun.shadow.camera.left = -40
    sun.shadow.camera.right = 40
    sun.shadow.camera.top = 40
    sun.shadow.camera.bottom = -40
    scene.add(sun)

    const wallHeight = (data.wallHeightMm || 2500) * MM

    // --- Grund -------------------------------------------------------------
    const groundGroup = new THREE.Group()
    scene.add(groundGroup)

    if (data.plot && data.plot.boundary.length >= 3) {
      const shape = new THREE.Shape(data.plot.boundary.map((p) => new THREE.Vector2(p.x * MM, p.y * MM)))
      const plotGeo = new THREE.ShapeGeometry(shape)
      const plot = new THREE.Mesh(plotGeo, new THREE.MeshLambertMaterial({ color: 0x7fae6e }))
      plot.rotation.x = -Math.PI / 2
      plot.receiveShadow = true
      groundGroup.add(plot)
    }
    // Stor mat baggrundsflade under alt.
    const base = new THREE.Mesh(
      new THREE.CircleGeometry(200, 48),
      new THREE.MeshLambertMaterial({ color: 0x9bb78c }),
    )
    base.rotation.x = -Math.PI / 2
    base.position.y = -0.02
    base.receiveShadow = true
    groundGroup.add(base)

    // --- Bygning (transformeret til grunden) ---------------------------------
    const building = new THREE.Group()
    const rot = ((data.plot?.rotationDeg ?? 0) * Math.PI) / 180
    building.rotation.y = -rot // 2D-rotation (x,y) → scene (x,z)
    building.position.set((data.plot?.offset.x ?? 0) * MM, 0, (data.plot?.offset.y ?? 0) * MM)
    scene.add(building)

    const wallMat = new THREE.MeshLambertMaterial({ color: 0xf3efe6 })
    const innerWallMat = new THREE.MeshLambertMaterial({ color: 0xe6e0d2 })

    for (const w of data.walls) {
      const dx = (w.to.x - w.from.x) * MM
      const dz = (w.to.y - w.from.y) * MM
      const len = Math.hypot(dx, dz)
      if (len === 0) continue
      const mesh = new THREE.Mesh(
        new THREE.BoxGeometry(len, wallHeight, Math.max(w.thicknessMm * MM, 0.05)),
        w.isLoadBearing ? wallMat : innerWallMat,
      )
      mesh.position.set(((w.from.x + w.to.x) / 2) * MM, wallHeight / 2, ((w.from.y + w.to.y) / 2) * MM)
      mesh.rotation.y = -Math.atan2(dz, dx)
      mesh.castShadow = true
      mesh.receiveShadow = true
      building.add(mesh)

      // Åbninger: indfarvede felter en anelse uden på væggen.
      for (const o of data.openings.filter((o) => o.wallId === w.id)) {
        const isWindow = o.type === 'window'
        const openingH = o.heightMm * MM
        const sill = isWindow ? 0.9 : 0
        const along = (o.offsetMm + o.widthMm / 2) * MM
        const box = new THREE.Mesh(
          new THREE.BoxGeometry(o.widthMm * MM, openingH, w.thicknessMm * MM + 0.04),
          new THREE.MeshLambertMaterial({
            color: isWindow ? 0x9cc7e8 : 0x8a5a2e,
            transparent: isWindow,
            opacity: isWindow ? 0.75 : 1,
          }),
        )
        const t = along / len
        box.position.set(
          (w.from.x * MM) + dx * t,
          sill + openingH / 2,
          (w.from.y * MM) + dz * t,
        )
        box.rotation.y = -Math.atan2(dz, dx)
        building.add(box)
      }
    }

    // Gulve fra rum-polygoner.
    for (const room of data.rooms) {
      if (room.polygon.length < 3) continue
      const shape = new THREE.Shape(room.polygon.map((p) => new THREE.Vector2(p.x * MM, p.y * MM)))
      const floor = new THREE.Mesh(
        new THREE.ShapeGeometry(shape),
        new THREE.MeshLambertMaterial({ color: 0xd9c9a8 }),
      )
      floor.rotation.x = -Math.PI / 2
      floor.position.y = 0.02
      floor.receiveShadow = true
      building.add(floor)
    }

    // Tag: saddeltag over bygningens omrids (bbox), ryg langs den lange side.
    if (data.walls.length > 0) {
      const xs = data.walls.flatMap((w) => [w.from.x, w.to.x])
      const ys = data.walls.flatMap((w) => [w.from.y, w.to.y])
      const minX = Math.min(...xs) * MM
      const maxX = Math.max(...xs) * MM
      const minY = Math.min(...ys) * MM
      const maxY = Math.max(...ys) * MM
      const spanX = maxX - minX
      const spanY = maxY - minY
      const angle = ((data.roofAngleDeg ?? 0) * Math.PI) / 180
      const overhang = 0.4
      const roofMat = new THREE.MeshLambertMaterial({ color: 0x745441 })

      if (angle > 0.01 && spanX > 0 && spanY > 0) {
        const alongX = spanX >= spanY
        const width = (alongX ? spanY : spanX) + overhang * 2
        const length = (alongX ? spanX : spanY) + overhang * 2
        const ridge = (width / 2) * Math.tan(angle)
        // Trekantet prisme via Shape + Extrude.
        const tri = new THREE.Shape([
          new THREE.Vector2(-width / 2, 0),
          new THREE.Vector2(width / 2, 0),
          new THREE.Vector2(0, ridge),
        ])
        const prism = new THREE.Mesh(
          new THREE.ExtrudeGeometry(tri, { depth: length, bevelEnabled: false }),
          roofMat,
        )
        prism.castShadow = true
        // Extrude sker langs objektets +Z; rotér så ryggen følger den lange
        // akse, og skub prismet halvdelen af længden tilbage langs sin egen
        // udstrækningsakse så det centreres over bygningen.
        prism.rotation.y = alongX ? Math.PI / 2 : 0
        prism.position.set((minX + maxX) / 2, wallHeight, (minY + maxY) / 2)
        prism.translateZ(-length / 2)
        building.add(prism)
      } else if (spanX > 0 && spanY > 0) {
        const flat = new THREE.Mesh(
          new THREE.BoxGeometry(spanX + overhang, 0.15, spanY + overhang),
          roofMat,
        )
        flat.position.set((minX + maxX) / 2, wallHeight + 0.075, (minY + maxY) / 2)
        flat.castShadow = true
        building.add(flat)
      }
    }

    // --- Træer (i grund-koordinater) -----------------------------------------
    for (const t of data.trees ?? []) {
      const h = t.heightMm * MM
      const crownR = (t.crownDiameterMm / 2) * MM
      const trunkH = Math.max(h - crownR * 1.4, h * 0.3)
      const tree = new THREE.Group()
      const trunk = new THREE.Mesh(
        new THREE.CylinderGeometry(0.12, 0.18, trunkH, 8),
        new THREE.MeshLambertMaterial({ color: 0x6d4c33 }),
      )
      trunk.position.y = trunkH / 2
      trunk.castShadow = true
      const crown = new THREE.Mesh(
        new THREE.SphereGeometry(crownR, 12, 10),
        new THREE.MeshLambertMaterial({ color: 0x3f7d3a }),
      )
      crown.position.y = trunkH + crownR * 0.7
      crown.castShadow = true
      tree.add(trunk, crown)
      tree.position.set(t.position.x * MM, 0, t.position.y * MM)
      scene.add(tree)
    }

    // --- Kamera & controls -------------------------------------------------------
    const center = sceneCenter(data)
    const camera = new THREE.PerspectiveCamera(50, width / height, 0.1, 500)
    const dist = Math.max(sceneRadius(data) * 2.2, 12)
    camera.position.set(center.x * MM + dist, dist * 0.7, center.y * MM + dist)

    const controls = new OrbitControls(camera, renderer.domElement)
    controls.target.set(center.x * MM, 1, center.y * MM)
    controls.maxPolarAngle = Math.PI / 2 - 0.03 // ikke under jorden
    controls.enableDamping = true

    let disposed = false
    function animate() {
      if (disposed) return
      requestAnimationFrame(animate)
      controls.update()
      renderer.render(scene, camera)
    }
    animate()

    function onResize() {
      const w = mount.clientWidth
      camera.aspect = w / height
      camera.updateProjectionMatrix()
      renderer.setSize(w, height)
    }
    window.addEventListener('resize', onResize)

    return () => {
      disposed = true
      window.removeEventListener('resize', onResize)
      controls.dispose()
      renderer.dispose()
      mount.removeChild(renderer.domElement)
    }
  }, [data])

  return <div ref={mountRef} style={{ width: '100%', height: '58vh', borderRadius: 8, overflow: 'hidden' }} />
}

// Kameraet fokuserer på bygningen (transformeret til grunden) — findes der
// ingen vægge, bruges grundens skel i stedet.
function focusPoints(data: DrawingData): Point[] {
  if (data.walls.length > 0) {
    const rot = ((data.plot?.rotationDeg ?? 0) * Math.PI) / 180
    const cos = Math.cos(rot)
    const sin = Math.sin(rot)
    const ox = data.plot?.offset.x ?? 0
    const oy = data.plot?.offset.y ?? 0
    return data.walls.flatMap((w) => [w.from, w.to]).map((p) => ({
      x: ox + p.x * cos - p.y * sin,
      y: oy + p.x * sin + p.y * cos,
    }))
  }
  return data.plot?.boundary ?? []
}

function sceneCenter(data: DrawingData): Point {
  const pts = focusPoints(data)
  if (pts.length === 0) return { x: 0, y: 0 }
  return {
    x: pts.reduce((s, p) => s + p.x, 0) / pts.length,
    y: pts.reduce((s, p) => s + p.y, 0) / pts.length,
  }
}

function sceneRadius(data: DrawingData): number {
  const c = sceneCenter(data)
  let r = 0
  for (const p of focusPoints(data)) {
    r = Math.max(r, Math.hypot(p.x - c.x, p.y - c.y))
  }
  return r * MM
}
