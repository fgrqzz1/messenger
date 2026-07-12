import type { DeliveryStatus, DisplayMessage } from '../types/domain'

/**
 * Статус своего сообщения: pending → acked → read
 * (read = все остальные участники имеют last_read_message_id >= id).
 * Визуал (◌ / ✓ / ✓✓) — в MessageStatus.
 */
export function resolveOwnDeliveryStatus(
  message: DisplayMessage,
  currentUserId: number,
  readCursors: Record<number, number>,
): DeliveryStatus {
  if (message.delivery_status === 'pending') {
    return 'pending'
  }

  if (message.id <= 0) {
    return 'acked'
  }

  const otherCursors = Object.entries(readCursors)
    .filter(([userId]) => Number(userId) !== currentUserId)
    .map(([, cursor]) => cursor)

  if (otherCursors.length === 0) {
    return 'acked'
  }

  if (otherCursors.every((cursor) => cursor >= message.id)) {
    return 'read'
  }

  return 'acked'
}

/** Непрочитан ли чат по полям GET /chats. */
export function chatHasUnread(chat: {
  last_message_id?: number | null
  my_last_read_message_id?: number
}): boolean {
  const lastId = chat.last_message_id
  if (lastId == null || lastId <= 0) {
    return false
  }
  return lastId > (chat.my_last_read_message_id ?? 0)
}
