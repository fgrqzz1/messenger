import { X } from 'lucide-react'
import { useEffect, useId, useState, type FormEvent } from 'react'
import { createPortal } from 'react-dom'
import { createDirectChat, createGroupChat } from '../../api/chats'
import { useActiveChat } from '../../context/ActiveChatContext'
import { useSidebar } from '../../context/SidebarContext'
import { useChats } from '../../hooks/useChats'
import type { User } from '../../types/domain'
import { UserSearch } from '../UserSearch/UserSearch'
import styles from './CreateChatModal.module.css'

type Tab = 'direct' | 'group'

type CreateChatModalProps = {
  open: boolean
  onClose: () => void
}

export function CreateChatModal({ open, onClose }: CreateChatModalProps) {
  const titleId = useId()
  const { upsertCreatedChat } = useChats()
  const { setActiveChatId, requestOpenMembersPanel } = useActiveChat()
  const { isNarrow, closeSidebar } = useSidebar()
  const [tab, setTab] = useState<Tab>('direct')
  const [groupTitle, setGroupTitle] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (!open) {
      return
    }

    setTab('direct')
    setGroupTitle('')
    setError(null)
    setSubmitting(false)
  }, [open])

  useEffect(() => {
    if (!open) {
      return
    }

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [open, onClose])

  if (!open) {
    return null
  }

  const openCreatedChat = (chatId: number, openMembers: boolean) => {
    setActiveChatId(chatId)
    if (openMembers) {
      requestOpenMembersPanel()
    }
    if (isNarrow) {
      closeSidebar()
    }
    onClose()
  }

  const handleSelectUser = async (user: User) => {
    setSubmitting(true)
    setError(null)
    try {
      const chat = await createDirectChat(user.id)
      upsertCreatedChat(chat, { login: user.login, userId: user.id })
      openCreatedChat(chat.id, false)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Не удалось создать чат')
    } finally {
      setSubmitting(false)
    }
  }

  const handleCreateGroup = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const title = groupTitle.trim()
    if (!title) {
      setError('Укажите название группы')
      return
    }

    setSubmitting(true)
    setError(null)
    try {
      const chat = await createGroupChat(title)
      upsertCreatedChat(chat)
      openCreatedChat(chat.id, true)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Не удалось создать группу')
    } finally {
      setSubmitting(false)
    }
  }

  return createPortal(
    <div className={styles.overlay} role="presentation" onClick={onClose}>
      <div
        className={styles.modal}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        onClick={(event) => event.stopPropagation()}
      >
        <div className={styles.header}>
          <h2 id={titleId} className={styles.title}>
            Новый чат
          </h2>
          <button
            type="button"
            className={styles.closeBtn}
            aria-label="Закрыть"
            onClick={onClose}
          >
            <X size={16} strokeWidth={1.75} aria-hidden />
          </button>
        </div>

        <div className={styles.tabs} role="tablist">
          <button
            type="button"
            role="tab"
            aria-selected={tab === 'direct'}
            className={`${styles.tab} ${tab === 'direct' ? styles.tabActive : ''}`}
            onClick={() => {
              setTab('direct')
              setError(null)
            }}
          >
            Личный чат
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={tab === 'group'}
            className={`${styles.tab} ${tab === 'group' ? styles.tabActive : ''}`}
            onClick={() => {
              setTab('group')
              setError(null)
            }}
          >
            Группа
          </button>
        </div>

        <div className={styles.body}>
          {error && <div className={styles.error}>{error}</div>}

          {tab === 'direct' && (
            <div className={styles.panel}>
              <p className={styles.hint}>Найдите пользователя по логину</p>
              <UserSearch
                placeholder="Логин собеседника…"
                disabled={submitting}
                autoFocus
                onSelect={handleSelectUser}
              />
            </div>
          )}

          {tab === 'group' && (
            <form className={styles.panel} onSubmit={(event) => void handleCreateGroup(event)}>
              <label className={styles.label} htmlFor="group-title">
                Название группы
              </label>
              <input
                id="group-title"
                className={styles.input}
                type="text"
                value={groupTitle}
                onChange={(event) => setGroupTitle(event.target.value)}
                placeholder="Название…"
                disabled={submitting}
                autoFocus
                required
              />
              <button
                type="submit"
                className={styles.submit}
                disabled={submitting || !groupTitle.trim()}
              >
                {submitting ? 'Создание…' : 'Создать'}
              </button>
            </form>
          )}
        </div>
      </div>
    </div>,
    document.body,
  )
}
