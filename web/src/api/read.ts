import type { ChatReadState } from '../types/domain'
import { apiClient } from './client'

export function markChatRead(
  chatId: number,
  lastReadMessageId: number,
): Promise<void> {
  return apiClient<void>(`/chats/${chatId}/read`, {
    method: 'POST',
    body: JSON.stringify({ last_read_message_id: lastReadMessageId }),
  })
}

export function fetchChatReadState(chatId: number): Promise<ChatReadState[]> {
  return apiClient<ChatReadState[]>(`/chats/${chatId}/read-state`)
}
