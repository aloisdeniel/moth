package io.moth.push

import com.google.firebase.messaging.FirebaseMessagingService

/**
 * Forwards FCM registration-token rotation to Dart. Registered by the
 * plugin's own AndroidManifest.xml (manifest-merged into the app), so
 * rotation reaches the SDK with no app-side code.
 *
 * Token rotation only, by design: onMessageReceived is left untouched —
 * moth's plugin produces credentials, delivery and display stay app code.
 * An app that ships its own FirebaseMessagingService (for message handling)
 * takes over the MESSAGING_EVENT intent; onNewToken then stops reaching
 * this service, which is safe — the SDK re-registers with a fresh token on
 * every launch.
 */
class MothPushMessagingService : FirebaseMessagingService() {
  override fun onNewToken(token: String) {
    MothPushPlugin.instance?.onNewToken(token)
  }
}
