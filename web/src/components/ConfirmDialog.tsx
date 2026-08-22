import { AlertTriangle, X } from 'lucide-react'

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
  if (!open) return null
  return (
    <div className="dialog-backdrop" onMouseDown={() => !busy && onCancel()}>
      <div className="dialog" role="alertdialog" aria-modal="true" aria-labelledby="confirm-title" aria-describedby="confirm-description" onMouseDown={(event) => event.stopPropagation()}>
        <div className="dialog__header">
          <div>
            <span className="eyebrow"><AlertTriangle size={14} aria-hidden="true" />请确认</span>
            <h2 id="confirm-title">{title}</h2>
          </div>
          <button className="icon-button" type="button" onClick={onCancel} disabled={busy} aria-label="关闭">
            <X size={18} aria-hidden="true" />
          </button>
        </div>
        <div className="dialog__body">
          <p className="confirm-copy" id="confirm-description">{description}</p>
          <div className="form-actions">
            <button className="button button--soft" type="button" onClick={onCancel} disabled={busy}>取消</button>
            <button className={`button ${danger ? 'button--danger' : 'button--dark'}`} type="button" onClick={onConfirm} disabled={busy}>
              {busy ? '正在处理…' : confirmLabel}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
