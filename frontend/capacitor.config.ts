import type { CapacitorConfig } from '@capacitor/cli'

const config: CapacitorConfig = {
  appId: 'com.messenger.app',
  appName: 'Messenger',
  webDir: 'dist',
  server: {
    androidScheme: 'https',
    cleartext: false,
  },
  android: {
    allowMixedContent: false,
    // Enable Chrome remote debugging (`chrome://inspect/#devices`) even on
    // release builds. Android refuses to attach an inspector unless the app
    // is marked debuggable OR this flag is set via JNI. We have no Play Store /
    // closed-track distribution — every user is a tester, and the inspector is
    // invaluable when self-update fails silently in release.
    webContentsDebuggingEnabled: true,
  },
}

export default config
