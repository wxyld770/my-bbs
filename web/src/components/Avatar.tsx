import { getAvatar } from '../lib/format'
import type { User } from '../types'

interface AvatarProps {
  user?: Pick<User, 'id' | 'nickname' | 'username'> | null
  size?: 'sm' | 'md' | 'lg'
}

export function Avatar({ user, size = 'md' }: AvatarProps) {
  const avatar = getAvatar(user)
  return (
    <span
      className={`avatar avatar--${size}`}
      style={{ backgroundColor: avatar.backgroundColor }}
      role="img"
      aria-label={`${avatar.label}的头像`}
      title={avatar.label}
    >
      {avatar.initials}
    </span>
  )
}
