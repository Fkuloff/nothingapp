// Tests for the app-lock timing decision. The storage plumbing is the same
// localStorage/Preferences mirror pattern as the auth token; what must never
// regress is the pure expiry logic — a wrong answer either locks users out
// on every resume or silently stops locking at all.

import { describe, expect, it } from 'vitest'

import { isLockExpired } from './appLock'

describe('isLockExpired', () => {
  const NOW = 1_750_000_000_000

  it('locks when there is no record of last activity (fail closed)', () => {
    expect(isLockExpired(null, 60_000, NOW)).toBe(true)
  })

  it('locks when the stored value is corrupt', () => {
    expect(isLockExpired('not-a-number', 60_000, NOW)).toBe(true)
  })

  it('does not lock before the timeout elapses', () => {
    expect(isLockExpired(String(NOW - 59_999), 60_000, NOW)).toBe(false)
  })

  it('locks exactly at the timeout boundary', () => {
    expect(isLockExpired(String(NOW - 60_000), 60_000, NOW)).toBe(true)
  })

  it('locks after the timeout elapses', () => {
    expect(isLockExpired(String(NOW - 3_600_000), 60_000, NOW)).toBe(true)
  })

  it('timeout 0 ("Сразу") locks on any return to foreground', () => {
    expect(isLockExpired(String(NOW), 0, NOW)).toBe(true)
  })

  it('tolerates a clock that jumped backwards by staying unlocked within the window', () => {
    // now < lastActive (user changed the system clock) — difference is
    // negative, which is below any positive timeout: stay unlocked rather
    // than trap the user in a lock loop.
    expect(isLockExpired(String(NOW + 10_000), 60_000, NOW)).toBe(false)
  })
})
