// API Response types according to API_FULL_DOCUMENTATION.md

// Auth responses
export type AuthRegisterResponse = {
  user_id: number
  username: string
  name: string
  token: string
}

export type AuthLoginResponse = {
  user_id: number
  username: string
  name: string
  token: string
}

export type AuthMeResponse = {
  id: number
  username: string
  name: string
  phone: string
  avatar_url?: string
}

// User Profile
export type UserProfile = {
  id: number
  username: string
  name: string
  phone: string
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

export type AttachmentsUploadResponse = {
  success: boolean
  attachments: Attachment[]
}

// Contacts
export type ContactsResponse = {
  contacts: Contact[]
}

export type Contact = {
  id: number
  user_id: number
  contact_user_id: number
  created_at: string
}

// Avatar
export type AvatarUploadResponse = {
  success: boolean
  avatar_url: string
}

// Generic API responses
export type ApiSuccess = {
  success?: boolean
  message?: string
}

export type ApiError = {
  error: string
}

// WebSocket message types (client → server)
export type WSMessageSend = {
  action: 'send'
  text: string
  reply_to_id?: number
}

export type WSMessageEdit = {
  action: 'edit'
  message_id: number
  text: string
}

export type WSMessageDelete = {
  action: 'delete'
  message_id: number
}

export type WSMessageAction = WSMessageSend | WSMessageEdit | WSMessageDelete

// WebSocket events (server → client)
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
  text: string
  edited_at?: string
}

export type WSEventDelete = {
  action: 'delete'
  id: number
  is_deleted: boolean
}

export type WSEvent = WSEventNew | WSEventEdit | WSEventDelete | ApiError

// Legacy types for compatibility
export type ContactSummary = {
  id: number
  username: string
  name?: string
}

export type ChatSummary = {
  id: number
  created_at: string
  updated_at: string
  user1_id: number
  user2_id: number
  user1: UserProfile
  user2: UserProfile
  messages?: Message[]
}

