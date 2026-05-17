package com.messenger.app

import android.content.Intent
import android.net.Uri
import android.provider.Settings
import androidx.core.content.FileProvider

import com.getcapacitor.JSObject
import com.getcapacitor.Plugin
import com.getcapacitor.PluginCall
import com.getcapacitor.PluginMethod
import com.getcapacitor.annotation.CapacitorPlugin

import java.io.File
import java.io.FileOutputStream
import java.net.HttpURLConnection
import java.net.URL
import java.security.MessageDigest

/**
 * In-app self-update plugin: report the currently-installed version, download
 * a new APK, and hand it to the system PackageInstaller.
 *
 * Why a custom plugin?  The Capacitor ecosystem has no maintained APK-install
 * plugin. Filesystem can write the bytes but can't trigger ACTION_VIEW with
 * the right MIME + FileProvider URI; Browser can't pre-verify SHA-256.
 *
 * Threat model in scope:
 *   - MITM during APK download → SHA-256 mismatch → reject, don't install.
 *   - Disk corruption / partial download → same digest check catches it.
 *   - Replay of an old APK at the same URL → versionCode comparison happens
 *     JS-side BEFORE calling downloadApk, so we won't even try to install
 *     a downgrade. (And Android refuses downgrades anyway.)
 *
 * Out of scope (relies on Android):
 *   - APK supply-chain compromise: Android refuses any install whose signing
 *     cert doesn't match the currently-installed app. We get that for free.
 */
@CapacitorPlugin(name = "Updater")
class UpdaterPlugin : Plugin() {

    @PluginMethod
    fun getCurrentVersion(call: PluginCall) {
        val pm = context.packageManager
        val info = pm.getPackageInfo(context.packageName, 0)
        @Suppress("DEPRECATION") // PackageInfo.versionCode is fine on minSdk 24
        val versionCode = info.versionCode
        call.resolve(JSObject().apply {
            put("version_code", versionCode)
            put("version_name", info.versionName ?: "")
            put("package_name", context.packageName)
        })
    }

    /**
     * Downloads an APK from `url`, verifies its SHA-256, and stores it in
     * the app's cache dir. Returns the absolute file path so installApk can
     * pick it up.
     *
     * Streams to disk in 64 KB chunks while updating the digest in flight —
     * never holds the whole APK in memory (multi-MB allocations on the
     * Android heap cause OOM on low-end devices).
     *
     * Progress events ("download_progress") are emitted with bytes_loaded
     * + bytes_total so the UI can render a progress bar.
     */
    @PluginMethod
    fun downloadApk(call: PluginCall) {
        val url = call.getString("url")
        val expectedSha256 = call.getString("sha256")?.lowercase()
        val fileName = call.getString("fileName") ?: "update.apk"

        if (url == null || expectedSha256 == null) {
            call.reject("missing url or sha256")
            return
        }
        if (!expectedSha256.matches(Regex("^[a-f0-9]{64}$"))) {
            call.reject("sha256 must be 64 lowercase hex chars")
            return
        }

        // Run the actual download off the JS thread — Capacitor handlers run
        // on the bridge thread which blocks the WebView when busy.
        Thread {
            doDownload(call, url, expectedSha256, fileName)
        }.start()
    }

    private fun doDownload(call: PluginCall, url: String, expectedSha256: String, fileName: String) {
        val updatesDir = File(context.cacheDir, "updates").apply { mkdirs() }
        // Clean up any stale partial downloads from previous attempts so we
        // don't accumulate gigabytes of abandoned APKs in cache.
        updatesDir.listFiles()?.forEach { it.delete() }

        val outFile = File(updatesDir, fileName)
        val digest = MessageDigest.getInstance("SHA-256")

        try {
            val conn = (URL(url).openConnection() as HttpURLConnection).apply {
                connectTimeout = 30_000
                readTimeout = 60_000
                instanceFollowRedirects = true
            }
            try {
                val total = conn.contentLengthLong
                conn.inputStream.use { input ->
                    FileOutputStream(outFile).use { output ->
                        val buf = ByteArray(64 * 1024)
                        var loaded = 0L
                        var lastProgressEmit = 0L
                        while (true) {
                            val n = input.read(buf)
                            if (n < 0) break
                            output.write(buf, 0, n)
                            digest.update(buf, 0, n)
                            loaded += n

                            // Throttle progress events — emitting on every
                            // 64KB chunk floods the bridge for big APKs.
                            // 250ms feels smooth without spamming.
                            val now = System.currentTimeMillis()
                            if (now - lastProgressEmit >= 250) {
                                notifyListeners("download_progress", JSObject().apply {
                                    put("bytes_loaded", loaded)
                                    put("bytes_total", total)
                                })
                                lastProgressEmit = now
                            }
                        }
                    }
                }
            } finally {
                conn.disconnect()
            }
        } catch (e: Exception) {
            outFile.delete()
            call.reject("download failed: ${e.message}", e)
            return
        }

        val actualSha256 = digest.digest().joinToString("") { "%02x".format(it) }
        if (actualSha256 != expectedSha256) {
            outFile.delete()
            call.reject("sha256 mismatch: expected $expectedSha256, got $actualSha256")
            return
        }

        call.resolve(JSObject().apply {
            put("path", outFile.absolutePath)
            put("sha256", actualSha256)
            put("size_bytes", outFile.length())
        })
    }

    /**
     * Hands a previously-downloaded APK to the system PackageInstaller via
     * Intent.ACTION_VIEW. Android shows the standard install dialog; if the
     * user denies, the activity returns without installing — we don't get a
     * callback either way (intentional Android design), but a follow-up
     * getCurrentVersion call from JS reveals whether the install succeeded.
     */
    @PluginMethod
    fun installApk(call: PluginCall) {
        val path = call.getString("path")
        if (path == null) {
            call.reject("missing path")
            return
        }
        val file = File(path)
        if (!file.exists() || !file.canRead()) {
            call.reject("file missing: $path")
            return
        }

        // On Android 8+ the user must have toggled "Install unknown apps"
        // for our package. We can't query the toggle directly (no API),
        // but if they haven't, the resulting install screen will route
        // them through the settings flow. Either way the Intent below is
        // the right next step.
        val authority = "${context.packageName}.fileprovider"
        val uri: Uri = try {
            FileProvider.getUriForFile(context, authority, file)
        } catch (e: IllegalArgumentException) {
            call.reject("FileProvider authority not configured for $path: ${e.message}", e)
            return
        }

        val intent = Intent(Intent.ACTION_VIEW).apply {
            setDataAndType(uri, "application/vnd.android.package-archive")
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
        }

        try {
            context.startActivity(intent)
        } catch (e: Exception) {
            call.reject("startActivity failed: ${e.message}", e)
            return
        }

        call.resolve()
    }

    /**
     * Opens the system "Install unknown apps" settings page for our package
     * so the user can toggle the permission. Called when installApk surfaces
     * the "не разрешено" Android dialog and the user clicks "open settings".
     */
    @PluginMethod
    fun openInstallSettings(call: PluginCall) {
        val intent = Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES).apply {
            data = Uri.parse("package:${context.packageName}")
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        try {
            context.startActivity(intent)
            call.resolve()
        } catch (e: Exception) {
            call.reject("open settings failed: ${e.message}", e)
        }
    }
}
