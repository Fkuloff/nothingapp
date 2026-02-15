import { endpoints } from './endpoints'
import { httpDelete,httpGet, httpPost } from './httpClient'
import type { EnrichedContactsResponse, UserListItem, UserSearchResponse } from './types'

// Fetch current user's contacts with enriched user data
export async function getContacts(): Promise<UserListItem[]> {
  const response = await httpGet<EnrichedContactsResponse>(endpoints.contacts.list)
  return response.contacts || []
}

// Add a user to contacts
export async function addContact(userId: number): Promise<void> {
  await httpPost(endpoints.contacts.add(userId), {})
}

// Remove a user from contacts
export async function removeContact(userId: number): Promise<void> {
  await httpDelete(endpoints.contacts.remove(userId))
}

// Search users by username or name (minimum 2 characters)
export async function searchUsers(query: string): Promise<UserListItem[]> {
  if (query.length < 2) {
    return []
  }

  const response = await httpGet<UserSearchResponse>(
    `/api/users/search?q=${encodeURIComponent(query)}`
  )
  return response.users || []
}
