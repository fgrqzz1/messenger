import type { DeliveryStatus } from '../../types/domain'
import styles from './MessageStatus.module.css'

export type MessageStatusKind = 'sending' | 'sent' | 'read'

type MessageStatusProps = {
  status: MessageStatusKind
}

const ARIA: Record<MessageStatusKind, string> = {
  sending: 'Ожидает подтверждения',
  sent: 'Доставлено на сервер',
  read: 'Прочитано',
}

/** Map domain DeliveryStatus → UI kind (логика статуса снаружи). */
export function toMessageStatusKind(status: DeliveryStatus): MessageStatusKind {
  switch (status) {
    case 'pending':
      return 'sending'
    case 'acked':
      return 'sent'
    case 'read':
      return 'read'
  }
}

function SingleCheckIcon() {
  return (
    <svg
      className={styles.icon}
      viewBox="0 0 12 10"
      width="12"
      height="10"
      aria-hidden="true"
      focusable="false"
    >
      <path
        className={styles.check}
        d="M1.2 5.2 4.1 8.1 10.8 1.4"
        fill="none"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

function DoubleCheckIcon() {
  return (
    <svg
      className={styles.icon}
      viewBox="0 0 16 10"
      width="16"
      height="10"
      aria-hidden="true"
      focusable="false"
    >
      <path
        className={styles.check}
        d="M1.2 5.2 4.1 8.1 9.2 2.8"
        fill="none"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        className={`${styles.check} ${styles.checkSecond}`}
        d="M5.2 5.2 8.1 8.1 14.8 1.4"
        fill="none"
        strokeWidth="1.6"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

/**
 * Визуальный индикатор статуса своего сообщения.
 * ◌ / ✓ / ✓✓ (SVG). Какой статус показывать — решает вызывающий код.
 */
export function MessageStatus({ status }: MessageStatusProps) {
  return (
    <span className={styles.root} aria-label={ARIA[status]}>
      {status === 'sending' && <span className={styles.sending}>◌</span>}
      {status === 'sent' && (
        <span className={styles.sent}>
          <SingleCheckIcon />
        </span>
      )}
      {status === 'read' && (
        <span className={styles.read}>
          <DoubleCheckIcon />
        </span>
      )}
    </span>
  )
}
