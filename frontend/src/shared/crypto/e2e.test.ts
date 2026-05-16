// Tests for the E2E crypto primitives. These are the security-critical
// properties — if any of these break, scheme=2 messaging silently breaks
// (multi-device sync, inter-user encryption, vault-from-password recovery).
// All tests use pure WebCrypto + @noble/curves, no DOM or HTTP.

import { describe, expect, it } from 'vitest'

import {
  decryptFile,
  decryptMessage,
  deriveChatKey,
  encryptFile,
  encryptForGroup,
  encryptMessage,
  generateFileKey,
  isE2EAvailable,
  openVault,
  publicKeyBase64,
  rewrapAccountKey,
  setupNewVault,
  unwrapFileKey,
  wrapFileKey,
} from './e2e'
import { prepareAttachmentForUpload } from './peerKeys'

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

describe('e2e: attachment E2E (file body + wrapped file_key)', () => {
  it('encryptFile + decryptFile roundtrip preserves bytes', async () => {
    // The file_key is just a fresh AES-GCM key. We don't go through the
    // wrap/unwrap dance here — that's the next test. This one verifies the
    // raw body encrypt/decrypt is byte-identical.
    const fileKey = await generateFileKey()
    const plaintext = new TextEncoder().encode('this is a test attachment body 🧪')
    const blob = new Blob([plaintext], { type: 'text/plain' })

    const { ciphertext, iv } = await encryptFile(blob, fileKey)
    expect(ciphertext.byteLength).toBeGreaterThan(plaintext.byteLength) // AES-GCM auth tag
    expect(iv).toBeTruthy()

    const recovered = await decryptFile(ciphertext, iv, fileKey, 'text/plain')
    const recoveredBytes = new Uint8Array(await recovered.arrayBuffer())
    expect(Array.from(recoveredBytes)).toEqual(Array.from(plaintext))
    expect(recovered.type).toBe('text/plain')
  })

  it('wrapFileKey + unwrapFileKey via Alice→Bob ECDH chat_key', async () => {
    // The full path: Alice creates an attachment with a fresh file_key,
    // encrypts the body, wraps the file_key under chat_key = ECDH(alice, bob).
    // Bob receives the wrapped key, unwraps via ECDH(bob, alice), and
    // decrypts the body. The unwrapped key must produce the original bytes.
    const alice = await setupNewVault('alice')
    const bob = await setupNewVault('bob')
    const alicePub = await publicKeyBase64(alice.accountKey)
    const bobPub = await publicKeyBase64(bob.accountKey)

    const fileKey = await generateFileKey()
    const plaintext = new TextEncoder().encode('top secret attachment 🤫')
    const blob = new Blob([plaintext], { type: 'application/pdf' })
    const { ciphertext, iv } = await encryptFile(blob, fileKey)

    // Alice wraps file_key for Bob.
    const aliceToBobKey = await deriveChatKey(alice.accountKey, bobPub)
    const { encryptedFileKey, iv: wrapIv } = await wrapFileKey(fileKey, aliceToBobKey)

    // Bob unwraps with his side of the ECDH.
    const bobFromAliceKey = await deriveChatKey(bob.accountKey, alicePub)
    const unwrapped = await unwrapFileKey(encryptedFileKey, wrapIv, bobFromAliceKey)

    // Bob decrypts the body with the unwrapped key.
    const recovered = await decryptFile(ciphertext, iv, unwrapped, 'application/pdf')
    const recoveredBytes = new Uint8Array(await recovered.arrayBuffer())
    expect(Array.from(recoveredBytes)).toEqual(Array.from(plaintext))
  })

  it('Mallory cannot unwrap a file_key wrapped for Bob', async () => {
    // Confidentiality: the wrapped file_key is only readable by the intended
    // recipient. Mallory's ECDH with Alice produces a different key, AES-GCM
    // auth tag check fails on unwrap.
    const alice = await setupNewVault('alice')
    const bob = await setupNewVault('bob')
    const mallory = await setupNewVault('mallory')
    const alicePub = await publicKeyBase64(alice.accountKey)
    const bobPub = await publicKeyBase64(bob.accountKey)

    const fileKey = await generateFileKey()
    const aliceToBobKey = await deriveChatKey(alice.accountKey, bobPub)
    const { encryptedFileKey, iv } = await wrapFileKey(fileKey, aliceToBobKey)

    const malloryKey = await deriveChatKey(mallory.accountKey, alicePub)
    await expect(unwrapFileKey(encryptedFileKey, iv, malloryKey)).rejects.toBeDefined()
  })
})

describe('e2e: prepareAttachmentForUpload (high-level upload helper)', () => {
  it('produces an envelope per recipient + ciphertext decryptable by each', async () => {
    // The full upload-side ceremony: generate file_key, encrypt the body,
    // wrap file_key for every recipient. Each recipient should be able to
    // unwrap their envelope with deriveChatKey(self, sender.public) and
    // decrypt the body to the original bytes.
    const alice = await setupNewVault('alice')
    const bob = await setupNewVault('bob')
    const carol = await setupNewVault('carol')
    const alicePub = await publicKeyBase64(alice.accountKey)
    const bobPub = await publicKeyBase64(bob.accountKey)
    const carolPub = await publicKeyBase64(carol.accountKey)

    const original = new TextEncoder().encode('prep-attachment test payload')
    const file = new File([original], 'doc.bin', { type: 'application/octet-stream' })
    const prepared = await prepareAttachmentForUpload(
      file,
      [
        { userID: 1, publicKey: alicePub },
        { userID: 2, publicKey: bobPub },
        { userID: 3, publicKey: carolPub },
      ],
      alice.accountKey,
    )

    expect(prepared.envelopes).toHaveLength(3)
    expect(prepared.fileIv).toBeTruthy()
    expect(prepared.ciphertext.size).toBeGreaterThan(original.byteLength)

    const ctBytes = new Uint8Array(await prepared.ciphertext.arrayBuffer())

    // Bob unwraps his envelope and decrypts.
    const bobEnv = prepared.envelopes.find((e) => e.recipient_id === 2)
    if (!bobEnv) throw new Error('bob envelope missing')
    const bobChatKey = await deriveChatKey(bob.accountKey, alicePub)
    const bobFileKey = await unwrapFileKey(bobEnv.encrypted_file_key, bobEnv.iv, bobChatKey)
    const recoveredBlob = await decryptFile(ctBytes, prepared.fileIv, bobFileKey, 'application/octet-stream')
    const recovered = new Uint8Array(await recoveredBlob.arrayBuffer())
    expect(Array.from(recovered)).toEqual(Array.from(original))

    // Carol independently unwraps with her own ECDH.
    const carolEnv = prepared.envelopes.find((e) => e.recipient_id === 3)
    if (!carolEnv) throw new Error('carol envelope missing')
    const carolChatKey = await deriveChatKey(carol.accountKey, alicePub)
    const carolFileKey = await unwrapFileKey(carolEnv.encrypted_file_key, carolEnv.iv, carolChatKey)
    const carolBlob = await decryptFile(ctBytes, prepared.fileIv, carolFileKey, 'application/octet-stream')
    const carolBytes = new Uint8Array(await carolBlob.arrayBuffer())
    expect(Array.from(carolBytes)).toEqual(Array.from(original))
  })

  it('throws when recipients list is empty', async () => {
    // Defensive — UI should never call us with empty recipients (composer
    // disabled when e2eStatus !== 'ready'). But still surface it loudly so
    // a programmer error becomes a visible test failure not a silent skip.
    const alice = await setupNewVault('alice')
    const file = new File([new Uint8Array([1, 2, 3])], 'x.bin')
    await expect(prepareAttachmentForUpload(file, [], alice.accountKey)).rejects.toThrow(/no recipients/)
  })

  it('throws when any recipient is missing a public_key', async () => {
    // Same strict invariant as encryptForGroup — caller must gate on
    // e2eStatus = 'ready' (which itself requires every member to have an
    // uploaded public_key). Reaching this with an empty key is a bug.
    const alice = await setupNewVault('alice')
    const bob = await setupNewVault('bob')
    const file = new File([new Uint8Array([1, 2, 3])], 'x.bin')

    await expect(
      prepareAttachmentForUpload(
        file,
        [
          { userID: 2, publicKey: await publicKeyBase64(bob.accountKey) },
          { userID: 3, publicKey: '' }, // Carol hasn't onboarded
        ],
        alice.accountKey,
      ),
    ).rejects.toThrow(/public_key/)
  })

  it('produces a fresh IV per call (no reuse across uploads)', async () => {
    // AES-GCM with the same key + same IV is catastrophic. encryptFile pulls
    // a new IV from crypto.getRandomValues per call, and prepareAttachmentForUpload
    // pulls a fresh file_key per call too. Two prepares of the same file →
    // different IVs, different envelopes.
    const alice = await setupNewVault('alice')
    const bob = await setupNewVault('bob')
    const bobPub = await publicKeyBase64(bob.accountKey)

    const file = new File([new TextEncoder().encode('same content')], 'a.bin')
    const a = await prepareAttachmentForUpload(file, [{ userID: 2, publicKey: bobPub }], alice.accountKey)
    const b = await prepareAttachmentForUpload(file, [{ userID: 2, publicKey: bobPub }], alice.accountKey)

    expect(a.fileIv).not.toBe(b.fileIv)
    // Each envelope wraps a *different* file_key (and uses a different IV),
    // so the encrypted_file_key strings differ too.
    expect(a.envelopes[0].encrypted_file_key).not.toBe(b.envelopes[0].encrypted_file_key)
    expect(a.envelopes[0].iv).not.toBe(b.envelopes[0].iv)
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
