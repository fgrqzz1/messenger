import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { fetchChats } from '../api/chats'
import { fetchChatMembers } from '../api/members'
import { useAuth } from '../context/AuthContext'
import type { Chat, ChatListItem } from '../types/domain'

type ReloadChatsOptions = {
  /** Не показывать «Загрузка…» в сайдбаре (фоновый синк по WS). */
  silent?: boolean
}

type ChatsContextValue = {
  chats: ChatListItem[]
  peerNames: Record<number, string>
  loading: boolean
  error: string | null
  /** Обновляет превью; `false`, если чата ещё нет в локальном списке. */
  updateChatPreview: (chatId: number, body: string, createdAt: string) => boolean
  /** Сразу вставляет ответ POST /chats в начало списка (без ожидания WS/рефетча). */
  upsertCreatedChat: (chat: Chat, peerLogin?: string) => void
  /**
   * Если `chat_id` из `new_message` отсутствует в списке — точечный рефетч GET /chats.
   * Иначе no-op.
   */
  ensureChatFromMessage: (chatId: number) => Promise<void>
  reloadChats: (options?: ReloadChatsOptions) => Promise<void>
}

const ChatsContext = createContext<ChatsContextValue | null>(null)

function sortChatsByLastMessage(chats: ChatListItem[]): ChatListItem[] {
  return [...chats].sort((a, b) => {
    const aTime = a.last_message_at ? Date.parse(a.last_message_at) : 0
    const bTime = b.last_message_at ? Date.parse(b.last_message_at) : 0
    if (aTime !== bTime) {
      return bTime - aTime
    }
    return b.id - a.id
  })
}

function chatToListItem(chat: Chat): ChatListItem {
  return {
    id: chat.id,
    type: chat.type,
    title: chat.title,
    last_message_body: null,
    last_message_at: null,
  }
}

async function loadPeerNames(
  items: ChatListItem[],
  currentUserId: number,
): Promise<Record<number, string>> {
  const directChats = items.filter((chat) => chat.type === 'direct' && !chat.title)
  if (directChats.length === 0) {
    return {}
  }

  const entries = await Promise.all(
    directChats.map(async (chat) => {
      try {
        const members = await fetchChatMembers(chat.id)
        const peer = members.find((member) => member.user_id !== currentUserId)
        return peer ? ([chat.id, peer.login] as const) : null
      } catch {
        return null
      }
    }),
  )

  const next: Record<number, string> = {}
  for (const entry of entries) {
    if (entry) {
      next[entry[0]] = entry[1]
    }
  }
  return next
}

export function ChatsProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated, currentUser } = useAuth()
  const [chats, setChats] = useState<ChatListItem[]>([])
  const [peerNames, setPeerNames] = useState<Record<number, string>>({})
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const chatsRef = useRef(chats)
  const refreshInFlightRef = useRef<Promise<void> | null>(null)

  chatsRef.current = chats

  const reloadChats = useCallback(
    async (options?: ReloadChatsOptions) => {
      if (!isAuthenticated || !currentUser) {
        setChats([])
        setPeerNames({})
        return
      }

      if (refreshInFlightRef.current) {
        return refreshInFlightRef.current
      }

      const silent = options?.silent === true
      const run = async () => {
        if (!silent) {
          setLoading(true)
        }
        setError(null)

        try {
          const items = await fetchChats()
          setChats(items)
          setPeerNames(await loadPeerNames(items, currentUser.id))
        } catch (err: unknown) {
          setError(err instanceof Error ? err.message : 'Не удалось загрузить чаты')
        } finally {
          if (!silent) {
            setLoading(false)
          }
        }
      }

      refreshInFlightRef.current = run().finally(() => {
        refreshInFlightRef.current = null
      })
      return refreshInFlightRef.current
    },
    [currentUser, isAuthenticated],
  )

  useEffect(() => {
    if (!isAuthenticated) {
      setChats([])
      setPeerNames({})
      setLoading(false)
      setError(null)
      return
    }

    let cancelled = false

    void (async () => {
      setLoading(true)
      setError(null)
      try {
        const items = await fetchChats()
        if (cancelled) {
          return
        }
        setChats(items)
        if (currentUser) {
          setPeerNames(await loadPeerNames(items, currentUser.id))
        } else {
          setPeerNames({})
        }
      } catch (err: unknown) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Не удалось загрузить чаты')
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    })()

    return () => {
      cancelled = true
    }
  }, [currentUser, isAuthenticated])

  const updateChatPreview = useCallback(
    (chatId: number, body: string, createdAt: string): boolean => {
      if (!chatsRef.current.some((chat) => chat.id === chatId)) {
        return false
      }

      setChats((prev) =>
        sortChatsByLastMessage(
          prev.map((chat) =>
            chat.id === chatId
              ? {
                  ...chat,
                  last_message_body: body,
                  last_message_at: createdAt,
                }
              : chat,
          ),
        ),
      )
      return true
    },
    [],
  )

  const upsertCreatedChat = useCallback(
    (chat: Chat, peerLogin?: string) => {
      const item = chatToListItem(chat)
      setChats((prev) => {
        if (prev.some((existing) => existing.id === chat.id)) {
          return prev
        }
        return [item, ...prev]
      })

      if (peerLogin) {
        setPeerNames((prev) => ({ ...prev, [chat.id]: peerLogin }))
        return
      }

      if (chat.type === 'direct' && currentUser) {
        void (async () => {
          const names = await loadPeerNames([item], currentUser.id)
          if (names[chat.id]) {
            setPeerNames((prev) => ({ ...prev, [chat.id]: names[chat.id] }))
          }
        })()
      }
    },
    [currentUser],
  )

  const ensureChatFromMessage = useCallback(
    async (chatId: number) => {
      if (chatsRef.current.some((chat) => chat.id === chatId)) {
        return
      }
      await reloadChats({ silent: true })
    },
    [reloadChats],
  )

  const value = useMemo(
    () => ({
      chats,
      peerNames,
      loading,
      error,
      updateChatPreview,
      upsertCreatedChat,
      ensureChatFromMessage,
      reloadChats,
    }),
    [
      chats,
      error,
      ensureChatFromMessage,
      loading,
      peerNames,
      reloadChats,
      updateChatPreview,
      upsertCreatedChat,
    ],
  )

  return <ChatsContext.Provider value={value}>{children}</ChatsContext.Provider>
}

export function useChats(): ChatsContextValue {
  const ctx = useContext(ChatsContext)
  if (!ctx) {
    throw new Error('useChats must be used within ChatsProvider')
  }
  return ctx
}
