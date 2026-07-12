import styles from './EmptyState.module.css'

type EmptyStateProps = {
  variant: 'noChats' | 'selectChat'
  title: string
  compact?: boolean
}

function NoChatsIllustration() {
  return (
    <svg
      className={styles.emptyStateIllustration}
      viewBox="0 0 120 96"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
    >
      <rect
        x="18"
        y="14"
        width="84"
        height="58"
        rx="10"
        stroke="var(--text-secondary)"
        strokeWidth="2"
        opacity="0.55"
      />
      <path
        d="M34 36h52M34 48h36"
        stroke="var(--text-secondary)"
        strokeWidth="2"
        strokeLinecap="round"
        opacity="0.45"
      />
      <circle cx="88" cy="70" r="14" fill="var(--bg-elevated)" stroke="var(--accent)" strokeWidth="2" />
      <path
        d="M88 64v12M82 70h12"
        stroke="var(--accent)"
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  )
}

function SelectChatIllustration() {
  return (
    <svg
      className={styles.emptyStateIllustration}
      viewBox="0 0 120 96"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
    >
      <rect
        x="12"
        y="22"
        width="52"
        height="34"
        rx="12"
        fill="var(--bg-elevated)"
        stroke="var(--text-secondary)"
        strokeWidth="2"
        opacity="0.85"
      />
      <path
        d="M24 68 36 56h16"
        stroke="var(--text-secondary)"
        strokeWidth="2"
        strokeLinejoin="round"
        opacity="0.55"
      />
      <rect
        x="56"
        y="12"
        width="52"
        height="34"
        rx="12"
        fill="color-mix(in srgb, var(--accent) 28%, transparent)"
        stroke="var(--accent)"
        strokeWidth="2"
      />
      <path
        d="M96 56 84 46H68"
        stroke="var(--accent)"
        strokeWidth="2"
        strokeLinejoin="round"
        opacity="0.8"
      />
      <circle cx="28" cy="78" r="3" fill="var(--text-secondary)" opacity="0.4" />
      <circle cx="40" cy="78" r="3" fill="var(--text-secondary)" opacity="0.55" />
      <circle cx="52" cy="78" r="3" fill="var(--accent)" opacity="0.7" />
    </svg>
  )
}

export function EmptyState({ variant, title, compact }: EmptyStateProps) {
  return (
    <div className={`${styles.emptyState} ${compact ? styles.emptyStateCompact : ''}`}>
      {variant === 'noChats' ? <NoChatsIllustration /> : <SelectChatIllustration />}
      <p className={styles.emptyStateTitle}>{title}</p>
    </div>
  )
}
