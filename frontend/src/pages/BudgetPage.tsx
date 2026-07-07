import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { api, kr, parseKr } from '../api/client'
import type { BudgetItem, BudgetSummary, Expense, Phase, Supplier } from '../api/types'
import { confirmDelete, Empty, ErrorText, Modal, useLoad } from '../components'

export default function BudgetPage() {
  const { projectId } = useParams()
  const { data: summary, reload: reloadSummary } = useLoad(
    () => api.get<BudgetSummary>(`/projects/${projectId}/budget/summary`), [projectId])
  const { data: items, error, reload: reloadItems } = useLoad(
    () => api.get<BudgetItem[]>(`/projects/${projectId}/budget-items`), [projectId])
  const { data: expenses, reload: reloadExpenses } = useLoad(
    () => api.get<Expense[]>(`/projects/${projectId}/expenses`), [projectId])
  const { data: phases } = useLoad(() => api.get<Phase[]>(`/projects/${projectId}/phases`), [projectId])
  const { data: suppliers } = useLoad(() => api.get<Supplier[]>(`/projects/${projectId}/suppliers`), [projectId])

  const [editItem, setEditItem] = useState<BudgetItem | null>(null)
  const [newItem, setNewItem] = useState(false)
  const [editExpense, setEditExpense] = useState<Expense | null>(null)
  const [newExpense, setNewExpense] = useState(false)

  const reloadAll = () => (reloadSummary(), reloadItems(), reloadExpenses())
  const phaseName = (id: string | null) => phases?.find((p) => p.id === id)?.name ?? '–'

  return (
    <>
      <h1>Budget & økonomi</h1>
      <p className="page-sub">Estimeret budget mod faktiske udgifter.</p>
      <ErrorText error={error} />

      {summary && (
        <div className="grid cols-3">
          <div className="card stat">
            <div className="label">Budgetteret</div>
            <div className="value">{kr(summary.estimatedOre)}</div>
          </div>
          <div className="card stat">
            <div className="label">Brugt</div>
            <div className="value">{kr(summary.spentOre)}</div>
          </div>
          <div className="card stat">
            <div className="label">Tilbage</div>
            <div className="value" style={{ color: summary.remainingOre < 0 ? 'var(--red)' : 'var(--green)' }}>
              {kr(summary.remainingOre)}
            </div>
          </div>
        </div>
      )}

      <div className="grid cols-2">
        <div className="card">
          <h3>Pr. fase</h3>
          <GroupTable groups={summary?.byPhase ?? []} />
        </div>
        <div className="card">
          <h3>Pr. kategori</h3>
          <GroupTable groups={summary?.byCategory ?? []} />
        </div>
      </div>

      <h2>Budgetposter</h2>
      <div className="card">
        <table className="tbl">
          <thead>
            <tr><th>Beskrivelse</th><th>Kategori</th><th>Fase</th><th className="num">Estimat</th><th></th></tr>
          </thead>
          <tbody>
            {items?.map((b) => (
              <tr key={b.id}>
                <td>{b.description}</td>
                <td>{b.category || '–'}</td>
                <td>{phaseName(b.phaseId)}</td>
                <td className="num">{kr(b.estimatedAmountOre)}</td>
                <td className="row-actions">
                  <button className="btn small secondary" onClick={() => setEditItem(b)}>Redigér</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {items?.length === 0 && <Empty>Ingen budgetposter endnu.</Empty>}
      </div>
      <button className="btn" onClick={() => setNewItem(true)}>+ Budgetpost</button>

      <h2>Udgifter</h2>
      <div className="card">
        <table className="tbl">
          <thead>
            <tr><th>Dato</th><th>Beskrivelse</th><th>Budgetpost</th><th>Leverandør</th><th className="num">Beløb</th><th></th></tr>
          </thead>
          <tbody>
            {expenses?.map((e) => (
              <tr key={e.id}>
                <td>{e.incurredOn.slice(0, 10)}</td>
                <td>{e.description}</td>
                <td>{items?.find((b) => b.id === e.budgetItemId)?.description ?? '–'}</td>
                <td>{suppliers?.find((s) => s.id === e.supplierId)?.companyName ?? '–'}</td>
                <td className="num">{kr(e.amountOre)}</td>
                <td className="row-actions">
                  <button className="btn small secondary" onClick={() => setEditExpense(e)}>Redigér</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {expenses?.length === 0 && <Empty>Ingen udgifter registreret.</Empty>}
      </div>
      <button className="btn" onClick={() => setNewExpense(true)}>+ Udgift</button>

      {(editItem || newItem) && (
        <BudgetItemModal
          item={editItem} projectId={projectId!} phases={phases ?? []}
          onDone={() => (setEditItem(null), setNewItem(false), reloadAll())}
          onClose={() => (setEditItem(null), setNewItem(false))}
        />
      )}
      {(editExpense || newExpense) && (
        <ExpenseModal
          expense={editExpense} projectId={projectId!} phases={phases ?? []}
          items={items ?? []} suppliers={suppliers ?? []}
          onDone={() => (setEditExpense(null), setNewExpense(false), reloadAll())}
          onClose={() => (setEditExpense(null), setNewExpense(false))}
        />
      )}
    </>
  )
}

function GroupTable({ groups }: { groups: { key: string; estimatedOre: number; spentOre: number; remainingOre: number }[] }) {
  if (groups.length === 0) return <Empty>Ingen data.</Empty>
  return (
    <table className="tbl">
      <thead>
        <tr><th></th><th className="num">Estimat</th><th className="num">Brugt</th><th className="num">Tilbage</th></tr>
      </thead>
      <tbody>
        {groups.map((g) => (
          <tr key={g.key}>
            <td>{g.key}</td>
            <td className="num">{kr(g.estimatedOre)}</td>
            <td className="num">{kr(g.spentOre)}</td>
            <td className="num" style={{ color: g.remainingOre < 0 ? 'var(--red)' : undefined }}>{kr(g.remainingOre)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

const ZERO = '00000000-0000-0000-0000-000000000000'

function BudgetItemModal({ item, projectId, phases, onDone, onClose }: {
  item: BudgetItem | null
  projectId: string
  phases: Phase[]
  onDone: () => void
  onClose: () => void
}) {
  const [description, setDescription] = useState(item?.description ?? '')
  const [category, setCategory] = useState(item?.category ?? '')
  const [phaseId, setPhaseId] = useState(item?.phaseId ?? '')
  const [amount, setAmount] = useState(item ? (item.estimatedAmountOre / 100).toFixed(2).replace('.', ',') : '')
  const [error, setError] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    const ore = parseKr(amount)
    if (ore === null) return setError('Ugyldigt beløb — brug fx 12.500,00')
    const body = { description, category, phaseId: phaseId || ZERO, estimatedAmountOre: ore }
    try {
      if (item) await api.patch(`/budget-items/${item.id}`, body)
      else await api.post(`/projects/${projectId}/budget-items`, body)
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function remove() {
    if (!item || !confirmDelete('budgetposten')) return
    await api.del(`/budget-items/${item.id}`)
    onDone()
  }

  return (
    <Modal title={item ? 'Redigér budgetpost' : 'Ny budgetpost'} onClose={onClose}>
      <form onSubmit={submit}>
        <div className="field" style={{ marginBottom: 10 }}>
          <label>Beskrivelse *</label>
          <input value={description} onChange={(e) => setDescription(e.target.value)} required autoFocus={!item} />
        </div>
        <div className="form-row">
          <div className="field">
            <label>Kategori</label>
            <input value={category} onChange={(e) => setCategory(e.target.value)} placeholder="Materialer, Håndværker…" />
          </div>
          <div className="field">
            <label>Fase</label>
            <select value={phaseId} onChange={(e) => setPhaseId(e.target.value)}>
              <option value="">(ingen)</option>
              {phases.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
            </select>
          </div>
          <div className="field">
            <label>Estimat (kr.) *</label>
            <input value={amount} onChange={(e) => setAmount(e.target.value)} required placeholder="150.000,00" />
          </div>
        </div>
        <ErrorText error={error} />
        <div className="form-row" style={{ justifyContent: 'space-between', marginTop: 12 }}>
          <div>{item && <button type="button" className="btn danger" onClick={remove}>Slet</button>}</div>
          <div className="row-actions">
            <button type="button" className="btn secondary" onClick={onClose}>Annullér</button>
            <button className="btn">Gem</button>
          </div>
        </div>
      </form>
    </Modal>
  )
}

function ExpenseModal({ expense, projectId, phases, items, suppliers, onDone, onClose }: {
  expense: Expense | null
  projectId: string
  phases: Phase[]
  items: BudgetItem[]
  suppliers: Supplier[]
  onDone: () => void
  onClose: () => void
}) {
  const [description, setDescription] = useState(expense?.description ?? '')
  const [amount, setAmount] = useState(expense ? (expense.amountOre / 100).toFixed(2).replace('.', ',') : '')
  const [date, setDate] = useState(expense?.incurredOn.slice(0, 10) ?? new Date().toISOString().slice(0, 10))
  const [budgetItemId, setBudgetItemId] = useState(expense?.budgetItemId ?? '')
  const [phaseId, setPhaseId] = useState(expense?.phaseId ?? '')
  const [supplierId, setSupplierId] = useState(expense?.supplierId ?? '')
  const [error, setError] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    const ore = parseKr(amount)
    if (ore === null) return setError('Ugyldigt beløb')
    const body = {
      description,
      amountOre: ore,
      incurredOn: `${date}T00:00:00Z`,
      budgetItemId: budgetItemId || ZERO,
      phaseId: phaseId || ZERO,
      supplierId: supplierId || ZERO,
    }
    try {
      if (expense) await api.patch(`/expenses/${expense.id}`, body)
      else await api.post(`/projects/${projectId}/expenses`, body)
      onDone()
    } catch (err) {
      setError((err as Error).message)
    }
  }

  async function remove() {
    if (!expense || !confirmDelete('udgiften')) return
    await api.del(`/expenses/${expense.id}`)
    onDone()
  }

  return (
    <Modal title={expense ? 'Redigér udgift' : 'Ny udgift'} onClose={onClose}>
      <form onSubmit={submit}>
        <div className="form-row">
          <div className="field" style={{ flex: 2 }}>
            <label>Beskrivelse *</label>
            <input value={description} onChange={(e) => setDescription(e.target.value)} required autoFocus={!expense} />
          </div>
          <div className="field">
            <label>Beløb (kr.) *</label>
            <input value={amount} onChange={(e) => setAmount(e.target.value)} required />
          </div>
          <div className="field">
            <label>Dato</label>
            <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </div>
        </div>
        <div className="form-row">
          <div className="field">
            <label>Budgetpost</label>
            <select value={budgetItemId} onChange={(e) => setBudgetItemId(e.target.value)}>
              <option value="">(ingen)</option>
              {items.map((b) => <option key={b.id} value={b.id}>{b.description}</option>)}
            </select>
          </div>
          <div className="field">
            <label>Fase</label>
            <select value={phaseId} onChange={(e) => setPhaseId(e.target.value)}>
              <option value="">(ingen)</option>
              {phases.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
            </select>
          </div>
          <div className="field">
            <label>Leverandør</label>
            <select value={supplierId} onChange={(e) => setSupplierId(e.target.value)}>
              <option value="">(ingen)</option>
              {suppliers.map((s) => <option key={s.id} value={s.id}>{s.companyName}</option>)}
            </select>
          </div>
        </div>
        <p className="hint">Kvittering: upload den under Dokumenter og knyt den til udgiften.</p>
        <ErrorText error={error} />
        <div className="form-row" style={{ justifyContent: 'space-between', marginTop: 12 }}>
          <div>{expense && <button type="button" className="btn danger" onClick={remove}>Slet</button>}</div>
          <div className="row-actions">
            <button type="button" className="btn secondary" onClick={onClose}>Annullér</button>
            <button className="btn">Gem</button>
          </div>
        </div>
      </form>
    </Modal>
  )
}
