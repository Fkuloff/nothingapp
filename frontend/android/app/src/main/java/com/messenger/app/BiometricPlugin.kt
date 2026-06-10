package com.messenger.app

import androidx.biometric.BiometricManager
import androidx.biometric.BiometricManager.Authenticators.BIOMETRIC_WEAK
import androidx.biometric.BiometricPrompt
import androidx.core.content.ContextCompat
import androidx.fragment.app.FragmentActivity

import com.getcapacitor.JSObject
import com.getcapacitor.Plugin
import com.getcapacitor.PluginCall
import com.getcapacitor.PluginMethod
import com.getcapacitor.annotation.CapacitorPlugin

/**
 * Biometric gate for the app-lock feature.
 *
 * Why a custom plugin? Same story as UpdaterPlugin: the community biometric
 * plugins lag behind Capacitor majors, and we need exactly two calls — "is
 * биометрия available" and "show the prompt". The app-lock semantics (what
 * locks, when, password fallback) all live JS-side in shared/appLock.ts;
 * this plugin is a dumb yes/no oracle.
 *
 * BIOMETRIC_WEAK (class 2: most face unlocks) is deliberately allowed: this
 * gates UI access on an already-trusted device, it does not release crypto
 * material — the account_key is in app storage regardless (see the threat
 * model note in shared/crypto/e2e.ts). Device-credential fallback is NOT
 * offered here because the lock screen has its own fallback — the account
 * password, which doubles as the E2E vault check.
 */
@CapacitorPlugin(name = "Biometric")
class BiometricPlugin : Plugin() {

    @PluginMethod
    fun isAvailable(call: PluginCall) {
        val status = BiometricManager.from(context).canAuthenticate(BIOMETRIC_WEAK)
        call.resolve(JSObject().apply {
            put("available", status == BiometricManager.BIOMETRIC_SUCCESS)
        })
    }

    @PluginMethod
    fun authenticate(call: PluginCall) {
        val act = activity as? FragmentActivity
        if (act == null) {
            call.reject("activity_unavailable")
            return
        }
        val title = call.getString("title") ?: "Подтвердите личность"
        val subtitle = call.getString("subtitle")
        val cancelTitle = call.getString("cancelTitle") ?: "Отмена"

        act.runOnUiThread {
            val prompt = BiometricPrompt(
                act,
                ContextCompat.getMainExecutor(context),
                object : BiometricPrompt.AuthenticationCallback() {
                    override fun onAuthenticationSucceeded(result: BiometricPrompt.AuthenticationResult) {
                        call.resolve()
                    }

                    override fun onAuthenticationError(errorCode: Int, errString: CharSequence) {
                        // Covers user cancel, lockout after too many attempts,
                        // and hardware errors. The JS side treats any reject as
                        // "not verified" and falls back to the password form.
                        call.reject(errString.toString(), errorCode.toString())
                    }

                    // onAuthenticationFailed (single bad attempt) is not
                    // overridden: the system prompt stays open and lets the
                    // user retry — rejecting here would close our promise
                    // while the prompt is still on screen.
                },
            )
            val info = BiometricPrompt.PromptInfo.Builder()
                .setTitle(title)
                .apply { if (!subtitle.isNullOrEmpty()) setSubtitle(subtitle) }
                .setNegativeButtonText(cancelTitle)
                .setAllowedAuthenticators(BIOMETRIC_WEAK)
                .setConfirmationRequired(false)
                .build()
            prompt.authenticate(info)
        }
    }
}
