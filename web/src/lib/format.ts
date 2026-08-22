import type { ISODateTime, User } from '../types'

const AVATAR_COLORS = [
  '#4967e8',
  '#7c4dcb',
  '#b5457b',
  '#c45b36',
  '#ad7717',
  '#30815d',
  '#247a91',
  '#476b9e',
] as const

const DEFAULT_LOCALE = 'zh-CN'
const EMPTY_TEXT = '—'

export interface AvatarPresentation {
  initials: string
  backgroundColor: (typeof AVATAR_COLORS)[number]
  label: string
}

export function getDisplayName(
  user: Pick<User, 'nickname' | 'username'> | null | undefined,
): string {
  return user?.nickname.trim() || user?.username.trim() || '匿名用户'
}

export function getInitials(
  userOrName: Pick<User, 'nickname' | 'username'> | string | null | undefined,
  maxLength = 2,
): string {
  const name =
    typeof userOrName === 'string'
      ? userOrName.trim()
      : getDisplayName(userOrName)
  if (!name) return '?'

  const words = name.split(/\s+/u).filter(Boolean)
  const initials =
    words.length > 1
      ? `${Array.from(words[0] ?? '')[0] ?? ''}${Array.from(words[1] ?? '')[0] ?? ''}`
      : Array.from(name).slice(0, Math.max(1, maxLength)).join('')
  return initials.toLocaleUpperCase(DEFAULT_LOCALE)
}

export function getAvatar(
  user: Pick<User, 'id' | 'nickname' | 'username'> | null | undefined,
): AvatarPresentation {
  const label = getDisplayName(user)
  const seed = user ? `${user.id}:${user.username}` : label
  const index = positiveHash(seed) % AVATAR_COLORS.length
  return {
    initials: getInitials(label),
    backgroundColor: AVATAR_COLORS[index] ?? AVATAR_COLORS[0],
    label,
  }
}

export function getAvatarColor(seed: string | number): string {
  const index = positiveHash(String(seed)) % AVATAR_COLORS.length
  return AVATAR_COLORS[index] ?? AVATAR_COLORS[0]
}

export function formatDateTime(
  value: ISODateTime | Date | null | undefined,
  options: Intl.DateTimeFormatOptions = {},
  locale = DEFAULT_LOCALE,
): string {
  const date = toValidDate(value)
  if (!date) return EMPTY_TEXT

  return new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    ...options,
  }).format(date)
}

export function formatRelativeTime(
  value: ISODateTime | Date | null | undefined,
  now: Date | number = Date.now(),
  locale = DEFAULT_LOCALE,
): string {
  const date = toValidDate(value)
  const reference = typeof now === 'number' ? new Date(now) : now
  if (!date || Number.isNaN(reference.getTime())) return EMPTY_TEXT

  const seconds = (date.getTime() - reference.getTime()) / 1_000
  const absoluteSeconds = Math.abs(seconds)
  const formatter = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' })

  if (absoluteSeconds < 60) return formatter.format(Math.round(seconds), 'second')
  if (absoluteSeconds < 3_600) {
    return formatter.format(Math.round(seconds / 60), 'minute')
  }
  if (absoluteSeconds < 86_400) {
    return formatter.format(Math.round(seconds / 3_600), 'hour')
  }
  if (absoluteSeconds < 604_800) {
    return formatter.format(Math.round(seconds / 86_400), 'day')
  }
  if (absoluteSeconds < 2_592_000) {
    return formatter.format(Math.round(seconds / 604_800), 'week')
  }
  return formatDateTime(date, { hour: undefined, minute: undefined }, locale)
}

export function truncateText(
  value: string | null | undefined,
  maxLength = 120,
  suffix = '…',
): string {
  const normalized = normalizeSummary(value)
  if (!normalized) return ''
  const characters = Array.from(normalized)
  if (characters.length <= maxLength) return normalized
  if (maxLength <= 0) return ''
  return `${characters.slice(0, maxLength).join('').trimEnd()}${suffix}`
}

export function formatSummary(
  value: string | null | undefined,
  maxLength = 120,
): string {
  return truncateText(value, maxLength)
}

export function normalizeSummary(value: string | null | undefined): string {
  return value?.replace(/\s+/gu, ' ').trim() ?? ''
}

export function formatCount(value: number): string {
  const count = Number.isFinite(value) ? Math.max(0, value) : 0
  return new Intl.NumberFormat(DEFAULT_LOCALE, {
    notation: count >= 10_000 ? 'compact' : 'standard',
    maximumFractionDigits: 1,
  }).format(count)
}

function toValidDate(
  value: ISODateTime | Date | null | undefined,
): Date | null {
  if (value === null || value === undefined || value === '') return null
  const date = value instanceof Date ? value : new Date(value)
  return Number.isNaN(date.getTime()) ? null : date
}

function positiveHash(value: string): number {
  let hash = 0
  for (const character of value) {
    hash = (hash * 31 + (character.codePointAt(0) ?? 0)) | 0
  }
  return hash >>> 0
}
