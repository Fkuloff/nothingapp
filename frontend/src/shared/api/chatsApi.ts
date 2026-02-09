import type { ChatListResponse, ChatItem, MessagesResponse, Message } from './types'
import { httpGet, httpPost } from './httpClient'
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

// Create a chat with another user by username
export async function createChat(otherUsername: string): Promise<void> {
  await httpPost(endpoints.chats.create, { other_username: otherUsername })
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
