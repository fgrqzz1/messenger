import { useMemo } from 'react'
import { useAuth } from '../../context/AuthContext'
import { useChatMembers } from '../../hooks/useChatMembers'
import type { User } from '../../types/domain'
import { UserSearch } from '../UserSearch/UserSearch'
import styles from './MembersPanel.module.css'

type MembersPanelProps = {
  chatId: number
  open: boolean
  onClose: () => void
}

export function MembersPanel({ chatId, open, onClose }: MembersPanelProps) {
  const { currentUser } = useAuth()
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

  const excludeUserIds = useMemo(
    () => members.map((member) => member.user_id),
    [members],
  )

  const handleSelectUser = async (user: User) => {
    await addMember(user.id)
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
          <h2 className={styles.title}>Участники</h2>
          <button
            type="button"
            className={styles.closeBtn}
            aria-label="Закрыть"
            onClick={onClose}
          >
            ×
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
