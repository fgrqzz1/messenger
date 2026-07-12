import { useCallback, useEffect, useMemo, useRef, useState, type KeyboardEvent } from 'react'
import { getChatDisplayName } from '../../api/chats'
import { useActiveChat } from '../../context/ActiveChatContext'
import { useAuth } from '../../context/AuthContext'
import { useSidebar } from '../../context/SidebarContext'
import { useChats } from '../../hooks/useChats'
import { useMemberNames } from '../../hooks/useMemberNames'
import { useMessages } from '../../hooks/useMessages'
import { useReadState } from '../../hooks/useReadState'
import { useWebSocket } from '../../hooks/useWebSocket'
import { MembersPanel } from '../MembersPanel/MembersPanel'
import { SearchPanel } from '../SearchPanel/SearchPanel'
import { Avatar } from '../Avatar/Avatar'
import {
  MessageStatus,
  toMessageStatusKind,
} from '../MessageStatus/MessageStatus'
import { resolveOwnDeliveryStatus } from '../../utils/deliveryStatus'
import { formatMessageTime } from '../../utils/formatMessageTime'
import styles from './ChatWindow.module.css'

type ChatWindowProps = {
  chatId: number | null
  chatTitle: string | null
  chatType: 'direct' | 'group' | null
  avatarUserId: number | null
}

function resolveSenderName(
  senderId: number,
  chatType: 'direct' | 'group' | null,
  currentUserId: number | null,
  memberNames: Record<number, string>,
): string | undefined {
  if (chatType !== 'group' || currentUserId === senderId) {
    return undefined
  }

  return memberNames[senderId]
}

function resizeTextarea(element: HTMLTextAreaElement): void {
  element.style.height = 'auto'
  element.style.height = `${Math.min(element.scrollHeight, 120)}px`
}

export function ChatWindow({ chatId, chatTitle, chatType, avatarUserId }: ChatWindowProps) {
  const { currentUser } = useAuth()
  const { membersPanelRequest } = useActiveChat()
  const { isNarrow, toggleSidebar } = useSidebar()
  const { advanceMyReadCursor } = useChats()
  const { sendMessage, registerChatHandlers, registerReadHandler } = useWebSocket()
  const {
    messages,
    loading,
    loadingMore,
    error,
    listRef,
    handleScroll,
    messageKey,
    scrollToMessage,
    highlightedMessageId,
  } = useMessages(chatId, registerChatHandlers)
  const { readCursors } = useReadState({
    chatId,
    messages,
    loading,
    currentUserId: currentUser?.id ?? null,
    advanceMyReadCursor,
    registerReadHandler,
  })
  const memberNames = useMemberNames(chatId, chatType, currentUser)
  const [draft, setDraft] = useState('')
  const [membersOpen, setMembersOpen] = useState(false)
  const [searchOpen, setSearchOpen] = useState(false)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const lastMembersRequestRef = useRef(0)

  const headerTitle = chatTitle ?? 'Выберите чат'
  const canSend = chatId !== null && draft.trim().length > 0

  useEffect(() => {
    setSearchOpen(false)
    if (membersPanelRequest > lastMembersRequestRef.current) {
      lastMembersRequestRef.current = membersPanelRequest
      setMembersOpen(true)
      return
    }
    setMembersOpen(false)
  }, [chatId, membersPanelRequest])

  useEffect(() => {
    const textarea = textareaRef.current
    if (textarea) {
      resizeTextarea(textarea)
    }
  }, [draft, chatId])

  const openMembers = useCallback(() => {
    setSearchOpen(false)
    setMembersOpen(true)
  }, [])

  const handleSearchSelect = useCallback(
    async (messageId: number) => {
      await scrollToMessage(messageId)
    },
    [scrollToMessage],
  )

  const handleSend = useCallback(() => {
    if (chatId === null || !draft.trim()) {
      return
    }

    sendMessage(chatId, draft)
    setDraft('')
    const textarea = textareaRef.current
    if (textarea) {
      textarea.style.height = 'auto'
    }
  }, [chatId, draft, sendMessage])

  const handleKeyDown = useCallback(
    (event: KeyboardEvent<HTMLTextAreaElement>) => {
      if (event.key === 'Enter' && !event.shiftKey) {
        event.preventDefault()
        handleSend()
      }
    },
    [handleSend],
  )

  return (
    <section className={styles.chatWindow}>
      <div className={styles.chatMain}>
        <header className={styles.header}>
          {isNarrow && (
            <button
              type="button"
              className={styles.menuBtn}
              aria-label="Список чатов"
              onClick={toggleSidebar}
            >
              ☰
            </button>
          )}
          <button
            type="button"
            className={styles.headerTitleBtn}
            onClick={chatId !== null ? openMembers : undefined}
            disabled={chatId === null}
          >
            {chatId !== null && avatarUserId !== null && chatTitle && (
              <Avatar userId={avatarUserId} login={chatTitle} size="sm" />
            )}
            <h1 className={styles.headerTitle}>{headerTitle}</h1>
          </button>
          <div className={styles.headerActions}>
            <button
              type="button"
              className={`${styles.headerBtn} ${searchOpen ? styles.headerBtnActive : ''}`}
              aria-label="Поиск по чату"
              disabled={chatId === null}
              onClick={() => {
                setMembersOpen(false)
                setSearchOpen((open) => !open)
              }}
            >
              ⌕
            </button>
            <button
              type="button"
              className={`${styles.headerBtn} ${membersOpen ? styles.headerBtnActive : ''}`}
              aria-label="Сведения о чате"
              disabled={chatId === null}
              onClick={() => {
                setSearchOpen(false)
                setMembersOpen((open) => !open)
              }}
            >
              i
            </button>
          </div>
        </header>

        {chatId === null ? (
          <div className={styles.emptyState}>Выберите чат в списке слева</div>
        ) : (
          <>
            {searchOpen && (
              <SearchPanel
                chatId={chatId}
                memberNames={memberNames}
                onClose={() => setSearchOpen(false)}
                onSelectMessage={handleSearchSelect}
              />
            )}

            {error && <div className={styles.errorBanner}>{error}</div>}

            <div className={styles.feedArea}>
              <ul
                ref={listRef}
                className={styles.messageList}
                onScroll={handleScroll}
              >
                {loadingMore && (
                  <li className={styles.loadMoreHint}>Загрузка…</li>
                )}
                {loading && messages.length === 0 && (
                  <li className={styles.stateMessage}>Загрузка сообщений…</li>
                )}
                {!loading && !error && messages.length === 0 && (
                  <li className={styles.stateMessage}>Нет сообщений</li>
                )}
                {messages.map((msg) => {
                  const isOwn = currentUser?.id === msg.sender_id
                  const senderName = resolveSenderName(
                    msg.sender_id,
                    chatType,
                    currentUser?.id ?? null,
                    memberNames,
                  )
                  const deliveryStatus =
                    isOwn && currentUser
                      ? resolveOwnDeliveryStatus(msg, currentUser.id, readCursors)
                      : null
                  const showDelivery = isOwn && (msg.delivery_status != null || msg.id > 0)
                  const isHighlighted = msg.id > 0 && msg.id === highlightedMessageId

                  return (
                    <li
                      key={messageKey(msg)}
                      data-message-id={msg.id > 0 ? msg.id : undefined}
                      className={`${styles.bubbleWrap} ${isOwn ? styles.bubbleWrapOwn : styles.bubbleWrapOther} ${isHighlighted ? styles.bubbleWrapHighlight : ''}`}
                    >
                      {!isOwn && senderName && (
                        <span className={styles.senderName}>{senderName}</span>
                      )}
                      <div
                        className={`${styles.bubble} ${isOwn ? styles.bubbleOwn : styles.bubbleOther}`}
                      >
                        {msg.body}
                      </div>
                      <div className={styles.meta}>
                        <span className={styles.timestamp}>
                          {formatMessageTime(msg.created_at)}
                        </span>
                        {showDelivery && deliveryStatus && (
                          <MessageStatus status={toMessageStatusKind(deliveryStatus)} />
                        )}
                      </div>
                    </li>
                  )
                })}
              </ul>
            </div>
          </>
        )}

        <div className={styles.inputArea}>
          <textarea
            ref={textareaRef}
            className={styles.textarea}
            rows={1}
            placeholder="Сообщение…"
            value={draft}
            onChange={(e) => {
              setDraft(e.target.value)
              resizeTextarea(e.currentTarget)
            }}
            onKeyDown={handleKeyDown}
            disabled={chatId === null}
          />
          <button
            type="button"
            className={styles.sendBtn}
            disabled={!canSend}
            aria-label="Отправить"
            onClick={handleSend}
          >
            ➤
          </button>
        </div>
      </div>

      {chatId !== null && (
        <MembersPanel
          chatId={chatId}
          open={membersOpen}
          onClose={() => setMembersOpen(false)}
        />
      )}
    </section>
  )
}

export function ConnectedChatWindow() {
  const { chats, peerNames, peerUserIds } = useChats()
  const { activeChatId } = useActiveChat()

  const activeChat = useMemo(
    () => chats.find((chat) => chat.id === activeChatId) ?? null,
    [activeChatId, chats],
  )

  const chatTitle = activeChat ? getChatDisplayName(activeChat, peerNames) : null
  const avatarUserId = activeChat
    ? activeChat.type === 'direct' && peerUserIds[activeChat.id] != null
      ? peerUserIds[activeChat.id]!
      : activeChat.id
    : null

  return (
    <ChatWindow
      chatId={activeChat?.id ?? null}
      chatTitle={chatTitle}
      chatType={activeChat?.type ?? null}
      avatarUserId={avatarUserId}
    />
  )
}
