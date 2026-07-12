import styles from './Avatar.module.css'

export type AvatarSize = 'sm' | 'md' | 'lg'

type AvatarProps = {
  /** Seed for stable hue (user id, or chat id for groups). */
  userId: number
  login: string
  size?: AvatarSize
  className?: string
}

/** Deterministic hue 0..359 from numeric id. */
export function avatarHue(userId: number): number {
  // 32-bit mix so the same id always yields the same hue across screens
  const mixed = Math.imul(Math.trunc(userId) | 0, 2654435761) >>> 0
  return mixed % 360
}

export function avatarInitials(login: string): string {
  const chars = [...login.trim()]
  if (chars.length === 0) {
    return '?'
  }
  if (chars.length === 1) {
    return chars[0]!.toUpperCase()
  }
  return `${chars[0]!}${chars[1]!}`.toUpperCase()
}

export function Avatar({ userId, login, size = 'md', className }: AvatarProps) {
  const hue = avatarHue(userId)
  const initials = avatarInitials(login)
  const sizeClass =
    size === 'sm' ? styles.sm : size === 'lg' ? styles.lg : styles.md

  return (
    <span
      className={`${styles.avatar} ${sizeClass} ${className ?? ''}`.trim()}
      style={{
        backgroundColor: `hsl(${hue} 52% 38%)`,
        color: 'var(--text-primary)',
      }}
      aria-hidden="true"
      title={login}
    >
      {initials}
    </span>
  )
}
