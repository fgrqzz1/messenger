import { Plus, Search } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { getChatDisplayName } from '../../api/chats'
import { useActiveChat } from '../../context/ActiveChatContext'
import { useAuth } from '../../context/AuthContext'
import { useSidebar } from '../../context/SidebarContext'
import { useChats } from '../../hooks/useChats'
import { useWebSocket } from '../../hooks/useWebSocket'
import { chatHasUnread } from '../../utils/deliveryStatus'
import { formatChatTime } from '../../utils/formatChatTime'
import { Avatar } from '../Avatar/Avatar'
import { CreateChatModal } from '../CreateChatModal/CreateChatModal'
import { EmptyState } from '../EmptyState/EmptyState'
import { ChatListSkeleton } from '../Skeleton/Skeleton'
import styles from './Sidebar.module.css'

type SidebarProps = {
  onOpenProfile: () => void
}

function AppLogo() {
  return (
    <svg
      className={styles.brandIcon}
      viewBox="0 0 24 24"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
    >
      <path
        d="M5 7.5A2.5 2.5 0 0 1 7.5 5h9A2.5 2.5 0 0 1 19 7.5v6A2.5 2.5 0 0 1 16.5 16H11l-4 3v-3H7.5A2.5 2.5 0 0 1 5 13.5v-6Z"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinejoin="round"
      />
      <circle cx="9" cy="10.5" r="1" fill="currentColor" />
      <circle cx="12" cy="10.5" r="1" fill="currentColor" />
      <circle cx="15" cy="10.5" r="1" fill="currentColor" />
    </svg>
  )
}

export function Sidebar({ onOpenProfile }: SidebarProps) {
  const { currentUser } = useAuth()
  const { chats, loading, error, peerNames, peerUserIds } = useChats()
  const { activeChatId, setActiveChatId } = useActiveChat()
  const { status } = useWebSocket()
  const { isNarrow, sidebarOpen, closeSidebar } = useSidebar()
  const [search, setSearch] = useState('')
  const [createOpen, setCreateOpen] = useState(false)

  const isOnline = status === 'online'

  const filteredChats = useMemo(() => {
    const query = search.trim().toLowerCase()
    if (!query) {
      return chats
    }
    return chats.filter((chat) =>
      getChatDisplayName(chat, peerNames).toLowerCase().includes(query),
    )
  }, [chats, peerNames, search])

  useEffect(() => {
    if (filteredChats.length === 0) {
      return
    }

    const activeStillVisible = filteredChats.some((chat) => chat.id === activeChatId)
    if (activeChatId === null || !activeStillVisible) {
      setActiveChatId(filteredChats[0].id)
    }
  }, [activeChatId, filteredChats, setActiveChatId])

  const selectChat = (chatId: number) => {
    setActiveChatId(chatId)
    if (isNarrow) {
      closeSidebar()
    }
  }

  return (
    <aside
      className={`${styles.sidebar} ${isNarrow && sidebarOpen ? styles.sidebarOpen : ''}`}
    >
      <div className={styles.sidebarUpper}>
        <div className={styles.header}>
          <div className={styles.brand}>
            <AppLogo />
            <span className={styles.brandName}>Messenger</span>
          </div>
          <div className={styles.searchRow}>
            <div className={styles.searchField}>
              <Search className={styles.searchIcon} size={16} strokeWidth={1.75} aria-hidden />
              <input
                className={styles.search}
                type="search"
                placeholder="Поиск чатов…"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
            <button
              type="button"
              className={styles.createBtn}
              aria-label="Создать чат"
              title="Создать чат"
              onClick={() => setCreateOpen(true)}
            >
              <Plus size={18} strokeWidth={2} aria-hidden />
            </button>
          </div>
        </div>

        <div className={styles.listRegion}>
          {loading ? (
            <ChatListSkeleton />
          ) : (
            <ul className={styles.chatList}>
              {error && <li className={styles.stateMessage}>{error}</li>}
              {!error && filteredChats.length === 0 && (
                <li className={styles.emptyItem}>
                  <EmptyState
                    variant="noChats"
                    compact
                    title={search.trim() ? 'Ничего не найдено' : 'Нет чатов'}
                  />
                </li>
              )}
              {!error &&
                filteredChats.map((chat) => {
                  const unread = chatHasUnread(chat)
                  const name = getChatDisplayName(chat, peerNames)
                  const avatarId =
                    chat.type === 'direct' && peerUserIds[chat.id] != null
                      ? peerUserIds[chat.id]!
                      : chat.id
                  return (
                    <li
                      key={chat.id}
                      className={`${styles.chatItem} ${chat.id === activeChatId ? styles.chatItemActive : ''} ${unread ? styles.chatItemUnread : ''}`}
                      onClick={() => selectChat(chat.id)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') selectChat(chat.id)
                      }}
                      role="button"
                      tabIndex={0}
                    >
                      <Avatar userId={avatarId} login={name} size="sm" />
                      <div className={styles.chatBody}>
                        <div className={styles.chatRow}>
                          <h2 className={styles.chatName}>{name}</h2>
                          <div className={styles.chatMeta}>
                            {unread && (
                              <span className={styles.unreadDot} aria-label="Есть непрочитанные" />
                            )}
                            <span className={styles.chatTime}>
                              {formatChatTime(chat.last_message_at)}
                            </span>
                          </div>
                        </div>
                        <p className={styles.chatPreview}>
                          {chat.last_message_body ?? 'Нет сообщений'}
                        </p>
                      </div>
                    </li>
                  )
                })}
            </ul>
          )}
        </div>
      </div>

      <footer className={`chromeBar ${styles.footer}`}>
        <div className={styles.profileCard}>
          <button
            type="button"
            className={styles.profileBtn}
            onClick={onOpenProfile}
            aria-label="Открыть профиль"
          >
            {currentUser && (
              <Avatar userId={currentUser.id} login={currentUser.login} size="sm" />
            )}
            <span className={styles.profileLogin}>
              {currentUser?.login ?? 'Профиль'}
            </span>
          </button>
          <div className={styles.connection}>
            <span
              className={`${styles.statusDot} ${isOnline ? styles.statusDotOnline : styles.statusDotReconnecting}`}
              aria-hidden="true"
            />
            <span>{isOnline ? 'online' : 'reconnecting'}</span>
          </div>
        </div>
      </footer>

      <CreateChatModal open={createOpen} onClose={() => setCreateOpen(false)} />
    </aside>
  )
}
