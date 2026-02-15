// API Response types according to API_FULL_DOCUMENTATION.md

// Auth responses - unified type for login/register
export type AuthResponse = {
  user_id: number
  username: string
  name: string
  token: string
}

// Aliases for backwards compatibility
export type AuthLoginResponse = AuthResponse
export type AuthRegisterResponse = AuthResponse

export type AuthMeResponse = {
  id: number
  username: string
  name: string
  avatar_url?: string
}

// User Profile
export type UserProfile = {
  id: number
  username: string
  name: string
  avatar_url?: string
  is_own?: boolean
  is_contact?: boolean
  created_at?: string
  updated_at?: string
}

// Chats
export type ChatListResponse = {
  chats: ChatItem[]
}

export type ChatItem = {
  id: number
  other_user_id: number
  other_user_name: string
  avatar_url?: string
  last_message?: string
  unread_count: number
  updated_at: string
}

export type ChatCreateResponse = {
  id: number
  user1_id: number
  user2_id: number
  created_at: string
}

// Messages
export type MessagesResponse = {
  messages: Message[]
}

export type Message = {
  id: number
  chat_id: number
  user_id: number
  text: string
  reply_to_id?: number | null
  edited_at?: string | null
  is_deleted: boolean
  created_at: string
  attachments: Attachment[]
}

// Attachments
export type Attachment = {
  id: number
  message_id: number
  file_type: 'image' | 'video' | 'document' | 'audio'
  storage_key: string
  file_name: string
  file_size: number
  mime_type: string
  thumbnail_key?: string
  width?: number
  height?: number
  duration?: number
  created_at?: string
}

// User list item (used in contacts list, search results)
export type UserListItem = {
  id: number
  username: string
  name: string
  avatar_url?: string
}

// User search response
export type UserSearchResponse = {
  users: UserListItem[]
}

// Enriched contacts response (backend returns UserListItem array)
export type EnrichedContactsResponse = {
  contacts: UserListItem[]
}

// Avatar
export type AvatarUploadResponse = {
  success: boolean
  avatar_url: string
}

// Generic API responses
export type ApiError = {
  error: string
}

// WebSocket message types (client -> server)
// All messages must include chat_id since we use a global WebSocket connection
export type WSMessageSend = {
  action: 'send'
  chat_id: number
  text: string
  reply_to_id?: number
}

export type WSMessageEdit = {
  action: 'edit'
  chat_id: number
  message_id: number
  text: string
}

export type WSMessageDelete = {
  action: 'delete'
  chat_id: number
  message_id: number
}

export type WSMessageMarkRead = {
  action: 'mark_read'
  chat_id: number
}

export type WSMessageAction = WSMessageSend | WSMessageEdit | WSMessageDelete | WSMessageMarkRead

// WebSocket events (server -> client)
export type WSEventNew = {
  action: 'new'
  id: number
  chat_id: number
  user_id: number
  text: string
  reply_to_id?: number | null
  edited_at?: string | null
  is_deleted: boolean
  created_at: string
  updated_at?: string
}

export type WSEventEdit = {
  action: 'edit'
  id: number
  chat_id: number
  text: string
  edited_at?: string
}

export type WSEventDelete = {
  action: 'delete'
  id: number
  chat_id: number
  is_deleted: boolean
}

export type WSEventPresenceChanged = {
  action: 'presence_changed'
  user_id: number
  is_online: boolean
}

export type WSEventChatCleared = {
  action: 'chat_cleared'
  chat_id: number
  user_id: number
}

export type WSEventChatDeleted = {
  action: 'chat_deleted'
  chat_id: number
  user_id: number
}

export type WSEvent = WSEventNew | WSEventEdit | WSEventDelete | WSEventPresenceChanged | WSEventChatCleared | WSEventChatDeleted | ApiError
