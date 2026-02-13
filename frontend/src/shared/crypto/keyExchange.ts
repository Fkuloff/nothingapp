import { httpGet, httpPut } from '../api/httpClient'
import { generateKeyPair, exportKey, importPublicKey, importPrivateKey, deriveSharedKey } from './keys'
import { savePrivateKey, savePublicKey, getPrivateKey, getPublicKey, getChatKey, saveChatKey, hasIdentityKeys } from './keyStore'
import { arrayBufferToBase64, base64ToArrayBuffer } from './encryption'

// --- Public Key Exchange ---

async function uploadPublicKeyToServer(publicKey: JsonWebKey): Promise<void> {
  await httpPut('/api/keys', { public_key: JSON.stringify(publicKey) })
}

export async function fetchPublicKey(userId: number): Promise<JsonWebKey | null> {
  try {
    const res = await httpGet<{ user_id: number; public_key: string }>(`/api/keys/${userId}`)
    return JSON.parse(res.public_key) as JsonWebKey
  } catch {
    return null
  }
}

// --- Chat Key Derivation ---

export async function getOrDeriveChatKey(chatId: number, otherUserId: number): Promise<CryptoKey | null> {
  const cached = await getChatKey(chatId)
  if (cached) return cached

  const myPrivateKey = await getPrivateKey()
  if (!myPrivateKey) return null

  const theirJwk = await fetchPublicKey(otherUserId)
  if (!theirJwk) return null

  const theirPublicKey = await importPublicKey(theirJwk)
  const sharedKey = await deriveSharedKey(myPrivateKey, theirPublicKey)
  await saveChatKey(chatId, sharedKey)
  return sharedKey
}

// --- Key Initialization ---

export type KeyInitResult = {
  ready: boolean
  needsRestore: boolean
  needsBackupFirst: boolean
}

export async function initializeKeys(userId?: number): Promise<KeyInitResult> {
  if (await hasIdentityKeys()) {
    const pub = await getPublicKey()
    if (pub) {
      await uploadPublicKeyToServer(await exportKey(pub)).catch(() => {})
    }
    return { ready: true, needsRestore: false, needsBackupFirst: false }
  }

  // Check if backup exists on server
  try {
    await httpGet('/api/keys/backup')
    return { ready: false, needsRestore: true, needsBackupFirst: false }
  } catch {
    // No backup — check if this user already has keys on another device
    if (userId) {
      const existing = await fetchPublicKey(userId)
      if (existing) {
        return { ready: false, needsRestore: false, needsBackupFirst: true }
      }
    }

    // Truly first time — generate new keys
    const keyPair = await generateKeyPair()
    await savePrivateKey(keyPair.privateKey)
    await savePublicKey(keyPair.publicKey)
    await uploadPublicKeyToServer(await exportKey(keyPair.publicKey))
    return { ready: true, needsRestore: false, needsBackupFirst: false }
  }
}

// --- Key Backup ---

const PBKDF2_ITERATIONS = 100_000

async function derivePbkdf2Key(password: string, salt: BufferSource, usage: KeyUsage): Promise<CryptoKey> {
  const raw = await crypto.subtle.importKey('raw', new TextEncoder().encode(password), 'PBKDF2', false, ['deriveKey'])
  return crypto.subtle.deriveKey(
    { name: 'PBKDF2', salt, iterations: PBKDF2_ITERATIONS, hash: 'SHA-256' },
    raw,
    { name: 'AES-GCM', length: 256 },
    false,
    [usage],
  )
}

export async function backupPrivateKey(password: string): Promise<void> {
  const privateKey = await getPrivateKey()
  if (!privateKey) throw new Error('No private key to backup')

  const keyData = new TextEncoder().encode(JSON.stringify(await exportKey(privateKey)))
  const salt = crypto.getRandomValues(new Uint8Array(16))
  const iv = crypto.getRandomValues(new Uint8Array(12))
  const wrappingKey = await derivePbkdf2Key(password, salt, 'encrypt')
  const encrypted = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, wrappingKey, keyData)

  await httpPut('/api/keys/backup', {
    encrypted_key: arrayBufferToBase64(encrypted),
    salt: arrayBufferToBase64(salt),
    iv: arrayBufferToBase64(iv),
  })
}

export async function restorePrivateKey(password: string): Promise<boolean> {
  let backup: { encrypted_key: string; salt: string; iv: string }
  try {
    backup = await httpGet('/api/keys/backup')
  } catch {
    return false
  }

  try {
    const salt = base64ToArrayBuffer(backup.salt)
    const iv = base64ToArrayBuffer(backup.iv)
    const unwrappingKey = await derivePbkdf2Key(password, salt, 'decrypt')
    const decrypted = await crypto.subtle.decrypt({ name: 'AES-GCM', iv }, unwrappingKey, base64ToArrayBuffer(backup.encrypted_key))

    const privateKeyJwk = JSON.parse(new TextDecoder().decode(decrypted)) as JsonWebKey
    await savePrivateKey(await importPrivateKey(privateKeyJwk))

    // Derive public key from private JWK
    const publicKeyJwk: JsonWebKey = { ...privateKeyJwk, d: undefined, key_ops: [] }
    await savePublicKey(await importPublicKey(publicKeyJwk))

    return true
  } catch {
    return false
  }
}
