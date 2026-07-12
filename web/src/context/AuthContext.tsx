import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useSyncExternalStore,
} from 'react'
import * as authApi from '../api/auth'
import { configureClient } from '../api/client'
import { ApiError } from '../api/errors'
import type { User } from '../types/domain'

type AuthContextValue = {
  isAuthenticated: boolean
  currentUser: User | null
  login: (login: string, password: string) => Promise<void>
  register: (login: string, password: string) => Promise<void>
  logout: () => void
  patchCurrentUser: (patch: Partial<User>) => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

function subscribe(callback: () => void): () => void {
  return authApi.subscribeAuth(callback)
}

function getSnapshot(): boolean {
  return authApi.isAuthenticated()
}

function getUserSnapshot(): User | null {
  return authApi.getCurrentUser()
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useSyncExternalStore(subscribe, getSnapshot, () => false)
  const currentUser = useSyncExternalStore(subscribe, getUserSnapshot, () => null)

  const logout = useCallback(() => {
    authApi.clearTokens()
  }, [])

  const patchCurrentUser = useCallback((patch: Partial<User>) => {
    authApi.patchCurrentUser(patch)
  }, [])

  useEffect(() => {
    configureClient({ onSessionExpired: logout })
  }, [logout])

  const login = useCallback(async (loginValue: string, password: string) => {
    await authApi.login(loginValue, password)
  }, [])

  const register = useCallback(async (loginValue: string, password: string) => {
    await authApi.register(loginValue, password)
    await authApi.login(loginValue, password)
  }, [])

  const value = useMemo<AuthContextValue>(
    () => ({
      isAuthenticated,
      currentUser,
      login,
      register,
      logout,
      patchCurrentUser,
    }),
    [isAuthenticated, currentUser, login, register, logout, patchCurrentUser],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return ctx
}

export function mapAuthError(
  err: unknown,
  mode: 'login' | 'register',
): { login?: string; password?: string } {
  if (!(err instanceof ApiError)) {
    return { login: 'Не удалось связаться с сервером' }
  }

  if (err.code === 'conflict') {
    return { login: 'Логин занят' }
  }

  if (err.code === 'invalid_credentials') {
    return { password: 'Неверный пароль' }
  }

  if (err.code === 'validation_error') {
    return mode === 'register'
      ? { login: 'Проверьте логин и пароль' }
      : { login: 'Введите логин и пароль' }
  }

  return { login: err.message || 'Ошибка запроса' }
}
