import { useEffect, useRef } from 'react'
import * as THREE from 'three'
import { OrbitControls } from 'three/addons/controls/OrbitControls.js'
import { RoomEnvironment } from 'three/addons/environments/RoomEnvironment.js'
import type { DrawingData, Point } from '../api/types'

// Renders the measured 2D model as a walk-around 3D scene. Units: the model
// is millimetres; the scene is metres (mm × 0.001). The building (walls,
// rooms, roof) is placed on the plot via plot.offset/rotationDeg — the same
// transform the generated site plan uses.

const MM = 0.001

export default function Drawing3DView({ data, orthoURL }: { data: DrawingData; orthoURL?: string | null }) {
  const mountRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const mount = mountRef.current!
    const width = mount.clientWidth
    const height = Math.max(mount.clientHeight, 420)

    const scene = new THREE.Scene()
    // Blød himmelgradient + dis i horisonten giver dybde.
    scene.background = new THREE.Color(0xcfe0ee)
    scene.fog = new THREE.Fog(0xcfe0ee, 60, 220)

    const renderer = new THREE.WebGLRenderer({ antialias: true })
    renderer.setSize(width, height)
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
    renderer.shadowMap.enabled = true
    renderer.shadowMap.type = THREE.PCFSoftShadowMap
    renderer.toneMapping = THREE.ACESFilmicToneMapping
    renderer.toneMappingExposure = 1.05
    mount.appendChild(renderer.domElement)

    // Billedbaseret miljølys giver bløde reflekser i alle materialer.
    const pmrem = new THREE.PMREMGenerator(renderer)
    scene.environment = pmrem.fromScene(new RoomEnvironment(), 0.04).texture

    // Lav eftermiddagssol med bløde skygger.
    const sun = new THREE.DirectionalLight(0xfff2dd, 2.6)
    sun.position.set(35, 30, 18)
    sun.castShadow = true
    sun.shadow.mapSize.set(2048, 2048)
    sun.shadow.camera.left = -45
    sun.shadow.camera.right = 45
    sun.shadow.camera.top = 45
    sun.shadow.camera.bottom = -45
    sun.shadow.radius = 6
    sun.shadow.bias = -0.0004
    scene.add(sun)
    scene.add(new THREE.HemisphereLight(0xbcd6ee, 0x7d9468, 0.45))

    const wallHeight = (data.wallHeightMm || 2500) * MM

    // --- Grund -------------------------------------------------------------
    const groundGroup = new THREE.Group()
    scene.add(groundGroup)

    if (data.plot && data.plot.boundary.length >= 3) {
      const shape = new THREE.Shape(data.plot.boundary.map((p) => new THREE.Vector2(p.x * MM, p.y * MM)))
      const plot = new THREE.Mesh(
        new THREE.ShapeGeometry(shape),
        new THREE.MeshStandardMaterial({ color: 0x6f9e5c, roughness: 0.95 }),
      )
      plot.rotation.x = -Math.PI / 2
      plot.receiveShadow = true
      groundGroup.add(plot)
    }
    // Stor mat baggrundsflade under alt, tonet mod disen.
    const base = new THREE.Mesh(
      new THREE.CircleGeometry(220, 64),
      new THREE.MeshStandardMaterial({ color: 0x8aa878, roughness: 1 }),
    )
    base.rotation.x = -Math.PI / 2
    base.position.y = -0.02
    base.receiveShadow = true
    groundGroup.add(base)

    // Luftfoto draperet på jorden, centreret på grunden — samme udsnit som
    // 2D-baggrunden, så tegning og virkelighed flugter.
    if (orthoURL && data.geo) {
      const sizeM = data.geo.sizeM || 150
      const center = data.plot && data.plot.boundary.length >= 3
        ? polygonCentroid(data.plot.boundary)
        : { x: 0, y: 0 }
      new THREE.TextureLoader().load(orthoURL, (texture) => {
        texture.colorSpace = THREE.SRGBColorSpace
        const photo = new THREE.Mesh(
          new THREE.PlaneGeometry(sizeM, sizeM),
          new THREE.MeshStandardMaterial({ map: texture, roughness: 1 }),
        )
        photo.rotation.x = -Math.PI / 2
        photo.position.set(center.x * MM, 0.01, center.y * MM)
        photo.receiveShadow = true
        groundGroup.add(photo)
      })
    }

    // --- Bygning (transformeret til grunden) ---------------------------------
    const building = new THREE.Group()
    const rot = ((data.plot?.rotationDeg ?? 0) * Math.PI) / 180
    building.rotation.y = -rot // 2D-rotation (x,y) → scene (x,z)
    building.position.set((data.plot?.offset.x ?? 0) * MM, 0, (data.plot?.offset.y ?? 0) * MM)
    scene.add(building)

    const wallMat = new THREE.MeshStandardMaterial({ color: 0xf5f1e8, roughness: 0.85 })
    const innerWallMat = new THREE.MeshStandardMaterial({ color: 0xe9e3d5, roughness: 0.9 })
    const edgeMat = new THREE.LineBasicMaterial({ color: 0x4a4238, transparent: true, opacity: 0.35 })
    const glassMat = new THREE.MeshPhysicalMaterial({
      color: 0xbfd9ee, roughness: 0.08, metalness: 0, transmission: 0.7,
      transparent: true, opacity: 0.9,
    })
    const frameMat = new THREE.MeshStandardMaterial({ color: 0x3d3a34, roughness: 0.5 })
    const doorMat = new THREE.MeshStandardMaterial({ color: 0x6f4a26, roughness: 0.55 })

    for (const w of data.walls) {
      const dx = (w.to.x - w.from.x) * MM
      const dz = (w.to.y - w.from.y) * MM
      const len = Math.hypot(dx, dz)
      if (len === 0) continue
      const geometry = new THREE.BoxGeometry(len, wallHeight, Math.max(w.thicknessMm * MM, 0.05))
      const mesh = new THREE.Mesh(geometry, w.isLoadBearing ? wallMat : innerWallMat)
      mesh.position.set(((w.from.x + w.to.x) / 2) * MM, wallHeight / 2, ((w.from.y + w.to.y) / 2) * MM)
      mesh.rotation.y = -Math.atan2(dz, dx)
      mesh.castShadow = true
      mesh.receiveShadow = true
      // Diskrete kantlinjer giver det rene "arkitektmodel"-look.
      const edges = new THREE.LineSegments(new THREE.EdgesGeometry(geometry), edgeMat)
      mesh.add(edges)
      building.add(mesh)

      // Åbninger: glas i mørk ramme hhv. trædør.
      for (const o of data.openings.filter((o) => o.wallId === w.id)) {
        const isWindow = o.type === 'window'
        const openingH = o.heightMm * MM
        const sill = isWindow ? 0.9 : 0
        const along = (o.offsetMm + o.widthMm / 2) * MM
        const t = along / len
        const px = (w.from.x * MM) + dx * t
        const pz = (w.from.y * MM) + dz * t
        const angle = -Math.atan2(dz, dx)

        const group = new THREE.Group()
        const frame = new THREE.Mesh(
          new THREE.BoxGeometry(o.widthMm * MM, openingH, w.thicknessMm * MM + 0.05),
          frameMat,
        )
        group.add(frame)
        const fill = new THREE.Mesh(
          new THREE.BoxGeometry(o.widthMm * MM - 0.12, openingH - 0.12, w.thicknessMm * MM + 0.08),
          isWindow ? glassMat : doorMat,
        )
        group.add(fill)
        group.position.set(px, sill + openingH / 2, pz)
        group.rotation.y = angle
        building.add(group)
      }
    }

    // Gulve fra rum-polygoner.
    for (const room of data.rooms) {
      if (room.polygon.length < 3) continue
      const shape = new THREE.Shape(room.polygon.map((p) => new THREE.Vector2(p.x * MM, p.y * MM)))
      const floor = new THREE.Mesh(
        new THREE.ShapeGeometry(shape),
        new THREE.MeshStandardMaterial({ color: 0xcbb489, roughness: 0.65 }),
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
      const roofMat = new THREE.MeshStandardMaterial({ color: 0x5c4436, roughness: 0.8 })

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
    const trunkMat = new THREE.MeshStandardMaterial({ color: 0x5e4530, roughness: 0.9 })
    data.trees?.forEach((t, treeIndex) => {
      const h = t.heightMm * MM
      const crownR = (t.crownDiameterMm / 2) * MM
      const trunkH = Math.max(h - crownR * 1.4, h * 0.3)
      const tree = new THREE.Group()
      const trunk = new THREE.Mesh(new THREE.CylinderGeometry(0.1, 0.2, trunkH, 8), trunkMat)
      trunk.position.y = trunkH / 2
      trunk.castShadow = true
      tree.add(trunk)
      // Løvet som 3 let forskudte kugler i grønne nuancer — mere organisk
      // end én perfekt kugle. Deterministisk pr. træ-indeks.
      const greens = [0x3c7237, 0x4c8a42, 0x35652f]
      for (let k = 0; k < 3; k++) {
        const offset = ((treeIndex * 7 + k * 3) % 5) / 5 - 0.5
        const blob = new THREE.Mesh(
          new THREE.SphereGeometry(crownR * (0.75 + k * 0.12), 12, 10),
          new THREE.MeshStandardMaterial({ color: greens[k], roughness: 0.95 }),
        )
        blob.position.set(offset * crownR * 0.8, trunkH + crownR * (0.5 + k * 0.28), offset * crownR * 0.5)
        blob.castShadow = true
        tree.add(blob)
      }
      tree.position.set(t.position.x * MM, 0, t.position.y * MM)
      scene.add(tree)
    })

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
      pmrem.dispose()
      renderer.dispose()
      mount.removeChild(renderer.domElement)
    }
  }, [data, orthoURL])

  return <div ref={mountRef} style={{ width: '100%', height: '58vh', borderRadius: 8, overflow: 'hidden' }} />
}

function polygonCentroid(polygon: Point[]): Point {
  const n = polygon.length || 1
  return {
    x: polygon.reduce((s, p) => s + p.x, 0) / n,
    y: polygon.reduce((s, p) => s + p.y, 0) / n,
  }
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
