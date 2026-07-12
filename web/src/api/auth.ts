import type { TokenPair, User } from '../types/domain'
import { parseUserIdFromAccessToken } from '../utils/jwt'
import { ApiError, parseApiError } from './errors'

/**
 * Access и refresh токены хранятся только в оперативной памяти этого модуля.
 * Намеренно не используем localStorage/sessionStorage: при XSS украсть токен
 * из памяти сложнее, чем из persistent storage, и сессия не переживает
 * перезагрузку вкладки без повторного входа.
 */
let accessToken: string | null = null
let refreshToken: string | null = null
let currentUser: User | null = null

type AuthListener = () => void
const listeners = new Set<AuthListener>()

export function getAccessToken(): string | null {
  return accessToken
}

export function getRefreshToken(): string | null {
  return refreshToken
}

export function isAuthenticated(): boolean {
  return accessToken !== null
}

export function getCurrentUser(): User | null {
  return currentUser
}

function syncCurrentUserFromAccessToken(): void {
  if (!accessToken) {
    currentUser = null
    return
  }

  const userId = parseUserIdFromAccessToken(accessToken)
  if (userId === null) {
    currentUser = null
    return
  }

  currentUser = {
    id: userId,
    login: currentUser?.login ?? `#${userId}`,
  }
}

export function setTokens(access: string, refresh: string): void {
  accessToken = access
  refreshToken = refresh
  syncCurrentUserFromAccessToken()
  notifyListeners()
}

export function setAccessToken(access: string): void {
  accessToken = access
  syncCurrentUserFromAccessToken()
  notifyListeners()
}

export function clearTokens(): void {
  accessToken = null
  refreshToken = null
  currentUser = null
  notifyListeners()
}

/** Обновляет поля текущего пользователя в памяти (например после PATCH /me). */
export function patchCurrentUser(patch: Partial<User>): void {
  if (!currentUser) {
    return
  }
  currentUser = { ...currentUser, ...patch }
  notifyListeners()
}

export function subscribeAuth(listener: AuthListener): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

function notifyListeners(): void {
  for (const listener of listeners) {
    listener()
  }
}

const API_BASE = import.meta.env.VITE_API_URL ?? ''

async function postJson<T>(path: string, body: unknown, bearer?: string): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  if (bearer) {
    headers.Authorization = `Bearer ${bearer}`
  }

  const response = await fetch(`${API_BASE}${path}`, {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  })

  if (!response.ok) {
    throw await parseApiError(response)
  }

  return response.json() as Promise<T>
}

export async function register(loginValue: string, password: string): Promise<User> {
  return postJson<User>('/register', { login: loginValue, password })
}

export async function login(loginValue: string, password: string): Promise<TokenPair> {
  const pair = await postJson<TokenPair>('/login', { login: loginValue, password })
  accessToken = pair.access_token
  refreshToken = pair.refresh_token

  const userId = parseUserIdFromAccessToken(pair.access_token)
  currentUser =
    userId === null
      ? null
      : {
          id: userId,
          login: loginValue,
        }

  notifyListeners()
  return pair
}

export async function refresh(): Promise<string> {
  const currentRefresh = getRefreshToken()
  if (!currentRefresh) {
    throw new ApiError(401, 'unauthorized', 'No refresh token')
  }

  const data = await postJson<{ access_token: string }>(
    '/refresh',
    {},
    currentRefresh,
  )
  setAccessToken(data.access_token)
  return data.access_token
}
