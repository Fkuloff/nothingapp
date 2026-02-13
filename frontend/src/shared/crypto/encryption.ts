// AES-256-GCM encryption/decryption for messages and files

const IV_LENGTH = 12 // 96 bits — recommended for AES-GCM

/**
 * Encrypt text with AES-256-GCM.
 * Returns base64-encoded ciphertext and IV.
 */
export async function encryptText(
  text: string,
  key: CryptoKey,
): Promise<{ ciphertext: string; iv: string }> {
  const iv = crypto.getRandomValues(new Uint8Array(IV_LENGTH))
  const encoded = new TextEncoder().encode(text)

  const encrypted = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, encoded)

  return {
    ciphertext: arrayBufferToBase64(encrypted),
    iv: arrayBufferToBase64(iv),
  }
}

/**
 * Decrypt text with AES-256-GCM.
 */
export async function decryptText(
  ciphertext: string,
  iv: string,
  key: CryptoKey,
): Promise<string> {
  const encryptedData = base64ToArrayBuffer(ciphertext)
  const ivData = base64ToArrayBuffer(iv)

  const decrypted = await crypto.subtle.decrypt({ name: 'AES-GCM', iv: ivData }, key, encryptedData)

  return new TextDecoder().decode(decrypted)
}

/**
 * Encrypt a file (ArrayBuffer) with AES-256-GCM.
 * Returns encrypted data and IV.
 */
export async function encryptFile(
  data: ArrayBuffer,
  key: CryptoKey,
): Promise<{ encrypted: ArrayBuffer; iv: string }> {
  const iv = crypto.getRandomValues(new Uint8Array(IV_LENGTH))

  const encrypted = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, data)

  return {
    encrypted,
    iv: arrayBufferToBase64(iv),
  }
}

/**
 * Decrypt a file (ArrayBuffer) with AES-256-GCM.
 */
export async function decryptFile(
  data: ArrayBuffer,
  iv: string,
  key: CryptoKey,
): Promise<ArrayBuffer> {
  const ivData = base64ToArrayBuffer(iv)
  return crypto.subtle.decrypt({ name: 'AES-GCM', iv: ivData }, key, data)
}

// --- Base64 Helpers ---

export function arrayBufferToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer)
  let binary = ''
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i])
  }
  return btoa(binary)
}

export function base64ToArrayBuffer(base64: string): ArrayBuffer {
  const binary = atob(base64)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i)
  }
  return bytes.buffer
}
