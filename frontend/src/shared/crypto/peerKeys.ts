import { endpoints } from '../api/endpoints'
import { httpGet } from '../api/httpClient'
import type { GroupEnvelope, GroupRecipientKey } from './e2e'
import { deriveChatKey, encryptForGroup } from './e2e'

/**
 * In-memory cache of other users' X25519 public keys, keyed by user_id.
 *
 *   - `string`  → user has E2E set up, value is the base64 public_key
 *   - `null`    → server confirmed no E2E for this user; sender should fall back to scheme=1
 *   - missing   → not yet looked up
 *
 * Promises are memoised too so concurrent send paths don't issue duplicate HTTP
 * requests. The cache is wiped on logout (via clearPeerKeys below) and never persisted
 * to disk — public keys are cheap to refetch and bounded to the chats the user touches.
 */
const cache: Map<number, string | null> = new Map()
const inflight: Map<number, Promise<string | null>> = new Map()

type ProfileResponse = { public_key?: string | null }

/**
 * Look up another user's X25519 public_key. Returns null if the user hasn't onboarded
 * into E2E yet — callers should treat that as "fall back to scheme=1 for messages to
 * this peer".
 */
export async function getPeerPublicKey(userId: number): Promise<string | null> {
  if (cache.has(userId)) return cache.get(userId) ?? null
  const existing = inflight.get(userId)
  if (existing) return existing

  const promise = (async () => {
    try {
      const profile = await httpGet<ProfileResponse>(endpoints.profile(userId))
      const key = profile.public_key ?? null
      cache.set(userId, key)
      return key
    } catch (err) {
      console.warn('failed to fetch peer public_key:', err)
      // Don't cache failures forever — caller can retry. But don't refetch within
      // the same tick either; remove the inflight entry so the next attempt is fresh.
      return null
    } finally {
      inflight.delete(userId)
    }
  })()
  inflight.set(userId, promise)
  return promise
}

/**
 * Drop the entire cache. Called on logout so a different user signing in on the same
 * device doesn't inherit a stale view of peers' keys (also keeps the in-memory
 * footprint bounded across long sessions).
 */
export function clearPeerKeyCache(): void {
  cache.clear()
  inflight.clear()
}

/**
 * Compose the full path from "the other user in a 1-on-1 chat" to "an AES-GCM
 * key shared with that user". Pass null/0 for `peerUserId` (i.e. group chats
 * or unknown peer) and the function returns null, signalling "fall back to
 * scheme=1 server-side encryption".
 *
 *   chat_key = HKDF( X25519(my_account_key.private, peer.public_key), "msg-key" )
 *
 * Both sides arrive at the same key by ECDH symmetry, so encrypt/decrypt match.
 */
export async function getChatKey(
  accountKey: CryptoKey | null,
  peerUserId: number | null | undefined,
): Promise<CryptoKey | null> {
  if (!accountKey || !peerUserId) return null
  const peerPublicKey = await getPeerPublicKey(peerUserId)
  if (!peerPublicKey) return null
  return deriveChatKey(accountKey, peerPublicKey)
}

// ---------- group pairwise (Variant A) ----------

type GroupKeysResponse = { members?: Array<{ user_id: number; public_key: string }> }

/**
 * GroupKeyStatus is the full picture of a group's E2E readiness — both the
 * recipient list (only useful if `missingUserIds` is empty) and the set of
 * members who haven't onboarded into E2E yet (caller can show their names).
 *
 * Returns null only if the request itself fails (network, 403). A group where
 * every member is missing a public_key returns a status with all userIds in
 * `missingUserIds` — not null — so the UI can render a precise "N members
 * haven't set up encryption" message.
 */
export type GroupKeyStatus = {
  recipients: GroupRecipientKey[]
  missingUserIds: number[]
}

/**
 * Fetch the X25519 public_key of every member of a group. Not cached: group
 * membership and per-user public_keys can change at any time (member
 * added/removed, password rotation), and the latency to refetch is one
 * round-trip — much cheaper than serving a stale ciphertext that one member
 * can't decrypt.
 */
export async function getGroupKeyStatus(chatId: number): Promise<GroupKeyStatus | null> {
  try {
    const resp = await httpGet<GroupKeysResponse>(endpoints.groups.keys(chatId))
    const members = resp.members ?? []
    const recipients: GroupRecipientKey[] = members.map((m) => ({
      userID: m.user_id,
      publicKey: m.public_key,
    }))
    const missingUserIds = recipients.filter((r) => !r.publicKey).map((r) => r.userID)
    return { recipients, missingUserIds }
  } catch (err) {
    console.warn('failed to fetch group keys:', err)
    return null
  }
}

/**
 * Encrypt `plaintext` for every current member of a group via pairwise X25519
 * ECDH. Throws if any member lacks a public_key — callers MUST gate this on
 * a prior getGroupKeyStatus check (UI blocks send until every member is
 * E2E-ready, which the lazy-vault modal in App.tsx guarantees for active users).
 *
 * `senderUserId` is required so we can include a self-envelope (so the sender
 * can read their own message back on another device). X25519 ECDH-with-self
 * is well-defined, deterministic, and only reproducible by the sender.
 */
export async function encryptGroupMessage(
  plaintext: string,
  status: GroupKeyStatus,
  accountKey: CryptoKey,
  senderUserId: number,
): Promise<GroupEnvelope[]> {
  if (status.missingUserIds.length > 0) {
    throw new Error(`encryptGroupMessage: ${status.missingUserIds.length} members missing public_key`)
  }
  if (!status.recipients.some((r) => r.userID === senderUserId)) {
    throw new Error('encryptGroupMessage: sender not in recipient list')
  }
  return encryptForGroup(plaintext, status.recipients, accountKey)
}
