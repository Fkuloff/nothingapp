import { httpGet } from './httpClient'
import { endpoints } from './endpoints'

export type UserPresenceResponse = {
  user_id: number
  is_online: boolean
  last_seen: string
}

export async function getUserPresence(userId: number): Promise<UserPresenceResponse> {
  return httpGet<UserPresenceResponse>(endpoints.presence.get(userId))
}
