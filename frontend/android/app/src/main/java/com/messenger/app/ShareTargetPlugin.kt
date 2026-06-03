package com.messenger.app

import android.content.Intent

import com.getcapacitor.JSObject
import com.getcapacitor.Plugin
import com.getcapacitor.PluginCall
import com.getcapacitor.PluginMethod
import com.getcapacitor.annotation.CapacitorPlugin

/**
 * Receives ACTION_SEND (text/plain) intents from the system share-sheet —
 * "Share to Messenger". Capacitor has no share-RECEIVE plugin, so this is custom.
 *
 * Two paths (launchMode=singleTask):
 *   - cold start: the share Intent is the launch Intent; JS drains it via
 *     getSharedItem() once the WebView is up.
 *   - warm start: MainActivity.onNewIntent forwards the Intent here and we emit
 *     "shareReceived". Either way the text is handed out exactly once.
 */
@CapacitorPlugin(name = "ShareTarget")
class ShareTargetPlugin : Plugin() {

    // Buffered until the JS layer drains it; bridges the cold-start gap.
    private var pendingText: String? = null

    override fun load() {
        consumeIntent(activity?.intent)
    }

    /** Warm-start entry point (from MainActivity.onNewIntent). */
    fun handleIntent(intent: Intent?) {
        consumeIntent(intent)
        val text = pendingText ?: return
        // Only drain via the event if the JS side is actually listening. If the
        // share arrived before useShareTarget mounted (e.g. app warm-started onto
        // the login screen), keep it buffered so getSharedItem() can deliver it
        // once the UI comes up — otherwise the share would be lost.
        if (!hasListeners("shareReceived")) return
        pendingText = null
        notifyListeners("shareReceived", JSObject().apply { put("text", text) })
    }

    private fun consumeIntent(intent: Intent?) {
        if (intent == null) return
        if (intent.action != Intent.ACTION_SEND) return
        if (intent.type != "text/plain") return
        val text = intent.getStringExtra(Intent.EXTRA_TEXT) ?: return
        pendingText = text
    }

    /** Cold-start drain (one-shot); resolves {text: null} when nothing pending. */
    @PluginMethod
    fun getSharedItem(call: PluginCall) {
        val text = pendingText
        pendingText = null
        call.resolve(JSObject().apply { put("text", text) })
    }
}
