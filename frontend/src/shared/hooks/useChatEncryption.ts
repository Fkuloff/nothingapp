import { useCallback, useRef } from 'react'
import { encryptText, decryptText, encryptFile } from '../crypto'
import { getOrDeriveChatKey } from '../crypto/keyExchange'
import { hasIdentityKeys } from '../crypto/keyStore'
import type { Message, WSMessageAction, WSEventNew, WSEventEdit } from '../api/types'

/**
 * Hook that provides E2E encryption/decryption for chat messages.
 * Wraps the WebSocket send function to encrypt outgoing messages,
 * and provides helpers to decrypt incoming messages.
 */
export function useChatEncryption(otherUserId: number | undefined) {
  const keyCache = useRef<Map<number, CryptoKey>>(new Map())

  const getChatKeyForChat = useCallback(
    async (chatId: number): Promise<CryptoKey | null> => {
      if (!otherUserId) return null

      // Check in-memory cache first
      const cached = keyCache.current.get(chatId)
      if (cached) return cached

      const key = await getOrDeriveChatKey(chatId, otherUserId)
      if (key) {
        keyCache.current.set(chatId, key)
      }
      return key
    },
    [otherUserId],
  )

  /**
   * Wrap a WSMessageAction to encrypt text before sending.
   * Returns the encrypted action, or the original if encryption is unavailable.
   */
  const encryptAction = useCallback(
    async (data: WSMessageAction): Promise<WSMessageAction> => {
      if (data.action !== 'send' && data.action !== 'edit') return data
      if (!data.text || !(await hasIdentityKeys())) return data

      const chatId = data.chat_id
      const key = await getChatKeyForChat(chatId)
      if (!key) return data

      const { ciphertext, iv } = await encryptText(data.text, key)
      return { ...data, text: ciphertext, iv }
    },
    [getChatKeyForChat],
  )

  /**
   * Decrypt a single message. Returns the message with decrypted text.
   * If no IV is present (legacy/plaintext message), returns as-is.
   */
  const decryptMessage = useCallback(
    async (msg: Message, chatId: number): Promise<Message> => {
      if (!msg.iv || msg.is_deleted) return msg

      try {
        const key = await getChatKeyForChat(chatId)
        if (!key) return msg

        const plaintext = await decryptText(msg.text, msg.iv, key)
        return { ...msg, text: plaintext }
      } catch (err) {
        console.error('Failed to decrypt message:', msg.id, err)
        return { ...msg, text: '[Не удалось расшифровать]' }
      }
    },
    [getChatKeyForChat],
  )

  /**
   * Decrypt an array of messages (e.g., from API response).
   */
  const decryptMessages = useCallback(
    async (messages: Message[], chatId: number): Promise<Message[]> => {
      if (!(await hasIdentityKeys())) return messages

      return Promise.all(messages.map((msg) => decryptMessage(msg, chatId)))
    },
    [decryptMessage],
  )

  /**
   * Decrypt a WebSocket "new" or "edit" event.
   */
  const decryptWSEvent = useCallback(
    async (event: WSEventNew | WSEventEdit, chatId: number): Promise<typeof event> => {
      if (!('iv' in event) || !event.iv) return event

      try {
        const key = await getChatKeyForChat(chatId)
        if (!key) return event

        const plaintext = await decryptText(event.text, event.iv, key)
        return { ...event, text: plaintext }
      } catch (err) {
        console.error('Failed to decrypt WS event:', err)
        return { ...event, text: '[Не удалось расшифровать]' }
      }
    },
    [getChatKeyForChat],
  )

  /**
   * Encrypt a file before upload. Returns encrypted blob and metadata.
   */
  const encryptFileForUpload = useCallback(
    async (
      file: File,
      chatId: number,
    ): Promise<{ blob: Blob; iv: string; originalType: string; originalName: string } | null> => {
      if (!(await hasIdentityKeys())) return null

      const key = await getChatKeyForChat(chatId)
      if (!key) return null

      const data = await file.arrayBuffer()
      const { encrypted, iv } = await encryptFile(data, key)

      return {
        blob: new Blob([encrypted], { type: 'application/octet-stream' }),
        iv,
        originalType: file.type,
        originalName: file.name,
      }
    },
    [getChatKeyForChat],
  )

  return {
    encryptAction,
    decryptMessage,
    decryptMessages,
    decryptWSEvent,
    encryptFileForUpload,
  }
}
