// Common emojis for the emoji picker
export const EMOJIS = ['😀', '😁', '😂', '😉', '😍', '😎', '🤔', '😭', '🔥', '💯', '👍', '🙏'] as const

// File type icons
export const FILE_TYPE_ICONS = {
  image: '🖼️',
  video: '🎬',
  audio: '🎵',
  pdf: '📕',
  document: '📄',
  default: '📎',
} as const

// Allowed image MIME types
export const ALLOWED_IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'] as const

// WebSocket reconnection
export const MAX_RECONNECT_ATTEMPTS = 5
