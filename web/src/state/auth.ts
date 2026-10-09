import { reactive } from 'vue'

import { createSession, deleteCurrentSession, getCurrentUser, registerUser } from '../api/generated'
import type { User } from '../api/generated'
import { isProblem } from '../utils/problem'

export const authState = reactive<{ user: User | null; ready: boolean }>({
  user: null,
  ready: false,
})

let currentUserRequest: Promise<void> | undefined

export function ensureCurrentUser(): Promise<void> {
  if (authState.ready) return Promise.resolve()
  if (currentUserRequest) return currentUserRequest
  currentUserRequest = (async () => {
    try {
      const response = await getCurrentUser()
      authState.user = response.data
      authState.ready = true
    } catch (error) {
      if (!isProblem(error) || error.code !== 'UNAUTHENTICATED') throw error
      authState.user = null
      authState.ready = true
    } finally {
      currentUserRequest = undefined
    }
  })()
  return currentUserRequest
}

export async function login(username: string, password: string): Promise<void> {
  const response = await createSession({ body: { username, password } })
  authState.user = response.data.user
  authState.ready = true
}

export async function register(username: string, password: string): Promise<void> {
  const response = await registerUser({ body: { username, password } })
  authState.user = response.data.user
  authState.ready = true
}

export async function logout(): Promise<void> {
  await deleteCurrentSession()
  authState.user = null
  authState.ready = true
}
