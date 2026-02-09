import { FILE_TYPE_ICONS } from '../constants'

/**
 * Get an emoji icon for a file based on its MIME type
 */
export function getFileIcon(file: File): string {
  if (file.type.startsWith('image/')) return FILE_TYPE_ICONS.image
  if (file.type.startsWith('video/')) return FILE_TYPE_ICONS.video
  if (file.type.startsWith('audio/')) return FILE_TYPE_ICONS.audio
  if (file.type === 'application/pdf') return FILE_TYPE_ICONS.pdf
  return FILE_TYPE_ICONS.default
}

/**
 * Check if a file is an image
 */
export function isImageFile(file: File): boolean {
  return file.type.startsWith('image/')
}

/**
 * Format file size in human-readable format
 */
export function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
