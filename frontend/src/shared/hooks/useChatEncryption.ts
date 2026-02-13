import { useCallback } from 'react'
import { encryptText, encryptFile } from '../crypto'
import { getOrDeriveChatKey } from '../crypto/keyExchange'
import { hasIdentityKeys } from '../crypto/keyStore'
import type { WSMessageAction } from '../api/types'

/**
 * Hook for E2E encryption of outgoing messages and files.
 * Throws on failure — unencrypted data is never sent.
 */
export function useChatEncryption(otherUserId: number | undefined) {
  const requireKey = useCallback(
    async (chatId: number): Promise<CryptoKey> => {
      if (!(await hasIdentityKeys())) throw new Error('E2E_NO_KEYS')
      if (!otherUserId) throw new Error('E2E_NO_CHAT_KEY')
      const key = await getOrDeriveChatKey(chatId, otherUserId)
      if (!key) throw new Error('E2E_NO_CHAT_KEY')
      return key
    },
    [otherUserId],
  )

  const encryptAction = useCallback(
    async (data: WSMessageAction): Promise<WSMessageAction> => {
      if (data.action !== 'send' && data.action !== 'edit') return data
      if (!data.text) return data

      const key = await requireKey(data.chat_id)
      const { ciphertext, iv } = await encryptText(data.text, key)
      return { ...data, text: ciphertext, iv }
    },
    [requireKey],
  )

  const encryptFileForUpload = useCallback(
    async (
      file: File,
      chatId: number,
    ): Promise<{ blob: Blob; iv: string; originalType: string; originalName: string }> => {
      const key = await requireKey(chatId)
      const { encrypted, iv } = await encryptFile(await file.arrayBuffer(), key)

      return {
        blob: new Blob([encrypted], { type: 'application/octet-stream' }),
        iv,
        originalType: file.type,
        originalName: file.name,
      }
    },
    [requireKey],
  )

  return { encryptAction, encryptFileForUpload }
}
