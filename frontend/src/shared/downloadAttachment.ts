// Native-platform download for E2E attachments.
//
// On the web, `<a href={blobUrl} download={fileName}>` is enough — the
// browser routes the click through its download manager. The Capacitor
// Android WebView does NOT honour `download`: nothing happens, the file
// stays inside the WebView's memory. So on native we have to:
//   1. Write the decrypted Blob to the app's cache directory via
//      @capacitor/filesystem (must round-trip through base64 — Android
//      bridge marshals strings, not Blobs).
//   2. Invoke @capacitor/share with the resulting file:// URI, which
//      pops the native share sheet ("Save to Files", "Open with X",
//      "Send via Gmail", ...).
//
// The user picks where the file ends up. No new permissions needed —
// Cache directory is app-private and FileProvider-shareable via the
// Capacitor plugins' built-in manifest.

import { isNative } from './platform'

/** Strip path-traversal characters from a user-supplied filename so we can
 *  pass it as `path` to writeFile without escaping out of the cache dir. */
function sanitizeFileName(name: string): string {
  // Drop any directory components and dotdot — keep just the basename.
  // Falls back to "file" if everything got stripped.
  const cleaned = name.replace(/[/\\]/g, '_').replace(/\.\./g, '_').trim()
  return cleaned || 'file'
}

/** Encode a Blob as base64 in 64KB chunks. Avoids building a giant binary
 *  string in one go (btoa on a multi-MB string causes UI jank). */
async function blobToBase64(blob: Blob): Promise<string> {
  const buffer = await blob.arrayBuffer()
  const bytes = new Uint8Array(buffer)
  const chunkSize = 0x10000 // 64 KB
  let binary = ''
  for (let i = 0; i < bytes.length; i += chunkSize) {
    const slice = bytes.subarray(i, Math.min(i + chunkSize, bytes.length))
    binary += String.fromCharCode.apply(null, Array.from(slice) as number[])
  }
  return btoa(binary)
}

/**
 * Hands a decrypted attachment Blob to the native share sheet so the user
 * can save it to Files / Downloads / open with another app. No-op on web
 * (caller should let the default `<a download>` behaviour do its job).
 *
 * Errors bubble up so the caller can show a toast — typical failure modes
 * are user-cancelled share (benign) or out-of-cache-space (rare).
 */
export async function downloadAttachmentNative(blob: Blob, fileName: string): Promise<void> {
  if (!isNative()) return

  const safeName = sanitizeFileName(fileName)

  const [{ Filesystem, Directory }, { Share }] = await Promise.all([
    import('@capacitor/filesystem'),
    import('@capacitor/share'),
  ])

  const base64 = await blobToBase64(blob)
  const written = await Filesystem.writeFile({
    path: safeName,
    data: base64,
    directory: Directory.Cache,
    recursive: true,
  })

  await Share.share({
    title: safeName,
    url: written.uri,
    dialogTitle: 'Сохранить файл',
  })
}
