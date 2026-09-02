import { AlertTriangle, X } from 'lucide-react'
import { useEffect, useId, useRef } from 'react'

interface ConfirmDialogProps {
  open: boolean
  title: string
  description: string
  confirmLabel?: string
  busy?: boolean
  danger?: boolean
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({
  open,
  title,
  description,
  confirmLabel = '确认',
  busy = false,
  danger = false,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const cancelButtonRef = useRef<HTMLButtonElement>(null)
  const onCancelRef = useRef(onCancel)
  const busyRef = useRef(busy)
  const titleId = useId()
  const descriptionId = useId()

  useEffect(() => {
    onCancelRef.current = onCancel
    busyRef.current = busy
  }, [busy, onCancel])

  useEffect(() => {
    if (!open) return

    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    cancelButtonRef.current?.focus()

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        if (!busyRef.current) {
          event.preventDefault()
          onCancelRef.current()
        }
        return
      }
      if (event.key !== 'Tab') return

      const dialog = dialogRef.current
      if (!dialog) return
      const focusable = Array.from(
        dialog.querySelectorAll<HTMLElement>(
          'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
        ),
      )
      if (focusable.length === 0) {
        event.preventDefault()
        dialog.focus()
        return
      }

      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      if (!dialog.contains(document.activeElement)) {
        event.preventDefault()
        const target = event.shiftKey ? last : first
        target.focus()
      } else if (event.shiftKey && document.activeElement === first) {
        event.preventDefault()
        last.focus()
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault()
        first.focus()
      }
    }

    window.addEventListener('keydown', onKeyDown)
    return () => {
      window.removeEventListener('keydown', onKeyDown)
      document.body.style.overflow = previousOverflow
      previousFocus?.focus()
    }
  }, [open])

  if (!open) return null
  return (
    <div className="dialog-backdrop" onMouseDown={() => !busy && onCancel()}>
      <div ref={dialogRef} className="dialog" role="alertdialog" aria-modal="true" aria-labelledby={titleId} aria-describedby={descriptionId} aria-busy={busy} tabIndex={-1} onMouseDown={(event) => event.stopPropagation()}>
        <div className="dialog__header">
          <div>
            <span className="eyebrow"><AlertTriangle size={14} aria-hidden="true" />请确认</span>
            <h2 id={titleId}>{title}</h2>
          </div>
          <button className="icon-button" type="button" onClick={onCancel} disabled={busy} aria-label="关闭">
            <X size={18} aria-hidden="true" />
          </button>
        </div>
        <div className="dialog__body">
          <p className="confirm-copy" id={descriptionId}>{description}</p>
          <div className="form-actions">
            <button ref={cancelButtonRef} className="button button--soft" type="button" onClick={onCancel} disabled={busy}>取消</button>
            <button className={`button ${danger ? 'button--danger' : 'button--dark'}`} type="button" onClick={onConfirm} disabled={busy}>
              {busy ? '正在处理…' : confirmLabel}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
