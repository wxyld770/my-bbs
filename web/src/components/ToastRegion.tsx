import { CheckCircle2, CircleAlert, Info, X } from 'lucide-react'
import { useUI, type ToastTone } from '../context/UIContext'

const toneIcon: Record<ToastTone, typeof Info> = {
  success: CheckCircle2,
  error: CircleAlert,
  info: Info,
}

export function ToastRegion() {
  const { toasts, dismissToast } = useUI()
  return (
    <div className="toast-region" aria-live="polite" aria-atomic="false">
      {toasts.map((toast) => {
        const Icon = toneIcon[toast.tone]
        return (
          <div className={`toast toast--${toast.tone}`} key={toast.id} role="status">
            <Icon size={19} aria-hidden="true" />
            <div className="toast__content">
              <strong>{toast.title}</strong>
              {toast.message && <p>{toast.message}</p>}
            </div>
            <button
              className="icon-button"
              style={{ width: 28, height: 28 }}
              type="button"
              onClick={() => dismissToast(toast.id)}
              aria-label="关闭提示"
            >
              <X size={14} aria-hidden="true" />
            </button>
          </div>
        )
      })}
    </div>
  )
}
