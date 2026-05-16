// Tests for the E2E crypto primitives. These are the security-critical
// properties — if any of these break, scheme=2 messaging silently breaks
// (multi-device sync, inter-user encryption, vault-from-password recovery).
// All tests use pure WebCrypto + @noble/curves, no DOM or HTTP.

import { describe, expect, it } from 'vitest'

import {
  decryptMessage,
  deriveChatKey,
  encryptForGroup,
  encryptMessage,
  isE2EAvailable,
  openVault,
  publicKeyBase64,
  rewrapAccountKey,
  setupNewVault,
} from './e2e'

describe('e2e: runtime availability', () => {
  it('reports WebCrypto as available in the test environment', () => {
    // Node 18+ ships WebCrypto. If this fails the rest of the suite will too.
    expect(isE2EAvailable()).toBe(true)
  })
})

describe('e2e: vault roundtrip (password ↔ account_key)', () => {
  it('setupNewVault + openVault with the same password recovers the same account_key', async () => {
    // Multi-device's whole premise: a second device with the same password
    // unwraps the same account_key from the server-stored blob.
    const password = 'correct horse battery staple'
    const { accountKey: original, vault } = await setupNewVault(password)

    const recovered = await openVault(password, vault)

    // CryptoKey objects don't have value equality. Export both and byte-compare.
    const a = new Uint8Array(await crypto.subtle.exportKey('raw', original))
    const b = new Uint8Array(await crypto.subtle.exportKey('raw', recovered))
    expect(b).toEqual(a)
  })

  it('openVault with the wrong password rejects', async () => {
    // AES-GCM auth tag mismatches on a bad key. The frontend treats this as
    // "user typed the wrong password" rather than as data corruption — so the
    // promise rejecting is the contract we depend on.
    const { vault } = await setupNewVault('correct password')

    await expect(openVault('wrong password', vault)).rejects.toBeDefined()
  })

  it('different passwords produce different vaults (no salt reuse)', async () => {
    // If two onboardings somehow reused the same salt, two users with the same
    // password would have identical wrapped blobs. Salt is random per setup.
    const a = await setupNewVault('p')
    const b = await setupNewVault('p')

    expect(a.vault.vaultSalt).not.toEqual(b.vault.vaultSalt)
    expect(a.vault.encryptedAccountKey).not.toEqual(b.vault.encryptedAccountKey)
  })
})

describe('e2e: message encrypt/decrypt roundtrip', () => {
  it('decryptMessage(encryptMessage(m, k), k) returns the original plaintext', async () => {
    const { accountKey } = await setupNewVault('p')
    const plaintext = 'Привет, как дела? 🚀'

    const { ciphertext, iv } = await encryptMessage(plaintext, accountKey)
    const back = await decryptMessage(ciphertext, iv, accountKey)

    expect(back).toBe(plaintext)
  })

  it('encryptMessage produces a fresh random IV per call (no nonce reuse)', async () => {
    // AES-GCM is catastrophically broken under nonce reuse with the same key.
    // The IV must come from crypto.getRandomValues per message; this asserts
    // we don't accidentally cache one IV at module scope.
    const { accountKey } = await setupNewVault('p')
    const a = await encryptMessage('same plaintext', accountKey)
    const b = await encryptMessage('same plaintext', accountKey)

    expect(a.iv).not.toEqual(b.iv)
    expect(a.ciphertext).not.toEqual(b.ciphertext)
  })

  it('decryptMessage with the wrong key rejects', async () => {
    const a = await setupNewVault('alice-pass')
    const b = await setupNewVault('bob-pass')

    const { ciphertext, iv } = await encryptMessage('secret', a.accountKey)

    await expect(decryptMessage(ciphertext, iv, b.accountKey)).rejects.toBeDefined()
  })
})

describe('e2e: X25519 ECDH symmetry (inter-user E2E lynchpin)', () => {
  it('Alice and Bob derive the same chat_key from each other public_keys', async () => {
    // This is THE PROPERTY that makes scheme=2 work between two users.
    // Alice computes shared = X25519(Alice.private, Bob.public).
    // Bob computes shared = X25519(Bob.private, Alice.public).
    // Both must produce the same AES-GCM key — otherwise their ciphertexts
    // would never decrypt on the other side.
    //
    // chat_key is intentionally non-extractable, so we can't byte-compare the
    // raw keys. Instead we assert equality by the only property we care about:
    // ciphertext under Alice's key decrypts cleanly under Bob's key.
    const alice = await setupNewVault('alice')
    const bob = await setupNewVault('bob')

    const alicePub = await publicKeyBase64(alice.accountKey)
    const bobPub = await publicKeyBase64(bob.accountKey)

    const fromAlice = await deriveChatKey(alice.accountKey, bobPub)
    const fromBob = await deriveChatKey(bob.accountKey, alicePub)

    const { ciphertext, iv } = await encryptMessage('symmetry probe', fromAlice)
    const back = await decryptMessage(ciphertext, iv, fromBob)
    expect(back).toBe('symmetry probe')
  })

  it('Alice→Bob message decrypts on Bob with chat_key derived from Alice.public', async () => {
    // End-to-end happy path: encrypt with the chat_key Alice computed, then
    // decrypt with the chat_key Bob would compute. Pulls together everything
    // the send/receive paths in ChatWindow + ChatsPage rely on.
    const alice = await setupNewVault('alice')
    const bob = await setupNewVault('bob')

    const alicePub = await publicKeyBase64(alice.accountKey)
    const bobPub = await publicKeyBase64(bob.accountKey)

    const aliceSendKey = await deriveChatKey(alice.accountKey, bobPub)
    const { ciphertext, iv } = await encryptMessage('hi bob', aliceSendKey)

    const bobReceiveKey = await deriveChatKey(bob.accountKey, alicePub)
    const recovered = await decryptMessage(ciphertext, iv, bobReceiveKey)

    expect(recovered).toBe('hi bob')
  })

  it('publicKeyBase64 is deterministic — same account_key → same public_key', async () => {
    // Multi-device requires: two devices both derive the X25519 keypair from
    // the same account_key. Same input → same output, every time.
    const { accountKey } = await setupNewVault('p')

    const first = await publicKeyBase64(accountKey)
    const second = await publicKeyBase64(accountKey)

    expect(second).toBe(first)
  })

  it('different account_keys produce different public_keys', async () => {
    // Sanity — different users genuinely have different public keys, so the
    // derivation depends on the input (not a constant or attacker-controllable).
    const a = await setupNewVault('alice')
    const b = await setupNewVault('bob')

    const aPub = await publicKeyBase64(a.accountKey)
    const bPub = await publicKeyBase64(b.accountKey)

    expect(bPub).not.toBe(aPub)
  })
})

describe('e2e: group pairwise (encryptForGroup)', () => {
  it('produces one envelope per recipient, each decryptable by that recipient', async () => {
    // The defining property of Variant A: the sender encrypts the same plaintext
    // once per recipient using ECDH(self, recipient.public). Each recipient
    // independently arrives at the same chat_key via ECDH(self, sender.public)
    // and decrypts their own envelope.
    const alice = await setupNewVault('alice')
    const bob = await setupNewVault('bob')
    const carol = await setupNewVault('carol')

    const alicePub = await publicKeyBase64(alice.accountKey)
    const bobPub = await publicKeyBase64(bob.accountKey)
    const carolPub = await publicKeyBase64(carol.accountKey)

    const plaintext = 'привет всем 👋'
    const envelopes = await encryptForGroup(
      plaintext,
      [
        { userID: 1, publicKey: alicePub }, // sender's own echo envelope
        { userID: 2, publicKey: bobPub },
        { userID: 3, publicKey: carolPub },
      ],
      alice.accountKey,
    )

    expect(envelopes).toHaveLength(3)

    const findEnv = (recipientId: number) => {
      const env = envelopes.find((e) => e.recipient_id === recipientId)
      if (!env) throw new Error(`no envelope for recipient ${recipientId}`)
      return env
    }

    // Bob decrypts his envelope using deriveChatKey(self, alice.public).
    const bobEnv = findEnv(2)
    const bobKey = await deriveChatKey(bob.accountKey, alicePub)
    expect(await decryptMessage(bobEnv.ciphertext, bobEnv.iv, bobKey)).toBe(plaintext)

    // Carol decrypts her envelope the same way.
    const carolEnv = findEnv(3)
    const carolKey = await deriveChatKey(carol.accountKey, alicePub)
    expect(await decryptMessage(carolEnv.ciphertext, carolEnv.iv, carolKey)).toBe(plaintext)

    // Alice can decrypt her own echo envelope (ECDH-with-self is well-defined for X25519).
    const aliceEnv = findEnv(1)
    const aliceKey = await deriveChatKey(alice.accountKey, alicePub)
    expect(await decryptMessage(aliceEnv.ciphertext, aliceEnv.iv, aliceKey)).toBe(plaintext)
  })

  it('each envelope carries a distinct IV — no per-recipient nonce reuse', async () => {
    // Catastrophic if the same IV were reused: AES-GCM with key reuse + IV reuse
    // leaks the plaintext XOR and the auth key. encryptMessage pulls a fresh IV
    // per call, so independent envelopes must also have distinct IVs.
    const alice = await setupNewVault('alice')
    const bob = await setupNewVault('bob')
    const carol = await setupNewVault('carol')

    const envelopes = await encryptForGroup(
      'same plaintext',
      [
        { userID: 2, publicKey: await publicKeyBase64(bob.accountKey) },
        { userID: 3, publicKey: await publicKeyBase64(carol.accountKey) },
      ],
      alice.accountKey,
    )

    expect(envelopes[0].iv).not.toBe(envelopes[1].iv)
    expect(envelopes[0].ciphertext).not.toBe(envelopes[1].ciphertext)
  })

  it('throws if any recipient lacks a public_key', async () => {
    // Strict mode (current invariant): caller's UI must verify every member
    // has a public_key via getGroupKeyStatus before calling encryptForGroup.
    // Reaching this function with an empty key is a programmer error —
    // surface it loudly rather than partially encrypting.
    const alice = await setupNewVault('alice')
    const bob = await setupNewVault('bob')

    await expect(
      encryptForGroup(
        'hi',
        [
          { userID: 2, publicKey: await publicKeyBase64(bob.accountKey) },
          { userID: 3, publicKey: '' }, // Carol hasn't onboarded
        ],
        alice.accountKey,
      ),
    ).rejects.toThrow(/public_key/)
  })

  it('a non-recipient cannot decrypt any envelope', async () => {
    // Confidentiality: Mallory holds a valid account_key but isn't in the
    // recipient list. None of the envelopes' chat_keys derive from her
    // public, so her ECDH against the sender produces a different key and
    // AES-GCM rejects the ciphertext.
    const alice = await setupNewVault('alice')
    const bob = await setupNewVault('bob')
    const mallory = await setupNewVault('mallory')

    const alicePub = await publicKeyBase64(alice.accountKey)
    const bobPub = await publicKeyBase64(bob.accountKey)

    const envelopes = await encryptForGroup(
      'secret',
      [{ userID: 2, publicKey: bobPub }],
      alice.accountKey,
    )

    const malloryKey = await deriveChatKey(mallory.accountKey, alicePub)
    await expect(
      decryptMessage(envelopes[0].ciphertext, envelopes[0].iv, malloryKey),
    ).rejects.toBeDefined()
  })
})

describe('e2e: password rotation (rewrapAccountKey)', () => {
  it('keeps the same account_key after rotation — existing messages stay readable', async () => {
    // The whole point of rewrap-on-password-change: the account_key (and
    // therefore the X25519 keypair, and therefore every existing chat_key)
    // does NOT change. Only the wrapper does. If account_key changed, every
    // historical scheme=2 message would become unreadable.
    const { accountKey: original, vault: oldVault } = await setupNewVault('old-pass')

    const newVault = await rewrapAccountKey(original, 'new-pass')

    // Old password no longer unwraps the new vault — sanity check.
    await expect(openVault('old-pass', newVault)).rejects.toBeDefined()
    // New password unwraps it.
    const rotated = await openVault('new-pass', newVault)

    const a = new Uint8Array(await crypto.subtle.exportKey('raw', original))
    const b = new Uint8Array(await crypto.subtle.exportKey('raw', rotated))
    expect(b).toEqual(a)

    // The wrapper changed — different salts, different ciphertexts.
    expect(newVault.vaultSalt).not.toBe(oldVault.vaultSalt)
    expect(newVault.encryptedAccountKey).not.toBe(oldVault.encryptedAccountKey)
  })
})
