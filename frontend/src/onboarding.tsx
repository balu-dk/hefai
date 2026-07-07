import { useEffect, useState } from 'react'

/** useStoredState persists simple UI-state (tilstand, rundvisning, tjekliste). */
export function useStoredState<T extends string>(key: string, fallback: T): [T, (v: T) => void] {
  const [value, setValue] = useState<T>(() => (localStorage.getItem(key) as T) ?? fallback)
  const set = (v: T) => {
    localStorage.setItem(key, v)
    setValue(v)
  }
  return [value, set]
}

export interface TourStep {
  selector: string
  title: string
  text: string
}

/**
 * Guidet rundvisning: fremhæver ét element ad gangen med en forklarende
 * boble. Ingen dependencies — position beregnes fra elementets rect.
 */
export function Tour({ steps, onDone }: { steps: TourStep[]; onDone: () => void }) {
  const [index, setIndex] = useState(0)
  const [rect, setRect] = useState<DOMRect | null>(null)
  const step = steps[index]

  useEffect(() => {
    const el = document.querySelector(step.selector)
    if (!el) {
      // Elementet findes ikke (fx skjult i Enkel tilstand) — spring videre.
      if (index < steps.length - 1) setIndex(index + 1)
      else onDone()
      return
    }
    el.scrollIntoView({ block: 'nearest' })
    setRect(el.getBoundingClientRect())
  }, [index, step.selector, steps.length, onDone])

  if (!rect) return null

  const bubbleTop = Math.min(Math.max(rect.top - 10, 12), window.innerHeight - 220)
  const bubbleLeft = Math.min(rect.right + 16, window.innerWidth - 340)

  return (
    <div className="tour-backdrop">
      <div
        className="tour-highlight"
        style={{ top: rect.top - 6, left: rect.left - 6, width: rect.width + 12, height: rect.height + 12 }}
      />
      <div className="tour-bubble" style={{ top: bubbleTop, left: bubbleLeft }}>
        <div className="tour-step-no">Trin {index + 1} af {steps.length}</div>
        <h3 style={{ margin: '4px 0 6px' }}>{step.title}</h3>
        <p style={{ margin: 0, fontSize: 13.5 }}>{step.text}</p>
        <div className="form-row" style={{ justifyContent: 'space-between', marginTop: 14, marginBottom: 0 }}>
          <button className="btn small secondary" onClick={onDone}>Spring over</button>
          <div className="row-actions">
            {index > 0 && (
              <button className="btn small secondary" onClick={() => setIndex(index - 1)}>Tilbage</button>
            )}
            {index < steps.length - 1 ? (
              <button className="btn small" onClick={() => setIndex(index + 1)}>Næste</button>
            ) : (
              <button className="btn small" onClick={onDone}>Færdig — i gang!</button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
