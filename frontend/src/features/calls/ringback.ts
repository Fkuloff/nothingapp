/**
 * Programmatic call tones via Web Audio API. No external audio files needed.
 *
 * - createRingback(): played for the CALLER while waiting (425 Hz, 1s on / 4s off)
 * - createRingtone(): played for the RECEIVER on incoming call (440+480 Hz dual-tone, double ring)
 */

// ── Ringback (caller hears while waiting) ──────────────────────────

const RINGBACK_FREQ = 425
const RINGBACK_ON = 1000
const RINGBACK_OFF = 4000
const RINGBACK_VOL = 0.15

export function createRingback() {
  let ctx: AudioContext | null = null
  let osc: OscillatorNode | null = null
  let gain: GainNode | null = null
  let timer: ReturnType<typeof setInterval> | null = null
  let onTimeout: ReturnType<typeof setTimeout> | null = null

  function start() {
    stop()
    ctx = new AudioContext()
    osc = ctx.createOscillator()
    gain = ctx.createGain()
    osc.type = 'sine'
    osc.frequency.value = RINGBACK_FREQ
    osc.connect(gain)
    gain.connect(ctx.destination)
    gain.gain.value = 0
    osc.start()
    beepOn()
    timer = setInterval(beepOn, RINGBACK_ON + RINGBACK_OFF)
  }

  function beepOn() {
    if (!gain) return
    gain.gain.value = RINGBACK_VOL
    onTimeout = setTimeout(() => { if (gain) gain.gain.value = 0 }, RINGBACK_ON)
  }

  function stop() {
    if (onTimeout) clearTimeout(onTimeout)
    if (timer) clearInterval(timer)
    if (osc) { osc.stop(); osc.disconnect() }
    if (gain) gain.disconnect()
    if (ctx) ctx.close()
    ctx = null; osc = null; gain = null; timer = null; onTimeout = null
  }

  return { start, stop }
}

// ── Ringtone (receiver hears on incoming call) ─────────────────────

const RING_FREQ_1 = 440
const RING_FREQ_2 = 480
const RING_ON = 400    // each ring burst
const RING_GAP = 200   // gap between two bursts
const RING_PAUSE = 3000 // pause between ring cycles
const RING_VOL = 0.25

export function createRingtone() {
  let ctx: AudioContext | null = null
  let osc1: OscillatorNode | null = null
  let osc2: OscillatorNode | null = null
  let gain: GainNode | null = null
  let timer: ReturnType<typeof setInterval> | null = null
  let t1: ReturnType<typeof setTimeout> | null = null
  let t2: ReturnType<typeof setTimeout> | null = null
  let t3: ReturnType<typeof setTimeout> | null = null

  function start() {
    stop()
    ctx = new AudioContext()
    gain = ctx.createGain()
    gain.connect(ctx.destination)
    gain.gain.value = 0

    osc1 = ctx.createOscillator()
    osc1.type = 'sine'
    osc1.frequency.value = RING_FREQ_1
    osc1.connect(gain)
    osc1.start()

    osc2 = ctx.createOscillator()
    osc2.type = 'sine'
    osc2.frequency.value = RING_FREQ_2
    osc2.connect(gain)
    osc2.start()

    ringCycle()
    const cycleMs = RING_ON + RING_GAP + RING_ON + RING_PAUSE
    timer = setInterval(ringCycle, cycleMs)
  }

  // Double ring: burst — gap — burst — pause
  function ringCycle() {
    if (!gain) return
    gain.gain.value = RING_VOL
    t1 = setTimeout(() => { if (gain) gain.gain.value = 0 }, RING_ON)
    t2 = setTimeout(() => { if (gain) gain.gain.value = RING_VOL }, RING_ON + RING_GAP)
    t3 = setTimeout(() => { if (gain) gain.gain.value = 0 }, RING_ON + RING_GAP + RING_ON)
  }

  function stop() {
    if (t1) clearTimeout(t1)
    if (t2) clearTimeout(t2)
    if (t3) clearTimeout(t3)
    if (timer) clearInterval(timer)
    if (osc1) { osc1.stop(); osc1.disconnect() }
    if (osc2) { osc2.stop(); osc2.disconnect() }
    if (gain) gain.disconnect()
    if (ctx) ctx.close()
    ctx = null; osc1 = null; osc2 = null; gain = null
    timer = null; t1 = null; t2 = null; t3 = null
  }

  return { start, stop }
}
