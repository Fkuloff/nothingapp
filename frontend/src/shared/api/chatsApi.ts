import type { ChatListResponse, ChatItem, MessagesResponse, Message, WSEvent } from './types'
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

// Convert a WebSocket event to a partial Message structure
export function wsEventToMessage(event: WSEvent): Partial<Message> | null {
  if ('error' in event) {
    console.error('WebSocket error:', event.error)
    return null
  }

  switch (event.action) {
    case 'new':
      return {
        id: event.id,
        chat_id: event.chat_id,
        user_id: event.user_id,
        text: event.text,
        reply_to_id: event.reply_to_id,
        edited_at: event.edited_at,
        is_deleted: event.is_deleted,
        created_at: event.created_at,
        attachments: [],
      }
    case 'edit':
      return {
        id: event.id,
        text: event.text,
        edited_at: event.edited_at || new Date().toISOString(),
      }
    case 'delete':
      return {
        id: event.id,
        is_deleted: event.is_deleted,
      }
    default:
      return null
  }
}
