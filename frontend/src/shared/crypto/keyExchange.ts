// Key exchange with the server and chat key derivation

import { httpGet, httpPut } from '../api/httpClient'
import {
  generateKeyPair,
  exportPublicKey,
  importPublicKey,
  importPrivateKey as importPrivateKeyFromJwk,
  exportPrivateKey as exportPrivateKeyToJwk,
  deriveSharedKey,
} from './keys'
import {
  savePrivateKey,
  savePublicKey,
  getPrivateKey,
  getChatKey,
  saveChatKey,
  hasIdentityKeys,
} from './keyStore'
import { arrayBufferToBase64, base64ToArrayBuffer } from './encryption'

// --- Public Key Exchange ---

/** Upload the user's public key to the server */
export async function uploadPublicKeyToServer(publicKey: JsonWebKey): Promise<void> {
  await httpPut('/api/keys', { public_key: JSON.stringify(publicKey) })
}

/** Fetch another user's public key from the server */
export async function fetchPublicKey(
  userId: number,
): Promise<JsonWebKey | null> {
  try {
    const response = await httpGet<{ user_id: number; public_key: string }>(
      `/api/keys/${userId}`,
    )
    return JSON.parse(response.public_key) as JsonWebKey
  } catch {
    return null
  }
}

// --- Chat Key Derivation ---

/**
 * Get or derive the AES-256-GCM key for a specific chat.
 * Checks IndexedDB cache first, then derives from ECDH if needed.
 */
export async function getOrDeriveChatKey(
  chatId: number,
  otherUserId: number,
): Promise<CryptoKey | null> {
  // Check cache first
  const cached = await getChatKey(chatId)
  if (cached) return cached

  // Get my private key
  const myPrivateKey = await getPrivateKey()
  if (!myPrivateKey) return null

  // Fetch the other user's public key
  const theirPublicKeyJwk = await fetchPublicKey(otherUserId)
  if (!theirPublicKeyJwk) return null

  // Import and derive
  const theirPublicKey = await importPublicKey(theirPublicKeyJwk)
  const sharedKey = await deriveSharedKey(myPrivateKey, theirPublicKey)

  // Cache for future use
  await saveChatKey(chatId, sharedKey)

  return sharedKey
}

// --- Key Initialization ---

/**
 * Initialize E2E encryption keys.
 * - If keys exist in IndexedDB, use them.
 * - If no keys exist, check server for backup.
 * - If no backup exists, generate new keys.
 *
 * Returns true if keys are ready, false if backup restore is needed (user must enter password).
 */
export async function initializeKeys(): Promise<{
  ready: boolean
  needsRestore: boolean
}> {
  if (await hasIdentityKeys()) {
    return { ready: true, needsRestore: false }
  }

  // Check if there's a backup on the server
  try {
    await httpGet('/api/keys/backup')
    // Backup exists — user needs to enter password to restore
    return { ready: false, needsRestore: true }
  } catch {
    // No backup — generate new keys
    const keyPair = await generateKeyPair()
    await savePrivateKey(keyPair.privateKey)
    await savePublicKey(keyPair.publicKey)

    // Upload public key to server
    const publicKeyJwk = await exportPublicKey(keyPair.publicKey)
    await uploadPublicKeyToServer(publicKeyJwk)

    return { ready: true, needsRestore: false }
  }
}

// --- Key Backup (Multi-Device) ---

const PBKDF2_ITERATIONS = 100_000

/**
 * Create an encrypted backup of the private key and upload to server.
 * The key is encrypted with a password-derived key (PBKDF2 + AES-GCM).
 */
export async function backupPrivateKey(password: string): Promise<void> {
  const privateKey = await getPrivateKey()
  if (!privateKey) throw new Error('No private key to backup')

  // Export private key as JWK
  const privateKeyJwk = await exportPrivateKeyToJwk(privateKey)
  const keyData = new TextEncoder().encode(JSON.stringify(privateKeyJwk))

  // Derive wrapping key from password
  const salt = crypto.getRandomValues(new Uint8Array(16))
  const passwordKey = await crypto.subtle.importKey(
    'raw',
    new TextEncoder().encode(password),
    'PBKDF2',
    false,
    ['deriveKey'],
  )

  const wrappingKey = await crypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt,
      iterations: PBKDF2_ITERATIONS,
      hash: 'SHA-256',
    },
    passwordKey,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt'],
  )

  // Encrypt
  const iv = crypto.getRandomValues(new Uint8Array(12))
  const encrypted = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, wrappingKey, keyData)

  // Upload to server
  await httpPut('/api/keys/backup', {
    encrypted_key: arrayBufferToBase64(encrypted),
    salt: arrayBufferToBase64(salt),
    iv: arrayBufferToBase64(iv),
  })
}

/**
 * Restore private key from server backup using password.
 */
export async function restorePrivateKey(password: string): Promise<boolean> {
  // Fetch encrypted backup from server
  let backup: { encrypted_key: string; salt: string; iv: string }
  try {
    backup = await httpGet('/api/keys/backup')
  } catch {
    return false
  }

  // Derive unwrapping key from password
  const salt = base64ToArrayBuffer(backup.salt)
  const passwordKey = await crypto.subtle.importKey(
    'raw',
    new TextEncoder().encode(password),
    'PBKDF2',
    false,
    ['deriveKey'],
  )

  const unwrappingKey = await crypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt,
      iterations: PBKDF2_ITERATIONS,
      hash: 'SHA-256',
    },
    passwordKey,
    { name: 'AES-GCM', length: 256 },
    false,
    ['decrypt'],
  )

  // Decrypt
  try {
    const iv = base64ToArrayBuffer(backup.iv)
    const encryptedData = base64ToArrayBuffer(backup.encrypted_key)
    const decrypted = await crypto.subtle.decrypt(
      { name: 'AES-GCM', iv },
      unwrappingKey,
      encryptedData,
    )

    // Parse and import private key
    const privateKeyJwk = JSON.parse(new TextDecoder().decode(decrypted)) as JsonWebKey
    const privateKey = await importPrivateKeyFromJwk(privateKeyJwk)
    await savePrivateKey(privateKey)

    // Also generate/save the public key from the private key JWK
    const publicKeyJwk: JsonWebKey = { ...privateKeyJwk }
    delete publicKeyJwk.d // Remove private component
    publicKeyJwk.key_ops = []
    const publicKey = await importPublicKey(publicKeyJwk)
    await savePublicKey(publicKey)

    return true
  } catch {
    // Wrong password or corrupted backup
    return false
  }
}
