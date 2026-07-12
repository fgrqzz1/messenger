import { Pencil, X } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { updateChatTitle } from '../../api/chats'
import { ApiError } from '../../api/errors'
import { useAuth } from '../../context/AuthContext'
import { useChatMembers } from '../../hooks/useChatMembers'
import { useChats } from '../../hooks/useChats'
import type { User } from '../../types/domain'
import { UserSearch } from '../UserSearch/UserSearch'
import styles from './MembersPanel.module.css'

type MembersPanelProps = {
  chatId: number
  open: boolean
  onClose: () => void
}

function titleErrorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.status === 400) {
      return 'Название не может быть пустым'
    }
    if (err.status === 403) {
      return err.message || 'Недостаточно прав для смены названия'
    }
    return err.message
  }
  return err instanceof Error ? err.message : 'Не удалось сменить название'
}

export function MembersPanel({ chatId, open, onClose }: MembersPanelProps) {
  const { currentUser } = useAuth()
  const { chats, peerNames, setChatTitle } = useChats()
  const {
    members,
    loading,
    error,
    isAdmin,
    actionError,
    actionLoading,
    addMember,
    removeMember,
  } = useChatMembers(chatId, currentUser?.id ?? null, open)

  const chat = useMemo(
    () => chats.find((item) => item.id === chatId) ?? null,
    [chatId, chats],
  )
  const isGroup = chat?.type === 'group'
  const canRename = isGroup && isAdmin
  const displayTitle = useMemo(() => {
    if (!chat) {
      return 'Участники'
    }
    if (chat.type === 'group') {
      return chat.title?.trim() || `Чат ${chat.id}`
    }
    return peerNames[chat.id] ?? 'Участники'
  }, [chat, peerNames])

  const [editingTitle, setEditingTitle] = useState(false)
  const [titleDraft, setTitleDraft] = useState('')
  const [titleError, setTitleError] = useState<string | null>(null)
  const [titleSaving, setTitleSaving] = useState(false)

  useEffect(() => {
    if (!open) {
      setEditingTitle(false)
      setTitleError(null)
      setTitleSaving(false)
    }
  }, [open])

  useEffect(() => {
    setEditingTitle(false)
    setTitleError(null)
    setTitleSaving(false)
  }, [chatId])

  useEffect(() => {
    if (!canRename && editingTitle) {
      setEditingTitle(false)
      setTitleError(null)
    }
  }, [canRename, editingTitle])

  useEffect(() => {
    if (!editingTitle) {
      setTitleDraft(chat?.title?.trim() || displayTitle)
    }
  }, [chat?.title, displayTitle, editingTitle])

  const excludeUserIds = useMemo(
    () => members.map((member) => member.user_id),
    [members],
  )

  const handleSelectUser = async (user: User) => {
    await addMember(user.id)
  }

  const startEditTitle = () => {
    if (!canRename) {
      return
    }
    setTitleDraft(chat?.title?.trim() || displayTitle)
    setTitleError(null)
    setEditingTitle(true)
  }

  const cancelEditTitle = () => {
    setTitleDraft(chat?.title?.trim() || displayTitle)
    setTitleError(null)
    setEditingTitle(false)
  }

  const saveTitle = async () => {
    const next = titleDraft.trim()
    if (!next) {
      setTitleError('Название не может быть пустым')
      return
    }
    if (next === (chat?.title?.trim() || '')) {
      setEditingTitle(false)
      setTitleError(null)
      return
    }

    setTitleSaving(true)
    setTitleError(null)
    try {
      const updated = await updateChatTitle(chatId, next)
      const title = updated.title?.trim() || next
      setChatTitle(chatId, title)
      setEditingTitle(false)
    } catch (err: unknown) {
      setTitleError(titleErrorMessage(err))
    } finally {
      setTitleSaving(false)
    }
  }

  const bannerError = actionError ?? error

  return (
    <aside
      className={`${styles.membersPanel} ${open ? styles.open : styles.closed}`}
      aria-label="Участники чата"
      aria-hidden={!open}
    >
      <div className={styles.panelInner}>
        <div className={styles.header}>
          {editingTitle && canRename ? (
            <div className={styles.titleEdit}>
              <input
                className={`${styles.titleInput} ${titleError ? styles.titleInputError : ''}`}
                value={titleDraft}
                onChange={(e) => setTitleDraft(e.target.value)}
                disabled={titleSaving}
                autoFocus
                maxLength={100}
                aria-label="Название группы"
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault()
                    void saveTitle()
                  }
                  if (e.key === 'Escape') {
                    cancelEditTitle()
                  }
                }}
              />
              <div className={styles.titleEditActions}>
                <button
                  type="button"
                  className={styles.titleSaveBtn}
                  disabled={titleSaving}
                  onClick={() => void saveTitle()}
                >
                  {titleSaving ? '…' : 'Сохранить'}
                </button>
                <button
                  type="button"
                  className={styles.titleCancelBtn}
                  disabled={titleSaving}
                  onClick={cancelEditTitle}
                >
                  Отмена
                </button>
              </div>
              {titleError && <p className={styles.titleFieldError}>{titleError}</p>}
            </div>
          ) : (
            <div className={styles.titleRow}>
              <h2 className={styles.title}>{displayTitle}</h2>
              {canRename && (
                <button
                  type="button"
                  className={styles.editTitleBtn}
                  aria-label="Изменить название"
                  onClick={startEditTitle}
                >
                  <Pencil size={14} strokeWidth={1.75} aria-hidden />
                </button>
              )}
            </div>
          )}
          <button
            type="button"
            className={styles.closeBtn}
            aria-label="Закрыть"
            onClick={onClose}
          >
            <X size={16} strokeWidth={1.75} aria-hidden />
          </button>
        </div>

        {bannerError && <div className={styles.errorBanner}>{bannerError}</div>}

        {isAdmin && (
          <div className={styles.addSection}>
            <p className={styles.addHint}>Добавить по логину</p>
            <UserSearch
              placeholder="Логин…"
              disabled={actionLoading || !open}
              excludeUserIds={excludeUserIds}
              onSelect={handleSelectUser}
            />
          </div>
        )}

        <ul className={styles.memberList}>
          {loading && members.length === 0 && (
            <li className={styles.stateMessage}>Загрузка…</li>
          )}
          {!loading && members.length === 0 && !error && (
            <li className={styles.stateMessage}>Нет участников</li>
          )}
          {members.map((member) => (
            <li key={member.user_id} className={styles.memberItem}>
              <div className={styles.memberInfo}>
                <span className={styles.memberLogin}>{member.login}</span>
                <span className={styles.memberRole}>{member.role}</span>
              </div>
              {isAdmin && member.user_id !== currentUser?.id && (
                <button
                  type="button"
                  className={styles.removeBtn}
                  disabled={actionLoading || !open}
                  onClick={() => void removeMember(member.user_id)}
                >
                  Удалить
                </button>
              )}
            </li>
          ))}
        </ul>
      </div>
    </aside>
  )
}
