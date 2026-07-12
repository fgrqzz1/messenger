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
import styles from './Sidebar.module.css'

type SidebarProps = {
  onOpenProfile: () => void
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
      <div className={styles.searchWrap}>
        <div className={styles.searchRow}>
          <input
            className={styles.search}
            type="search"
            placeholder="Поиск чатов…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <button
            type="button"
            className={styles.createBtn}
            aria-label="Создать чат"
            title="Создать чат"
            onClick={() => setCreateOpen(true)}
          >
            +
          </button>
        </div>
      </div>

      <ul className={styles.chatList}>
        {loading && <li className={styles.stateMessage}>Загрузка…</li>}
        {!loading && error && <li className={styles.stateMessage}>{error}</li>}
        {!loading && !error && filteredChats.length === 0 && (
          <li className={styles.stateMessage}>
            {search.trim() ? 'Ничего не найдено' : 'Нет чатов'}
          </li>
        )}
        {!loading &&
          !error &&
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

      <footer className={styles.footer}>
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
          >
            {isOnline ? '●' : '○'}
          </span>
          <span>{isOnline ? 'online' : 'reconnecting'}</span>
        </div>
      </footer>

      <CreateChatModal open={createOpen} onClose={() => setCreateOpen(false)} />
    </aside>
  )
}
