package com.messenger.app

import android.content.ContentValues
import android.os.Build
import android.os.Environment
import android.provider.MediaStore
import android.util.Base64

import com.getcapacitor.JSObject
import com.getcapacitor.Plugin
import com.getcapacitor.PluginCall
import com.getcapacitor.PluginMethod
import com.getcapacitor.annotation.CapacitorPlugin

/**
 * Tiny native bridge that lets the JS layer write a base64 blob into the
 * system Downloads folder via MediaStore.Downloads (API 29+).
 *
 * Why not @capacitor/filesystem?  Filesystem can only write to its own
 * Directory.* roots; the public /Download folder isn't one of them. On
 * Android 11+ /Download/ is gated by MANAGE_EXTERNAL_STORAGE (a special
 * permission Google Play restricts). MediaStore.Downloads is the
 * Google-sanctioned escape hatch — no permissions at all, file appears
 * in the system Downloads notification + every file manager.
 *
 * On Android < 10 this plugin rejects with "unsupported_api"; the JS
 * caller is expected to fall back to Filesystem.writeFile(Documents/...).
 */
@CapacitorPlugin(name = "Downloads")
class DownloadsPlugin : Plugin() {

    @PluginMethod
    fun saveToDownloads(call: PluginCall) {
        val base64 = call.getString("data")
        val fileName = call.getString("fileName")
        val mimeType = call.getString("mimeType", "application/octet-stream")!!

        if (base64 == null || fileName == null) {
            call.reject("missing data or fileName")
            return
        }

        // MediaStore.Downloads is API 29 (Android 10). Earlier versions need
        // WRITE_EXTERNAL_STORAGE + Environment.getExternalStoragePublicDirectory,
        // which we deliberately don't add to the manifest to keep the perms
        // set minimal. JS caller handles the fallback to Documents/.
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.Q) {
            call.reject("unsupported_api")
            return
        }

        val bytes = try {
            Base64.decode(base64, Base64.NO_WRAP)
        } catch (e: IllegalArgumentException) {
            call.reject("bad base64: ${e.message}", e)
            return
        }

        val resolver = context.contentResolver
        val values = ContentValues().apply {
            put(MediaStore.Downloads.DISPLAY_NAME, fileName)
            put(MediaStore.Downloads.MIME_TYPE, mimeType)
            put(MediaStore.Downloads.RELATIVE_PATH, Environment.DIRECTORY_DOWNLOADS)
            // IS_PENDING gates other apps from reading the file until we
            // finish writing. Cleared in the second update() below.
            put(MediaStore.Downloads.IS_PENDING, 1)
        }

        val collection = MediaStore.Downloads.getContentUri(MediaStore.VOLUME_EXTERNAL_PRIMARY)
        val itemUri = resolver.insert(collection, values)
        if (itemUri == null) {
            call.reject("MediaStore.insert returned null")
            return
        }

        try {
            resolver.openOutputStream(itemUri)?.use { os ->
                os.write(bytes)
                os.flush()
            } ?: run {
                resolver.delete(itemUri, null, null)
                call.reject("openOutputStream returned null")
                return
            }
        } catch (e: Exception) {
            // Clean up the half-written pending row so the user doesn't end
            // up with a zero-byte ghost entry in their Downloads.
            runCatching { resolver.delete(itemUri, null, null) }
            call.reject("write failed: ${e.message}", e)
            return
        }

        resolver.update(itemUri, ContentValues().apply {
            put(MediaStore.Downloads.IS_PENDING, 0)
        }, null, null)

        call.resolve(JSObject().apply {
            put("uri", itemUri.toString())
            put("path", "Download/$fileName")
        })
    }
}
