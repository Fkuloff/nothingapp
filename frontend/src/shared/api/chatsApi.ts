import type { ChatListResponse, ChatItem, MessagesResponse, Message, ChatCreateResponse } from './types'
import { httpGet, httpPost, httpDelete } from './httpClient'
import { endpoints } from './endpoints'

// Fetch chats of the current user
export async function getCurrentUserChats(): Promise<ChatItem[]> {
  try {
    const response = await httpGet<ChatListResponse>(endpoints.chats.list)
    return response.chats || []
  } catch (error) {
    console.error('Failed to load chats:', error)
    return []
  }
}

// Create a chat with another user by user ID
export async function createChat(otherUserId: number): Promise<ChatCreateResponse> {
  return httpPost<ChatCreateResponse>(endpoints.chats.create, { other_user_id: otherUserId })
}

// Create a chat with another user by username
export async function createChatByUsername(otherUsername: string): Promise<ChatCreateResponse> {
  return httpPost<ChatCreateResponse>(endpoints.chats.create, { other_username: otherUsername })
}

// Delete a chat
export async function deleteChat(chatId: number): Promise<void> {
  await httpDelete(endpoints.chats.delete(chatId))
}

// Clear all messages in a chat
export async function clearChat(chatId: number): Promise<void> {
  await httpPost(endpoints.chats.clear(chatId), {})
}

// Fetch messages of a chat
export async function getChatMessages(chatId: number): Promise<Message[]> {
  try {
    const response = await httpGet<MessagesResponse>(endpoints.chats.messages(chatId))
    return response.messages || []
  } catch (error) {
    console.error('Failed to load messages:', error)
    return []
  }
}
