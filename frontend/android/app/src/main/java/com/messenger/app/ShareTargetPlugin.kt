package com.messenger.app

import android.content.Intent

import com.getcapacitor.JSObject
import com.getcapacitor.Plugin
import com.getcapacitor.PluginCall
import com.getcapacitor.PluginMethod
import com.getcapacitor.annotation.CapacitorPlugin

/**
 * Receives ACTION_SEND (text/plain) intents from the system share-sheet and
 * surfaces them to the JS layer — "Share to Messenger" from YouTube, a
 * browser, etc.
 *
 * Two delivery paths, because launchMode=singleTask:
 *   - cold start: the share Intent IS the launch Intent. JS calls
 *     getSharedItem() once the WebView is up and drains it (one-shot).
 *   - warm start (app already running): Android routes the new Intent to
 *     MainActivity.onNewIntent(), which forwards it here and we emit
 *     "shareReceived" so the already-mounted UI reacts immediately.
 *
 * One-shot semantics mirror the JS pendingShare pub/sub: a given shared text
 * is handed out exactly once, so re-opening the app doesn't re-trigger it.
 *
 * Why a custom plugin?  Capacitor has no share-RECEIVE plugin (@capacitor/share
 * only sends). Reading the launch Intent + handling onNewIntent for singleTask
 * needs native code regardless.
 */
@CapacitorPlugin(name = "ShareTarget")
class ShareTargetPlugin : Plugin() {

    // Buffered shared text waiting for the JS layer to drain it. Survives the
    // cold-start gap between activity launch and the WebView calling
    // getSharedItem().
    private var pendingText: String? = null

    override fun load() {
        // Capture the Intent that cold-started the activity (if it's a share).
        consumeIntent(activity?.intent)
    }

    /**
     * Called by MainActivity.onNewIntent for warm starts. Buffers the text and,
     * since the bridge is necessarily live (app already running), drains it
     * immediately via a "shareReceived" event.
     */
    fun handleIntent(intent: Intent?) {
        consumeIntent(intent)
        val text = pendingText ?: return
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

    /**
     * Drains the buffered cold-start share text (one-shot). Resolves with
     * {text: null} when nothing is pending — the JS bridge treats that as
     * "no share to handle".
     */
    @PluginMethod
    fun getSharedItem(call: PluginCall) {
        val text = pendingText
        pendingText = null
        call.resolve(JSObject().apply { put("text", text) })
    }
}
