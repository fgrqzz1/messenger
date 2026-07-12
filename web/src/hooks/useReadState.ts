import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { fetchChatReadState, markChatRead } from '../api/read'
import type { DisplayMessage } from '../types/domain'

type UseReadStateOptions = {
  chatId: number | null
  messages: DisplayMessage[]
  loading: boolean
  currentUserId: number | null
  advanceMyReadCursor: (chatId: number, messageId: number) => void
  registerReadHandler: (
    handler: ((userId: number, lastReadMessageId: number) => void) | null,
  ) => void
}

export function useReadState({
  chatId,
  messages,
  loading,
  currentUserId,
  advanceMyReadCursor,
  registerReadHandler,
}: UseReadStateOptions) {
  const [readCursors, setReadCursors] = useState<Record<number, number>>({})
  const lastMarkedRef = useRef(0)

  const applyRead = useCallback((userId: number, lastReadMessageId: number) => {
    setReadCursors((prev) => {
      const current = prev[userId] ?? 0
      if (lastReadMessageId <= current) {
        return prev
      }
      return { ...prev, [userId]: lastReadMessageId }
    })
  }, [])

  useEffect(() => {
    setReadCursors({})
    lastMarkedRef.current = 0

    if (chatId === null) {
      return
    }

    let cancelled = false

    fetchChatReadState(chatId)
      .then((states) => {
        if (cancelled) {
          return
        }
        setReadCursors((prev) => {
          const next: Record<number, number> = {}
          for (const state of states) {
            next[state.user_id] = Math.max(
              prev[state.user_id] ?? 0,
              state.last_read_message_id,
            )
          }
          return next
        })
      })
      .catch(() => {
        /* keep existing cursors on fetch failure */
      })

    return () => {
      cancelled = true
    }
  }, [chatId])

  useEffect(() => {
    if (chatId === null) {
      registerReadHandler(null)
      return
    }

    registerReadHandler(applyRead)
    return () => {
      registerReadHandler(null)
    }
  }, [applyRead, chatId, registerReadHandler])

  const maxMessageId = useMemo(() => {
    let max = 0
    for (const message of messages) {
      if (message.id > max) {
        max = message.id
      }
    }
    return max
  }, [messages])

  useEffect(() => {
    if (chatId === null || loading || currentUserId === null || maxMessageId <= 0) {
      return
    }
    if (maxMessageId <= lastMarkedRef.current) {
      return
    }

    const targetId = maxMessageId
    lastMarkedRef.current = targetId
    applyRead(currentUserId, targetId)
    advanceMyReadCursor(chatId, targetId)

    void markChatRead(chatId, targetId).catch(() => {
      if (lastMarkedRef.current === targetId) {
        lastMarkedRef.current = 0
      }
    })
  }, [
    advanceMyReadCursor,
    applyRead,
    chatId,
    currentUserId,
    loading,
    maxMessageId,
  ])

  return { readCursors }
}
