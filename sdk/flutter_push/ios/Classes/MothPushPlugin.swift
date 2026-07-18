import Flutter
import UIKit
import UserNotifications

/// moth's first-party APNs bridge. Produces the push credential the moth
/// server stores (`target: apns`, hex-encoded device token) and reports the
/// OS authorization state faithfully — nothing else. No notification
/// handlers, no foreground banners: delivery and display stay app code.
///
/// The plugin registers itself as a Flutter application-lifecycle delegate,
/// so `didRegisterForRemoteNotificationsWithDeviceToken` reaches it through
/// Flutter's AppDelegate forwarding — no app-side AppDelegate code required.
public class MothPushPlugin: NSObject, FlutterPlugin {
  private var channel: FlutterMethodChannel?

  /// The latest APNs device token, hex-encoded, once registration succeeded.
  private var deviceToken: String?

  /// getToken calls awaiting the registration callback.
  private var pendingTokenResults: [FlutterResult] = []

  public static func register(with registrar: FlutterPluginRegistrar) {
    let channel = FlutterMethodChannel(
      name: "moth_push", binaryMessenger: registrar.messenger())
    let instance = MothPushPlugin()
    instance.channel = channel
    registrar.addMethodCallDelegate(instance, channel: channel)
    // Delivers the UIApplicationDelegate remote-notification callbacks below
    // to this plugin.
    registrar.addApplicationDelegate(instance)
  }

  public func detachFromEngine(for registrar: FlutterPluginRegistrar) {
    channel = nil
    pendingTokenResults.removeAll()
  }

  public func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    switch call.method {
    case "requestPermission":
      let provisional =
        (call.arguments as? [String: Any])?["provisional"] as? Bool ?? false
      requestPermission(provisional: provisional, result: result)
    case "permissionStatus":
      permissionStatus(result: result)
    case "getToken":
      getToken(result: result)
    case "deviceMetadata":
      result(["model": modelIdentifier(), "appVersion": appVersion()])
    default:
      result(FlutterMethodNotImplemented)
    }
  }

  private func requestPermission(provisional: Bool, result: @escaping FlutterResult) {
    var options: UNAuthorizationOptions = [.alert, .badge, .sound]
    if provisional {
      // Quiet notifications without showing the prompt at all.
      options.insert(.provisional)
    }
    UNUserNotificationCenter.current().requestAuthorization(options: options) { _, _ in
      // Report what the OS now says rather than the callback boolean, so
      // provisional comes back as provisional, not a bare "granted".
      self.permissionStatus(result: result)
    }
  }

  private func permissionStatus(result: @escaping FlutterResult) {
    UNUserNotificationCenter.current().getNotificationSettings { settings in
      let status: String
      switch settings.authorizationStatus {
      case .authorized, .ephemeral: status = "granted"
      case .provisional: status = "provisional"
      case .denied: status = "denied"
      case .notDetermined: status = "unknown"
      @unknown default: status = "unknown"
      }
      DispatchQueue.main.async { result(status) }
    }
  }

  /// Registers with APNs and resolves with the device token. Registration
  /// does not require notification permission (a denied device still gets a
  /// token — senders decide what to skip), so this is called unconditionally.
  private func getToken(result: @escaping FlutterResult) {
    if let token = deviceToken {
      result(["target": "apns", "token": token])
      return
    }
    pendingTokenResults.append(result)
    DispatchQueue.main.async {
      UIApplication.shared.registerForRemoteNotifications()
    }
  }

  public func application(
    _ application: UIApplication,
    didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data
  ) {
    let token = deviceToken.map { String(format: "%02x", $0) }.joined()
    self.deviceToken = token
    let payload: [String: Any] = ["target": "apns", "token": token]
    for pending in pendingTokenResults { pending(payload) }
    pendingTokenResults.removeAll()
    // Forward every registration callback, not just changes: APNs tokens
    // rotate on restore/OS update, and re-registration is idempotent
    // server-side (upsert by device).
    channel?.invokeMethod("onTokenRefresh", arguments: payload)
  }

  public func application(
    _ application: UIApplication,
    didFailToRegisterForRemoteNotificationsWithError error: Error
  ) {
    // Broken native setup (no push entitlement, no network to APNs, plain
    // simulator): a typed error the SDK treats as a non-fatal registration
    // failure — it retries on the next trigger.
    for pending in pendingTokenResults {
      pending(
        FlutterError(
          code: "apns-error", message: error.localizedDescription, details: nil))
    }
    pendingTokenResults.removeAll()
  }

  /// The machine identifier (e.g. `iPhone16,1`) — more useful in the admin
  /// Devices panel than the marketing-name-less `UIDevice.model`.
  private func modelIdentifier() -> String {
    var systemInfo = utsname()
    uname(&systemInfo)
    return Mirror(reflecting: systemInfo.machine).children.reduce(into: "") {
      id, element in
      guard let value = element.value as? Int8, value != 0 else { return }
      id.append(String(UnicodeScalar(UInt8(bitPattern: value))))
    }
  }

  private func appVersion() -> String {
    let info = Bundle.main.infoDictionary
    let version = info?["CFBundleShortVersionString"] as? String ?? ""
    let build = info?["CFBundleVersion"] as? String ?? ""
    if version.isEmpty { return build }
    return build.isEmpty ? version : "\(version)+\(build)"
  }
}
