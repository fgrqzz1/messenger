export type ChatType = 'direct' | 'group'

export type ChatListItem = {
  id: number
  type: ChatType
  title?: string | null
  last_message_id?: number | null
  last_message_body?: string | null
  last_message_at?: string | null
  /** Курсор прочтения вызывающего; 0 если ещё не отмечал. */
  my_last_read_message_id?: number
}

export type ChatReadState = {
  user_id: number
  last_read_message_id: number
}

/** Ответ POST /chats */
export type Chat = {
  id: number
  type: ChatType
  title?: string | null
  user_a_id?: number | null
  user_b_id?: number | null
  created_by?: number | null
  created_at: string
}

export type User = {
  id: number
  login: string
}

export type TokenPair = {
  access_token: string
  refresh_token: string
}

export type DeliveryStatus = 'pending' | 'acked' | 'read'

export type Message = {
  id: number
  sender_id: number
  body: string
  created_at: string
}

/** Сообщение в ленте: серверное или оптимистичное (с client_msg_id до ack). */
export type DisplayMessage = Message & {
  client_msg_id?: string
  delivery_status?: DeliveryStatus
}

export type ChatMember = {
  user_id: number
  login: string
  role: 'member' | 'admin'
}
