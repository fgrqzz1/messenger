import { type FormEvent, useEffect, useState } from 'react'
import * as profileApi from '../../api/profile'
import { ApiError } from '../../api/errors'
import { useAuth } from '../../context/AuthContext'
import type { MeUser } from '../../types/domain'
import { formatRegistrationDate } from '../../utils/formatRegistrationDate'
import { Avatar } from '../../components/Avatar/Avatar'
import styles from './Profile.module.css'

type ProfileProps = {
  onBack: () => void
}

export function Profile({ onBack }: ProfileProps) {
  const { currentUser, logout, patchCurrentUser } = useAuth()
  const [me, setMe] = useState<MeUser | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const [editingLogin, setEditingLogin] = useState(false)
  const [loginDraft, setLoginDraft] = useState('')
  const [loginError, setLoginError] = useState<string | null>(null)
  const [loginSaving, setLoginSaving] = useState(false)

  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [passwordErrors, setPasswordErrors] = useState<{
    current?: string
    next?: string
    confirm?: string
  }>({})
  const [passwordSaving, setPasswordSaving] = useState(false)
  const [passwordSuccess, setPasswordSuccess] = useState(false)

  useEffect(() => {
    let cancelled = false
    void (async () => {
      setLoading(true)
      setLoadError(null)
      try {
        const data = await profileApi.getMe()
        if (cancelled) {
          return
        }
        setMe(data)
        setLoginDraft(data.login)
        patchCurrentUser({ id: data.id, login: data.login })
      } catch (err: unknown) {
        if (!cancelled) {
          setLoadError(err instanceof Error ? err.message : 'Не удалось загрузить профиль')
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
  }, [patchCurrentUser])

  const displayLogin = me?.login ?? currentUser?.login ?? ''
  const displayId = me?.id ?? currentUser?.id ?? 0

  const startEditLogin = () => {
    setLoginDraft(displayLogin)
    setLoginError(null)
    setEditingLogin(true)
  }

  const cancelEditLogin = () => {
    setLoginDraft(displayLogin)
    setLoginError(null)
    setEditingLogin(false)
  }

  const saveLogin = async () => {
    const next = loginDraft.trim()
    if (!next) {
      setLoginError('Введите логин')
      return
    }
    if (next === displayLogin) {
      setEditingLogin(false)
      setLoginError(null)
      return
    }

    setLoginSaving(true)
    setLoginError(null)
    try {
      const updated = await profileApi.updateLogin(next)
      setMe(updated)
      patchCurrentUser({ id: updated.id, login: updated.login })
      setEditingLogin(false)
    } catch (err: unknown) {
      if (err instanceof ApiError && err.code === 'conflict') {
        setLoginError('логин занят')
      } else {
        setLoginError(err instanceof Error ? err.message : 'Не удалось сохранить логин')
      }
    } finally {
      setLoginSaving(false)
    }
  }

  const handlePasswordSubmit = async (event: FormEvent) => {
    event.preventDefault()
    const nextErrors: { current?: string; next?: string; confirm?: string } = {}
    if (!currentPassword) {
      nextErrors.current = 'Введите текущий пароль'
    }
    if (!newPassword) {
      nextErrors.next = 'Введите новый пароль'
    }
    if (!confirmPassword) {
      nextErrors.confirm = 'Подтвердите новый пароль'
    } else if (newPassword && confirmPassword !== newPassword) {
      nextErrors.confirm = 'Пароли не совпадают'
    }

    setPasswordErrors(nextErrors)
    setPasswordSuccess(false)
    if (Object.keys(nextErrors).length > 0) {
      return
    }

    setPasswordSaving(true)
    try {
      await profileApi.updatePassword(currentPassword, newPassword)
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
      setPasswordErrors({})
      setPasswordSuccess(true)
    } catch (err: unknown) {
      if (err instanceof ApiError && err.code === 'invalid_credentials') {
        setPasswordErrors({ current: 'Неверный текущий пароль' })
      } else {
        setPasswordErrors({
          current: err instanceof Error ? err.message : 'Не удалось сменить пароль',
        })
      }
    } finally {
      setPasswordSaving(false)
    }
  }

  const handleLogout = () => {
    logout()
  }

  return (
    <div className={styles.screen}>
      <div className={styles.card}>
        <button type="button" className={styles.backBtn} onClick={onBack}>
          ← Назад к чатам
        </button>

        {loading && <p className={styles.hint}>Загрузка профиля…</p>}
        {loadError && <p className={styles.error}>{loadError}</p>}

        {!loading && !loadError && displayId > 0 && (
          <>
            <div className={styles.identity}>
              <Avatar userId={displayId} login={displayLogin} size="lg" />
              <div className={styles.identityText}>
                {editingLogin ? (
                  <div className={styles.loginEdit}>
                    <input
                      className={`${styles.input} ${loginError ? styles.inputError : ''}`}
                      value={loginDraft}
                      onChange={(e) => setLoginDraft(e.target.value)}
                      disabled={loginSaving}
                      autoFocus
                      aria-label="Новый логин"
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') {
                          e.preventDefault()
                          void saveLogin()
                        }
                        if (e.key === 'Escape') {
                          cancelEditLogin()
                        }
                      }}
                    />
                    <div className={styles.loginEditActions}>
                      <button
                        type="button"
                        className={styles.primaryBtn}
                        disabled={loginSaving}
                        onClick={() => void saveLogin()}
                      >
                        {loginSaving ? 'Сохранение…' : 'Сохранить'}
                      </button>
                      <button
                        type="button"
                        className={styles.ghostBtn}
                        disabled={loginSaving}
                        onClick={cancelEditLogin}
                      >
                        Отмена
                      </button>
                    </div>
                    {loginError && <p className={styles.fieldError}>{loginError}</p>}
                  </div>
                ) : (
                  <button
                    type="button"
                    className={styles.loginBtn}
                    onClick={startEditLogin}
                    title="Изменить логин"
                  >
                    <span className={styles.login}>{displayLogin}</span>
                    <span className={styles.loginHint}>изменить</span>
                  </button>
                )}
                {me?.created_at && (
                  <p className={styles.meta}>
                    Регистрация: {formatRegistrationDate(me.created_at)}
                  </p>
                )}
              </div>
            </div>

            <form className={styles.form} onSubmit={(e) => void handlePasswordSubmit(e)} noValidate>
              <h2 className={styles.sectionTitle}>Смена пароля</h2>

              <div className={styles.field}>
                <label className={styles.label} htmlFor="current-password">
                  Текущий пароль
                </label>
                <input
                  id="current-password"
                  className={`${styles.input} ${passwordErrors.current ? styles.inputError : ''}`}
                  type="password"
                  autoComplete="current-password"
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  disabled={passwordSaving}
                />
                {passwordErrors.current && (
                  <p className={styles.fieldError}>{passwordErrors.current}</p>
                )}
              </div>

              <div className={styles.field}>
                <label className={styles.label} htmlFor="new-password">
                  Новый пароль
                </label>
                <input
                  id="new-password"
                  className={`${styles.input} ${passwordErrors.next ? styles.inputError : ''}`}
                  type="password"
                  autoComplete="new-password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  disabled={passwordSaving}
                />
                {passwordErrors.next && (
                  <p className={styles.fieldError}>{passwordErrors.next}</p>
                )}
              </div>

              <div className={styles.field}>
                <label className={styles.label} htmlFor="confirm-password">
                  Подтверждение нового пароля
                </label>
                <input
                  id="confirm-password"
                  className={`${styles.input} ${passwordErrors.confirm ? styles.inputError : ''}`}
                  type="password"
                  autoComplete="new-password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  disabled={passwordSaving}
                />
                {passwordErrors.confirm && (
                  <p className={styles.fieldError}>{passwordErrors.confirm}</p>
                )}
              </div>

              {passwordSuccess && (
                <p className={styles.success}>Пароль обновлён</p>
              )}

              <button type="submit" className={styles.primaryBtn} disabled={passwordSaving}>
                {passwordSaving ? 'Сохранение…' : 'Сменить пароль'}
              </button>
            </form>

            <button type="button" className={styles.logoutBtn} onClick={handleLogout}>
              Выйти
            </button>
          </>
        )}
      </div>
    </div>
  )
}
