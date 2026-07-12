import { getAccessToken } from './auth'
import type { Message } from '../types/domain'

export const WS_FRAME_ACK = 'ack'
export const WS_FRAME_NEW_MESSAGE = 'new_message'
export const WS_FRAME_SEND_MESSAGE = 'send_message'
export const WS_FRAME_READ = 'read'

export type WsStatus = 'online' | 'reconnecting'

export type WsAckFrame = {
  type: typeof WS_FRAME_ACK
  client_msg_id: string
  server_id: number
}

export type WsNewMessageFrame = {
  type: typeof WS_FRAME_NEW_MESSAGE
  chat_id: number
  message: Message
}

export type WsReadFrame = {
  type: typeof WS_FRAME_READ
  chat_id: number
  user_id: number
  last_read_message_id: number
}

export type OutgoingMessage = {
  chatId: number
  clientMsgId: string
  body: string
}

type WsHandlers = {
  onStatusChange?: (status: WsStatus) => void
  onAck?: (frame: WsAckFrame, chatId?: number) => void
  onNewMessage?: (frame: WsNewMessageFrame) => void
  onRead?: (frame: WsReadFrame) => void
}

const INITIAL_BACKOFF_MS = 1_000
const MAX_BACKOFF_MS = 30_000

/**
 * WS URL всегда от адреса страницы (не от VITE_API_URL).
 * Иначе абсолютный `http://localhost:8080` ломает доступ с других устройств в LAN:
 * браузер пытается открыть WebSocket к localhost на самом устройстве.
 * http→ws / https→wss; host+port — из window.location.host.
 */
function getWsUrl(): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/ws`
}

function isAckFrame(value: unknown): value is WsAckFrame {
  return (
    typeof value === 'object' &&
    value !== null &&
    (value as WsAckFrame).type === WS_FRAME_ACK &&
    typeof (value as WsAckFrame).client_msg_id === 'string' &&
    typeof (value as WsAckFrame).server_id === 'number'
  )
}

function isNewMessageFrame(value: unknown): value is WsNewMessageFrame {
  return (
    typeof value === 'object' &&
    value !== null &&
    (value as WsNewMessageFrame).type === WS_FRAME_NEW_MESSAGE &&
    typeof (value as WsNewMessageFrame).chat_id === 'number' &&
    typeof (value as WsNewMessageFrame).message === 'object' &&
    (value as WsNewMessageFrame).message !== null
  )
}

function isReadFrame(value: unknown): value is WsReadFrame {
  return (
    typeof value === 'object' &&
    value !== null &&
    (value as WsReadFrame).type === WS_FRAME_READ &&
    typeof (value as WsReadFrame).chat_id === 'number' &&
    typeof (value as WsReadFrame).user_id === 'number' &&
    typeof (value as WsReadFrame).last_read_message_id === 'number'
  )
}

export class MessengerWebSocket {
  private ws: WebSocket | null = null
  private handlers: WsHandlers = {}
  private pendingQueue: OutgoingMessage[] = []
  private reconnectAttempt = 0
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private intentionalClose = false
  private status: WsStatus = 'reconnecting'

  setHandlers(handlers: WsHandlers): void {
    this.handlers = handlers
    handlers.onStatusChange?.(this.status)
  }

  connect(): void {
    this.intentionalClose = false
    this.clearReconnectTimer()
    this.openSocket()
  }

  disconnect(): void {
    this.intentionalClose = true
    this.clearReconnectTimer()
    this.closeSocket()
    this.setStatus('reconnecting')
  }

  sendMessage(payload: OutgoingMessage): void {
    const queued = this.pendingQueue.some(
      (item) =>
        item.chatId === payload.chatId && item.clientMsgId === payload.clientMsgId,
    )
    if (!queued) {
      this.pendingQueue.push(payload)
    }

    this.flushQueue()
  }

  private openSocket(): void {
    this.closeSocket()
    this.setStatus('reconnecting')

    const token = getAccessToken()
    if (!token) {
      return
    }

    const socket = new WebSocket(getWsUrl())
    this.ws = socket

    socket.onopen = () => {
      if (this.ws !== socket) {
        return
      }
      this.reconnectAttempt = 0
      this.setStatus('online')
      socket.send(JSON.stringify({ token: getAccessToken() ?? token }))
      this.flushQueue()
    }

    socket.onmessage = (event) => {
      if (this.ws !== socket) {
        return
      }

      let data: unknown
      try {
        data = JSON.parse(String(event.data))
      } catch {
        return
      }

      if (isAckFrame(data)) {
        const pending = this.pendingQueue.find(
          (item) => item.clientMsgId === data.client_msg_id,
        )
        this.removeFromQueue(data.client_msg_id)
        this.handlers.onAck?.(data, pending?.chatId)
        return
      }

      if (isNewMessageFrame(data)) {
        this.handlers.onNewMessage?.(data)
        return
      }

      if (isReadFrame(data)) {
        this.handlers.onRead?.(data)
      }
    }

    socket.onclose = () => {
      if (this.ws !== socket) {
        return
      }
      this.ws = null

      if (this.intentionalClose) {
        return
      }

      this.scheduleReconnect()
    }

    socket.onerror = () => {
      if (this.ws !== socket) {
        return
      }
      socket.close()
    }
  }

  private flushQueue(): void {
    const socket = this.ws
    if (!socket || socket.readyState !== WebSocket.OPEN || this.status !== 'online') {
      return
    }

    for (const item of [...this.pendingQueue]) {
      socket.send(
        JSON.stringify({
          type: WS_FRAME_SEND_MESSAGE,
          chat_id: item.chatId,
          client_msg_id: item.clientMsgId,
          body: item.body,
        }),
      )
    }
  }

  private removeFromQueue(clientMsgId: string): void {
    this.pendingQueue = this.pendingQueue.filter((item) => item.clientMsgId !== clientMsgId)
  }

  private scheduleReconnect(): void {
    if (this.intentionalClose || this.reconnectTimer !== null) {
      return
    }

    this.setStatus('reconnecting')

    const delay = Math.min(
      INITIAL_BACKOFF_MS * 2 ** this.reconnectAttempt,
      MAX_BACKOFF_MS,
    )
    this.reconnectAttempt += 1

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      if (!this.intentionalClose) {
        this.openSocket()
      }
    }, delay)
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
  }

  private closeSocket(): void {
    if (this.ws) {
      this.ws.onopen = null
      this.ws.onmessage = null
      this.ws.onclose = null
      this.ws.onerror = null
      this.ws.close()
      this.ws = null
    }
  }

  private setStatus(status: WsStatus): void {
    if (this.status === status) {
      return
    }
    this.status = status
    this.handlers.onStatusChange?.(status)
  }
}
