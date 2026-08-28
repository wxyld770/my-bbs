import { useEffect, useState } from 'react'
import { getAvatar } from '../lib/format'
import type { User } from '../types'

interface AvatarProps {
  user?: Pick<User, 'id' | 'nickname' | 'username' | 'avatar_url'> | null
  size?: 'sm' | 'md' | 'lg'
}

export function Avatar({ user, size = 'md' }: AvatarProps) {
  const avatar = getAvatar(user)
  const avatarURL = user?.avatar_url?.trim() ?? ''
  const [failedURL, setFailedURL] = useState('')

  useEffect(() => {
    setFailedURL('')
  }, [avatarURL])

  if (avatarURL && failedURL !== avatarURL) {
    return (
      <img
        className={`avatar avatar--${size}`}
        src={avatarURL}
        alt={`${avatar.label}的头像`}
        loading="lazy"
        decoding="async"
        referrerPolicy="no-referrer"
        title={avatar.label}
        onError={() => setFailedURL(avatarURL)}
      />
    )
  }
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
