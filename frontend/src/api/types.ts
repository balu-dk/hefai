// API types mirroring the Go backend's JSON contracts.

export interface User {
  id: string
  email: string
  displayName: string
}

export interface AuthResult {
  token: string
  user: User
}

export type ProjectKind = 'new_build' | 'renovation' | 'extension' | 'other'
export type ProjectStatus = 'planning' | 'in_progress' | 'on_hold' | 'completed' | 'archived'

export interface Project {
  id: string
  name: string
  description: string
  kind: ProjectKind
  status: ProjectStatus
  address: string
  municipality: string
  cadastralId: string
  plotAreaM2: number | null
  latitude: number | null
  longitude: number | null
  utmX: number | null
  utmY: number | null
  createdBy: string
  createdAt: string
}

export interface AddressSuggestion {
  text: string
  id: string
}

export interface AddressDetails {
  address: string
  municipality: string
  cadastralId: string
  plotAreaM2: number | null
  lat: number
  lon: number
  utmX: number
  utmY: number
}

export interface LocalPlan {
  planId: string
  name: string
  status: string
  docLink: string
}

export type PhaseStatus = 'not_started' | 'in_progress' | 'completed'

export interface Phase {
  id: string
  projectId: string
  name: string
  description: string
  sortOrder: number
  status: PhaseStatus
  plannedStart: string | null
  plannedEnd: string | null
  actualStart: string | null
  actualEnd: string | null
}

export type TaskStatus = 'todo' | 'blocked' | 'in_progress' | 'done' | 'cancelled'

export interface Task {
  id: string
  projectId: string
  phaseId: string | null
  roomId: string | null
  title: string
  description: string
  status: TaskStatus
  responsibleUserId: string | null
  responsibleSupplierId: string | null
  plannedStart: string | null
  plannedEnd: string | null
  actualStart: string | null
  actualEnd: string | null
}

export interface BoardTask extends Task {
  dependsOn: string[]
  blocks: string[]
  actionable: boolean
  waitingFor: string[]
}

export interface BudgetItem {
  id: string
  projectId: string
  phaseId: string | null
  category: string
  description: string
  estimatedAmountOre: number
  currency: string
}

export interface Expense {
  id: string
  projectId: string
  budgetItemId: string | null
  phaseId: string | null
  supplierId: string | null
  description: string
  amountOre: number
  currency: string
  incurredOn: string
}

export interface BudgetGroupTotal {
  key: string
  phaseId: string | null
  estimatedOre: number
  spentOre: number
  remainingOre: number
}

export interface BudgetSummary {
  estimatedOre: number
  spentOre: number
  remainingOre: number
  byPhase: BudgetGroupTotal[]
  byCategory: BudgetGroupTotal[]
}

export type MaterialStatus = 'needed' | 'ordered' | 'delivered' | 'in_stock' | 'used'

export interface Material {
  id: string
  projectId: string
  phaseId: string | null
  taskId: string | null
  roomId: string | null
  supplierId: string | null
  name: string
  spec: string
  quantity: number
  unit: string
  unitPriceOre: number | null
  currency: string
  status: MaterialStatus
  notes: string
}

export interface ShoppingListGroup {
  supplierId: string | null
  supplierName: string
  materials: Material[]
  totalOre: number
}

export interface Supplier {
  id: string
  projectId: string
  companyName: string
  contactPerson: string
  trade: string
  phone: string
  email: string
  hourlyRateOre: number | null
  notes: string
}

export type RoomKind = 'room' | 'zone' | 'outdoor'

export interface Room {
  id: string
  projectId: string
  name: string
  kind: RoomKind
  description: string
  areaM2: number | null
}

export type DocumentKind =
  | 'architect_drawing' | 'construction_drawing' | 'receipt' | 'photo'
  | 'warranty' | 'datasheet' | 'permit' | 'correspondence' | 'generated' | 'other'

export interface Doc {
  id: string
  projectId: string
  kind: DocumentKind
  title: string
  description: string
  filename: string
  mimeType: string
  sizeBytes: number
  capturedAt: string | null
  tags: string[]
  createdAt: string
}

export type LinkTargetType =
  | 'phase' | 'task' | 'room' | 'expense' | 'material' | 'supplier'
  | 'case_file' | 'structural_element'

export interface DocumentLink {
  id: string
  documentId: string
  targetType: LinkTargetType
  targetId: string
}

export type CaseType = 'unknown' | 'notification' | 'building_permit'
export type CaseStatus =
  | 'draft' | 'ready_for_submission' | 'submitted' | 'awaiting_response'
  | 'questions_from_municipality' | 'approved' | 'rejected' | 'closed'

export interface CaseFile {
  id: string
  projectId: string
  title: string
  description: string
  caseType: CaseType
  status: CaseStatus
  municipalCaseNumber: string
  submittedAt: string | null
  decidedAt: string | null
}

export interface CaseEvent {
  id: string
  caseFileId: string
  eventType: 'status_change' | 'correspondence' | 'note' | 'submission'
  direction: 'incoming' | 'outgoing' | 'internal' | null
  occurredAt: string
  summary: string
  body: string
  documentId: string | null
}

// --- drawing canvas model (mirrors Go domain.DrawingData) ---------------------

export interface Point {
  x: number
  y: number
}

export interface Wall {
  id: string
  from: Point
  to: Point
  thicknessMm: number
  isLoadBearing: boolean
}

export interface RoomShape {
  name: string
  polygon: Point[]
}

export interface Opening {
  wallId: string
  type: 'door' | 'window'
  offsetMm: number
  widthMm: number
  heightMm: number
}

export interface Plot {
  boundary: Point[]
  offset: Point
  rotationDeg: number
}

export interface Tree {
  position: Point
  heightMm: number
  crownDiameterMm: number
}

export interface GeoAnchor {
  lat: number
  lon: number
  sizeM: number
}

export interface DrawingData {
  walls: Wall[]
  rooms: RoomShape[]
  openings: Opening[]
  plot?: Plot | null
  trees?: Tree[]
  wallHeightMm?: number
  roofAngleDeg?: number
  geo?: GeoAnchor | null
}

export type DrawingKind = 'site_plan' | 'floor_plan' | 'elevation' | 'section' | 'detail' | 'other'

export interface Drawing {
  id: string
  projectId: string
  caseFileId: string | null
  kind: DrawingKind
  title: string
}

export interface DrawingVersion {
  id: string
  drawingId: string
  versionNo: number
  data: DrawingData
  scale: string
  note: string
  createdAt: string
}

export type ComplianceStatus = 'not_checked' | 'ok' | 'attention' | 'needs_confirmation' | 'confirmed'

export interface ComplianceItem {
  id: string
  caseFileId: string
  category: string
  requirement: string
  expectedValue: string
  actualValue: string
  status: ComplianceStatus
  sourceChunkId: string | null
  sourceRef: string
  note: string
}

export type SourceKind = 'br18' | 'eurocode' | 'local_plan' | 'municipal_guidance' | 'other'

export interface SourceDocument {
  id: string
  projectId: string | null
  title: string
  kind: SourceKind
  versionLabel: string
  url: string
  status: 'processing' | 'ready' | 'failed'
  chunkCount: number
}

export interface SourceHit {
  chunkId: string
  sourceId: string
  sourceTitle: string
  sourceKind: SourceKind
  sectionRef: string
  content: string
  rank: number
}

export interface AskResult {
  answer: string
  answered: boolean
  provider: string
  citations: SourceHit[]
  notice?: string
}

export type GeneratedKind =
  | 'site_plan' | 'floor_plan' | 'elevation' | 'area_statement'
  | 'project_description' | 'application_summary' | 'structural_package' | 'other'

export interface GeneratedDocument {
  id: string
  projectId: string
  caseFileId: string | null
  kind: GeneratedKind
  status: 'draft' | 'final'
  versionNo: number
  documentId: string | null
  createdAt: string
}

// --- structural ---------------------------------------------------------------

export type ElementType = 'beam' | 'column' | 'wall' | 'foundation' | 'roof' | 'slab' | 'other'
export type StructuralMaterialType = 'timber' | 'steel' | 'concrete' | 'masonry' | 'other'

export interface StructuralElement {
  id: string
  projectId: string
  roomId: string | null
  drawingId: string | null
  elementType: ElementType
  name: string
  isLoadBearing: boolean
  material: StructuralMaterialType
  materialSpec: string
  geometry: Record<string, unknown>
  notes: string
}

export type LoadType = 'dead' | 'live' | 'snow' | 'wind' | 'point' | 'line' | 'other'
export type LoadStatus = 'assumed' | 'engineer_confirmed' | 'engineer_changed'

export interface StructuralLoad {
  id: string
  projectId: string
  structuralElementId: string | null
  loadType: LoadType
  value: number
  unit: string
  standardReference: string
  derivation: Record<string, unknown>
  status: LoadStatus
  notes: string
}

export type EstimateStatus = 'advisory' | 'verified' | 'superseded' | 'rejected'

export interface Assumption {
  text: string
  reference: string
}

export interface CalculationEstimate {
  id: string
  projectId: string
  structuralElementId: string | null
  method: string
  methodVersion: string
  standardReference: string
  inputs: Record<string, unknown>
  assumptions: Assumption[]
  results: { results: Record<string, unknown>; notice: string }
  status: EstimateStatus
  notes: string
  createdAt: string
}

export interface CalcMethod {
  name: string
  description: string
  standardReference: string
  inputsExample: Record<string, unknown>
}

export interface StructuralPackage {
  id: string
  projectId: string
  versionNo: number
  title: string
  documentId: string | null
  status: 'draft' | 'sent' | 'reviewed'
  sentAt: string | null
  createdAt: string
}

export type ReviewVerdict = 'approved' | 'changed' | 'rejected' | 'comment'

export interface EngineerReviewItem {
  id: string
  structuralElementId: string | null
  loadId: string | null
  calculationEstimateId: string | null
  drawingId: string | null
  verdict: ReviewVerdict
  comment: string
  correctedValues: Record<string, unknown>
}

// --- AI-projektstart ------------------------------------------------------------

export interface Interview {
  goal: string
  propertyType: string
  sizeM2: number
  rooms: string[]
  features: string[]
  budgetOre: number
  selfBuild: string
  timeline: string
  notes: string
}

export interface BlueprintTask {
  title: string
  description: string
  phase: string
  dependsOn: number[]
}

export interface Blueprint {
  projectDescription: string
  caseDescription: string
  needsBuildingCase: boolean
  rooms: { name: string; kind: string; areaM2: number | null }[]
  tasks: BlueprintTask[]
  budgetItems: { description: string; category: string; phase: string; estimatedAmountOre: number }[]
  materials: { name: string; spec: string; quantity: number; unit: string; phase: string }[]
  notes: string
}

export interface BlueprintResult {
  blueprint: Blueprint
  source: 'llm' | 'template'
  provider: string
  notice?: string
}

export interface ApplyResult {
  roomsCreated: number
  tasksCreated: number
  dependenciesLinked: number
  budgetItemsCreated: number
  materialsCreated: number
  caseFileId: string | null
}

export interface EngineerReview {
  id: string
  structuralPackageId: string
  reviewerName: string
  reviewerCompany: string
  reviewerCredentials: string
  receivedAt: string
  overallStatus: 'approved' | 'approved_with_changes' | 'rejected' | 'partial'
  summary: string
  items: EngineerReviewItem[] | null
}
