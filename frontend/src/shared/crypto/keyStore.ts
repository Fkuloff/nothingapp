// IndexedDB-based storage for cryptographic keys

const DB_NAME = 'messenger-crypto'
const DB_VERSION = 1
const KEYS_STORE = 'keys'
const CHAT_KEYS_STORE = 'chat-keys'

const PRIVATE_KEY_ID = 'identity-private'
const PUBLIC_KEY_ID = 'identity-public'

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION)

    request.onerror = () => reject(request.error)
    request.onsuccess = () => resolve(request.result)

    request.onupgradeneeded = () => {
      const db = request.result
      if (!db.objectStoreNames.contains(KEYS_STORE)) {
        db.createObjectStore(KEYS_STORE, { keyPath: 'id' })
      }
      if (!db.objectStoreNames.contains(CHAT_KEYS_STORE)) {
        db.createObjectStore(CHAT_KEYS_STORE, { keyPath: 'chatId' })
      }
    }
  })
}

function dbPut(storeName: string, value: unknown): Promise<void> {
  return new Promise(async (resolve, reject) => {
    const db = await openDB()
    const tx = db.transaction(storeName, 'readwrite')
    const store = tx.objectStore(storeName)
    const request = store.put(value)
    request.onerror = () => reject(request.error)
    request.onsuccess = () => resolve()
    tx.oncomplete = () => db.close()
  })
}

function dbGet<T>(storeName: string, key: string | number): Promise<T | null> {
  return new Promise(async (resolve, reject) => {
    const db = await openDB()
    const tx = db.transaction(storeName, 'readonly')
    const store = tx.objectStore(storeName)
    const request = store.get(key)
    request.onerror = () => reject(request.error)
    request.onsuccess = () => resolve((request.result as T) ?? null)
    tx.oncomplete = () => db.close()
  })
}

function dbDelete(storeName: string, key: string | number): Promise<void> {
  return new Promise(async (resolve, reject) => {
    const db = await openDB()
    const tx = db.transaction(storeName, 'readwrite')
    const store = tx.objectStore(storeName)
    const request = store.delete(key)
    request.onerror = () => reject(request.error)
    request.onsuccess = () => resolve()
    tx.oncomplete = () => db.close()
  })
}

// --- Identity Key Storage ---

/** Save the user's private key to IndexedDB */
export async function savePrivateKey(key: CryptoKey): Promise<void> {
  await dbPut(KEYS_STORE, { id: PRIVATE_KEY_ID, key })
}

/** Get the user's private key from IndexedDB */
export async function getPrivateKey(): Promise<CryptoKey | null> {
  const result = await dbGet<{ id: string; key: CryptoKey }>(KEYS_STORE, PRIVATE_KEY_ID)
  return result?.key ?? null
}

/** Save the user's public key to IndexedDB */
export async function savePublicKey(key: CryptoKey): Promise<void> {
  await dbPut(KEYS_STORE, { id: PUBLIC_KEY_ID, key })
}

/** Get the user's public key from IndexedDB */
export async function getPublicKey(): Promise<CryptoKey | null> {
  const result = await dbGet<{ id: string; key: CryptoKey }>(KEYS_STORE, PUBLIC_KEY_ID)
  return result?.key ?? null
}

/** Check if identity keys exist */
export async function hasIdentityKeys(): Promise<boolean> {
  const privKey = await getPrivateKey()
  return privKey !== null
}

// --- Chat Key Cache ---

/** Cache a derived chat key */
export async function saveChatKey(chatId: number, key: CryptoKey): Promise<void> {
  await dbPut(CHAT_KEYS_STORE, { chatId, key })
}

/** Get a cached chat key */
export async function getChatKey(chatId: number): Promise<CryptoKey | null> {
  const result = await dbGet<{ chatId: number; key: CryptoKey }>(CHAT_KEYS_STORE, chatId)
  return result?.key ?? null
}

/** Remove a cached chat key */
export async function removeChatKey(chatId: number): Promise<void> {
  await dbDelete(CHAT_KEYS_STORE, chatId)
}

// --- Cleanup ---

/** Clear all crypto data (used on logout) */
export async function clearAllCryptoData(): Promise<void> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.deleteDatabase(DB_NAME)
    request.onerror = () => reject(request.error)
    request.onsuccess = () => resolve()
  })
}
