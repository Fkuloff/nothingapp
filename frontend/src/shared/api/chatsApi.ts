import { endpoints } from './endpoints'
import { httpDelete,httpGet, httpPost } from './httpClient'
import type { ChatCreateResponse,ChatItem, ChatListResponse, Message, MessagesResponse, PinnedMessage, WSEnvelope } from './types'

// Fetch chats of the current user
export async function getCurrentUserChats(): Promise<ChatItem[]> {
  const response = await httpGet<ChatListResponse>(endpoints.chats.list)
  return response.chats || []
}

// Create a chat with another user by user ID
export async function createChat(otherUserId: number): Promise<ChatCreateResponse> {
  return httpPost<ChatCreateResponse>(endpoints.chats.create, { other_user_id: otherUserId })
}

// Delete a chat
export async function deleteChat(chatId: number): Promise<void> {
  await httpDelete(endpoints.chats.delete(chatId))
}

// Clear all messages in a chat
export async function clearChat(chatId: number): Promise<void> {
  await httpPost(endpoints.chats.clear(chatId), {})
}

// Receipt pointers returned alongside a 1-on-1 chat's messages: the highest
// message ids the peer has been delivered / has read. Both 0 for groups or when
// the peer has no receipt yet.
type ChatMessagesResult = {
  messages: Message[]
  lastDelivered: number
  lastRead: number
}

// Fetch messages of a chat (+ the peer's read-receipt pointers for 1-on-1)
export async function getChatMessages(chatId: number): Promise<ChatMessagesResult> {
  const response = await httpGet<MessagesResponse & { last_delivered?: number; last_read?: number }>(
    endpoints.chats.messages(chatId),
  )
  return {
    messages: response.messages || [],
    lastDelivered: response.last_delivered ?? 0,
    lastRead: response.last_read ?? 0,
  }
}

// One attachment to forward: the source attachment id + the file_key re-wrapped
// (client-side) for the destination chat's recipients. The encrypted body is
// server-copied, not re-uploaded.
export type ForwardAttachmentPayload = {
  source_attachment_id: number
  envelopes: { recipient_id: number; encrypted_file_key: string; iv: string }[]
}

// Body of POST /api/chats/:id/forward. Text is a scheme=2 payload re-encrypted
// for the destination (1-on-1: text+iv; group: envelopes).
export type ForwardPayload = {
  text: string
  iv?: string
  scheme: number
  envelopes?: WSEnvelope[]
  forwarded_from_user_id?: number
  attachments?: ForwardAttachmentPayload[]
}

// Forward a message (text and/or attachments) into another chat. The server
// re-creates the message + server-copies attachment bodies; it never sees
// plaintext or any file_key (envelopes are re-wrapped client-side).
export async function forwardMessage(chatId: number, payload: ForwardPayload): Promise<void> {
  await httpPost(endpoints.chats.forward(chatId), payload)
}

// Pin a message in a chat
export async function pinMessage(chatId: number, messageId: number): Promise<void> {
  await httpPost(endpoints.chats.pin(chatId, messageId), {})
}

// Unpin a message from a chat
export async function unpinMessage(chatId: number, messageId: number): Promise<void> {
  await httpDelete(endpoints.chats.pin(chatId, messageId))
}

// Fetch pinned messages for a chat
export async function getPinnedMessages(chatId: number): Promise<PinnedMessage[]> {
  const response = await httpGet<{ pinned_messages: PinnedMessage[] }>(endpoints.chats.pins(chatId))
  return response.pinned_messages || []
}
