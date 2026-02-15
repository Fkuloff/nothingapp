import { endpoints } from './endpoints'
import { httpGet } from './httpClient'

export type UserPresenceResponse = {
  user_id: number
  is_online: boolean
  last_seen: string
}

export async function getUserPresence(userId: number): Promise<UserPresenceResponse> {
  return httpGet<UserPresenceResponse>(endpoints.presence.get(userId))
}
