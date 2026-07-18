package io.moth.push

import android.Manifest
import android.app.Activity
import android.app.NotificationManager
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import android.os.Handler
import android.os.Looper
import com.google.firebase.FirebaseApp
import com.google.firebase.messaging.FirebaseMessaging
import io.flutter.embedding.engine.plugins.FlutterPlugin
import io.flutter.embedding.engine.plugins.activity.ActivityAware
import io.flutter.embedding.engine.plugins.activity.ActivityPluginBinding
import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel
import io.flutter.plugin.common.PluginRegistry

/**
 * moth's first-party FCM bridge. Produces the push credential the moth
 * server stores (`target: fcm`, the FCM registration token) and reports the
 * POST_NOTIFICATIONS state faithfully — nothing else. No message handlers,
 * no notification display: delivery stays app code.
 *
 * FCM needs the app's own Firebase config (google-services.json) — the one
 * piece of setup moth cannot absorb. When Firebase is not initialized,
 * getToken fails with an actionable `firebase-not-initialized` error instead
 * of crashing; the Dart SDK treats it as a non-fatal registration failure.
 */
class MothPushPlugin :
  FlutterPlugin,
  ActivityAware,
  MethodChannel.MethodCallHandler,
  PluginRegistry.RequestPermissionsResultListener {

  private var channel: MethodChannel? = null
  private var context: Context? = null
  private var activity: Activity? = null
  private var activityBinding: ActivityPluginBinding? = null
  private val mainHandler = Handler(Looper.getMainLooper())

  /** The in-flight POST_NOTIFICATIONS request launched by [requestPermission]. */
  private var pendingPermission: MethodChannel.Result? = null

  companion object {
    private const val PERMISSION_REQUEST_CODE = 0x6d70 // "mp"

    /**
     * The engine-attached instance [MothPushMessagingService] forwards
     * onNewToken through. A rotation with no attached engine is dropped
     * safely: the SDK registers with a fresh token on next launch.
     */
    internal var instance: MothPushPlugin? = null
  }

  override fun onAttachedToEngine(binding: FlutterPlugin.FlutterPluginBinding) {
    context = binding.applicationContext
    channel = MethodChannel(binding.binaryMessenger, "moth_push")
    channel?.setMethodCallHandler(this)
    instance = this
  }

  override fun onDetachedFromEngine(binding: FlutterPlugin.FlutterPluginBinding) {
    if (instance === this) instance = null
    channel?.setMethodCallHandler(null)
    channel = null
    context = null
  }

  override fun onAttachedToActivity(binding: ActivityPluginBinding) {
    activity = binding.activity
    activityBinding = binding
    binding.addRequestPermissionsResultListener(this)
  }

  override fun onDetachedFromActivityForConfigChanges() = onDetachedFromActivity()

  override fun onReattachedToActivityForConfigChanges(binding: ActivityPluginBinding) =
    onAttachedToActivity(binding)

  override fun onDetachedFromActivity() {
    activityBinding?.removeRequestPermissionsResultListener(this)
    activityBinding = null
    activity = null
  }

  override fun onMethodCall(call: MethodCall, result: MethodChannel.Result) {
    when (call.method) {
      "requestPermission" -> requestPermission(result)
      "permissionStatus" -> result.success(permissionStatus())
      "getToken" -> getToken(result)
      "deviceMetadata" -> result.success(deviceMetadata())
      else -> result.notImplemented()
    }
  }

  /** Called by [MothPushMessagingService] when FCM rotates the registration token. */
  internal fun onNewToken(token: String) {
    mainHandler.post {
      channel?.invokeMethod(
        "onTokenRefresh",
        hashMapOf<String, Any?>("target" to "fcm", "token" to token),
      )
    }
  }

  /**
   * API 33+: the POST_NOTIFICATIONS runtime permission. Below 33 there is no
   * runtime permission — granted when notifications are enabled for the app.
   * Android cannot distinguish "never asked" from "denied" without
   * bookkeeping the OS may not honor, so an ungranted state reports denied —
   * honest enough for the registry, which stores denied devices anyway.
   */
  private fun permissionStatus(): String {
    val ctx = context ?: return "unknown"
    val enabled =
      if (Build.VERSION.SDK_INT >= 33) {
        ctx.checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) ==
          PackageManager.PERMISSION_GRANTED
      } else if (Build.VERSION.SDK_INT >= 24) {
        (ctx.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager)
          .areNotificationsEnabled()
      } else {
        true // no per-app notification switch to read before API 24
      }
    return if (enabled) "granted" else "denied"
  }

  private fun requestPermission(result: MethodChannel.Result) {
    if (Build.VERSION.SDK_INT < 33 || permissionStatus() == "granted") {
      // Nothing to prompt for: below 33 notifications need no runtime
      // permission; already-granted needs no second dialog.
      return result.success(permissionStatus())
    }
    val act =
      activity
        ?: return result.error(
          "unavailable",
          "No foreground activity to request the notification permission from.",
          null,
        )
    if (pendingPermission != null) {
      return result.error(
        "in-progress",
        "A notification-permission request is already in progress.",
        null,
      )
    }
    pendingPermission = result
    act.requestPermissions(
      arrayOf(Manifest.permission.POST_NOTIFICATIONS),
      PERMISSION_REQUEST_CODE,
    )
  }

  override fun onRequestPermissionsResult(
    requestCode: Int,
    permissions: Array<out String>,
    grantResults: IntArray,
  ): Boolean {
    if (requestCode != PERMISSION_REQUEST_CODE) return false
    val reply = pendingPermission ?: return true
    pendingPermission = null
    val granted =
      grantResults.isNotEmpty() && grantResults[0] == PackageManager.PERMISSION_GRANTED
    reply.success(if (granted) "granted" else "denied")
    return true
  }

  private fun getToken(result: MethodChannel.Result) {
    val ctx = context ?: return fail(result, "unavailable", "Plugin is not attached.")
    if (FirebaseApp.getApps(ctx).isEmpty()) {
      // Actionable, not a crash: FirebaseMessaging.getInstance() would throw
      // IllegalStateException("Default FirebaseApp is not initialized").
      return fail(
        result,
        "firebase-not-initialized",
        "Firebase is not initialized. moth_push uses Firebase Cloud Messaging " +
          "on Android, which needs your app's own Firebase project: download " +
          "google-services.json into android/app/ and apply the " +
          "com.google.gms.google-services Gradle plugin, then rebuild. " +
          "See the moth_push README.",
      )
    }
    FirebaseMessaging.getInstance().token.addOnCompleteListener { task ->
      val token = if (task.isSuccessful) task.result else null
      when {
        !task.isSuccessful ->
          fail(
            result,
            "fcm-error",
            task.exception?.message ?: "FCM registration-token retrieval failed.",
          )
        token.isNullOrEmpty() -> success(result, null) // no credential yet
        else ->
          success(result, hashMapOf<String, Any?>("target" to "fcm", "token" to token))
      }
    }
  }

  private fun deviceMetadata(): HashMap<String, Any?> {
    val model =
      if (Build.MODEL.startsWith(Build.MANUFACTURER, ignoreCase = true)) Build.MODEL
      else "${Build.MANUFACTURER} ${Build.MODEL}"
    val appVersion =
      try {
        val ctx = context
        val info = ctx?.packageManager?.getPackageInfo(ctx.packageName, 0)
        val code =
          if (Build.VERSION.SDK_INT >= 28) info?.longVersionCode
          else @Suppress("DEPRECATION") info?.versionCode?.toLong()
        if (info?.versionName == null) "" else "${info.versionName}+$code"
      } catch (_: Exception) {
        ""
      }
    return hashMapOf<String, Any?>("model" to model, "appVersion" to appVersion)
  }

  // Firebase task callbacks may arrive off the platform thread; channel
  // replies must not.
  private fun success(result: MethodChannel.Result, value: Any?) =
    mainHandler.post { result.success(value) }

  private fun fail(result: MethodChannel.Result, code: String, message: String) =
    mainHandler.post { result.error(code, message, null) }
}
