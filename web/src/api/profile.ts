import type { MeUser } from '../types/domain'
import { apiClient } from './client'

export function getMe(): Promise<MeUser> {
  return apiClient<MeUser>('/me')
}

export function updateLogin(login: string): Promise<MeUser> {
  return apiClient<MeUser>('/me', {
    method: 'PATCH',
    body: JSON.stringify({ login }),
  })
}

export function updatePassword(
  currentPassword: string,
  newPassword: string,
): Promise<void> {
  return apiClient<void>('/me/password', {
    method: 'PATCH',
    body: JSON.stringify({
      current_password: currentPassword,
      new_password: newPassword,
    }),
  })
}
