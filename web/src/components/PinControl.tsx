import { Pin, PinOff } from 'lucide-react'
import { useId, useState } from 'react'
import { POST_PIN_DURATION, type PostPinDuration } from '../types'

const PIN_OPTIONS: Array<{ value: PostPinDuration; label: string }> = [
  { value: POST_PIN_DURATION.DAY, label: '1 天' },
  { value: POST_PIN_DURATION.WEEK, label: '1 周' },
  { value: POST_PIN_DURATION.MONTH, label: '1 个月（30 天）' },
  { value: POST_PIN_DURATION.PERMANENT, label: '永久' },
]

interface PinControlProps {
  isPinned: boolean
  canPin: boolean
  busy: boolean
  postTitle: string
  onPin: (duration: PostPinDuration) => void | Promise<void>
  onUnpin: () => void | Promise<void>
}

export function PinControl({ isPinned, canPin, busy, postTitle, onPin, onUnpin }: PinControlProps) {
  const selectID = useId()
  const [duration, setDuration] = useState<PostPinDuration>(POST_PIN_DURATION.DAY)

  if (isPinned) {
    return (
      <button className="button button--soft button--small" type="button" onClick={() => void onUnpin()} disabled={busy} aria-label={`取消置顶《${postTitle}》`}>
        <PinOff size={14} aria-hidden="true" />{busy ? '处理中…' : '取消置顶'}
      </button>
    )
  }

  return (
    <span className="pin-control">
      <label className="sr-only" htmlFor={selectID}>《{postTitle}》的置顶期限</label>
      <select
        id={selectID}
        className="pin-control__select"
        value={duration}
        onChange={(event) => setDuration(event.target.value as PostPinDuration)}
        disabled={busy || !canPin}
        title={!canPin ? '请先将帖子设为公开' : '选择置顶期限'}
      >
        {PIN_OPTIONS.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
      </select>
      <button className="button button--soft button--small" type="button" onClick={() => void onPin(duration)} disabled={busy || !canPin} aria-label={`${canPin ? '置顶' : '仅公开帖子可置顶'}《${postTitle}》`} title={!canPin ? '请先将帖子设为公开' : undefined}>
        <Pin size={14} aria-hidden="true" />{busy ? '处理中…' : canPin ? '置顶' : '仅公开帖可置顶'}
      </button>
    </span>
  )
}
