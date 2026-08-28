import { Check, Copy, KeyRound, X } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'

interface InvitationDialogProps {
  code: string | null
  onClose: () => void
}

type CopyState = 'idle' | 'copied' | 'failed'

export function InvitationDialog({ code, onClose }: InvitationDialogProps) {
  const [copyState, setCopyState] = useState<CopyState>('idle')
  const dialogRef = useRef<HTMLDivElement>(null)
  const closeButtonRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (!code) return
    setCopyState('idle')
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
    closeButtonRef.current?.focus()
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose()
        return
      }
      if (event.key !== 'Tab') return

      const focusable = Array.from(
        dialogRef.current?.querySelectorAll<HTMLElement>('button:not([disabled]), [href], input, select, textarea, [tabindex]:not([tabindex="-1"])') ?? [],
      )
      if (focusable.length === 0) return
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault()
        last.focus()
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault()
        first.focus()
      }
    }
    document.body.style.overflow = 'hidden'
    window.addEventListener('keydown', onKeyDown)
    return () => {
      document.body.style.overflow = ''
      window.removeEventListener('keydown', onKeyDown)
      previousFocus?.focus()
    }
  }, [code, onClose])

  if (!code) return null

  const copyCode = async () => {
    try {
      await copyToClipboard(code)
      setCopyState('copied')
    } catch {
      setCopyState('failed')
    }
  }

  return (
    <div className="dialog-backdrop" onMouseDown={onClose}>
      <div
        ref={dialogRef}
        className="dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="invitation-title"
        aria-describedby="invitation-description"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <div className="dialog__header">
          <div>
            <span className="eyebrow"><KeyRound size={14} aria-hidden="true" />邀请新朋友</span>
            <h2 id="invitation-title">邀请码已生成</h2>
          </div>
          <button ref={closeButtonRef} className="icon-button" type="button" onClick={onClose} aria-label="关闭并清除邀请码">
            <X size={18} aria-hidden="true" />
          </button>
        </div>
        <div className="dialog__body">
          <p className="invitation-copy" id="invitation-description">
            请现在复制并妥善保管。关闭弹窗后，系统将不再展示这个邀请码。
          </p>
          <div className="invitation-code">
            <output aria-label="刚刚生成的邀请码">{code}</output>
            <button className="button button--dark" type="button" onClick={() => void copyCode()}>
              {copyState === 'copied' ? <Check size={16} aria-hidden="true" /> : <Copy size={16} aria-hidden="true" />}
              {copyState === 'copied' ? '已复制' : '复制邀请码'}
            </button>
          </div>
          <p className={`invitation-copy-status${copyState === 'failed' ? ' invitation-copy-status--error' : ''}`} aria-live="polite">
            {copyState === 'failed' ? '复制失败，请手动选择邀请码复制。' : copyState === 'copied' ? '邀请码已复制到剪贴板。' : '请只分享给你希望邀请的人。'}
          </p>
          <div className="form-actions">
            <button className="button button--soft" type="button" onClick={onClose}>我已保存，关闭</button>
          </div>
        </div>
      </div>
    </div>
  )
}

async function copyToClipboard(value: string): Promise<void> {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(value)
    return
  }

  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.style.position = 'fixed'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)
  textarea.select()
  const copied = document.execCommand('copy')
  textarea.remove()
  if (!copied) throw new Error('复制失败')
}
