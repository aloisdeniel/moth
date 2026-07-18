import Flutter
import StoreKit

/// moth's first-party StoreKit 2 billing bridge. Auto-renewing subscriptions
/// only. The verified transaction JWS is the receipt the moth server
/// validates; unverified transactions are rejected on-device and never
/// finished, and every transaction is finished only *after* its receipt has
/// been handed to Dart (an unfinished transaction replays on next launch; a
/// finished-but-unsent one is lost until restore).
public class MothBillingPlugin: NSObject, FlutterPlugin {
  private var channel: FlutterMethodChannel?
  private var updatesTask: Task<Void, Never>?

  public static func register(with registrar: FlutterPluginRegistrar) {
    let channel = FlutterMethodChannel(
      name: "moth_billing", binaryMessenger: registrar.messenger())
    let instance = MothBillingPlugin()
    instance.channel = channel
    registrar.addMethodCallDelegate(instance, channel: channel)
    instance.listenForUpdates()
  }

  public func detachFromEngine(for registrar: FlutterPluginRegistrar) {
    updatesTask?.cancel()
    updatesTask = nil
    channel = nil
  }

  public func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    switch call.method {
    case "getProducts":
      guard let args = call.arguments as? [String: Any],
        let ids = args["productIds"] as? [String]
      else {
        result(FlutterError(code: "store-error", message: "productIds is required", details: nil))
        return
      }
      Task { await self.getProducts(ids: ids, result: result) }
    case "purchase":
      guard let args = call.arguments as? [String: Any],
        let id = args["productId"] as? String
      else {
        result(FlutterError(code: "store-error", message: "productId is required", details: nil))
        return
      }
      Task { await self.purchase(id: id, result: result) }
    case "restore":
      Task { await self.restore(result: result) }
    default:
      result(FlutterMethodNotImplemented)
    }
  }

  @MainActor
  private func getProducts(ids: [String], result: @escaping FlutterResult) async {
    do {
      let products = try await Product.products(for: ids)
      result(
        products.map { product -> [String: Any] in
          var entry: [String: Any] = [
            "productId": product.id,
            "price": product.displayPrice,
            "currency": product.priceFormatStyle.currencyCode,
            "title": product.displayName,
            "description": product.description,
          ]
          if let intro = product.subscription?.introductoryOffer {
            entry["introPrice"] = intro.displayPrice
            entry["introPeriod"] = isoPeriod(intro.period)
            entry["introIsFreeTrial"] = intro.paymentMode == .freeTrial
          }
          return entry
        })
    } catch {
      result(
        FlutterError(code: "store-error", message: error.localizedDescription, details: nil))
    }
  }

  /// The ISO 8601 duration for a StoreKit subscription period (P3D, P1W,
  /// P1M, P1Y) — the same shape Play Billing's `billingPeriod` uses, so Dart
  /// sees one period format from both stores.
  private func isoPeriod(_ period: Product.SubscriptionPeriod) -> String {
    switch period.unit {
    case .day: return "P\(period.value)D"
    case .week: return "P\(period.value)W"
    case .month: return "P\(period.value)M"
    case .year: return "P\(period.value)Y"
    @unknown default: return "P\(period.value)D"
    }
  }

  @MainActor
  private func purchase(id: String, result: @escaping FlutterResult) async {
    do {
      guard let product = try await Product.products(for: [id]).first else {
        result(
          FlutterError(
            code: "not-found",
            message: "Product \"\(id)\" was not found in the store.", details: nil))
        return
      }
      switch try await product.purchase() {
      case .success(let verification):
        switch verification {
        case .verified(let transaction):
          result(["productId": transaction.productID, "jws": verification.jwsRepresentation])
          await transaction.finish()
        case .unverified:
          result(
            FlutterError(
              code: "store-error",
              message: "The store returned an unverified transaction.", details: nil))
        }
      case .userCancelled:
        result(nil)  // cancellation is a nil receipt, not an error
      case .pending:
        // Ask to Buy / deferred payment: completion arrives via
        // Transaction.updates and is forwarded as onTransactionUpdated.
        result(
          FlutterError(
            code: "pending", message: "The purchase is awaiting approval.", details: nil))
      @unknown default:
        result(
          FlutterError(code: "store-error", message: "Unknown purchase result.", details: nil))
      }
    } catch {
      result(
        FlutterError(code: "store-error", message: error.localizedDescription, details: nil))
    }
  }

  @MainActor
  private func restore(result: @escaping FlutterResult) async {
    var receipts: [String] = []
    for await entitlement in Transaction.currentEntitlements {
      if case .verified = entitlement {
        receipts.append(entitlement.jwsRepresentation)
      }
    }
    result(receipts)
  }

  /// Surfaces out-of-band transactions (Ask to Buy approvals, renewals,
  /// purchases made outside the app) to Dart, finishing each only once Dart
  /// has taken the receipt.
  private func listenForUpdates() {
    updatesTask = Task { [weak self] in
      for await update in Transaction.updates {
        guard case .verified(let transaction) = update else { continue }
        await self?.forward(update: update, transaction: transaction)
      }
    }
  }

  @MainActor
  private func forward(update: VerificationResult<Transaction>, transaction: Transaction) async {
    guard let channel = channel else { return }
    channel.invokeMethod(
      "onTransactionUpdated",
      arguments: [
        "productId": transaction.productID,
        "jws": update.jwsRepresentation,
      ]
    ) { reply in
      // Finish only when Dart actually took the receipt. A
      // FlutterMethodNotImplemented reply (no Dart handler yet — e.g. a
      // launch replay before MothStoreBilling is constructed, or a
      // channel-buffer overflow) or a FlutterError (handler failed, adapter
      // disposed, receipt refused) leaves the transaction unfinished so
      // StoreKit replays it on a later launch instead of losing it until a
      // manual restore.
      if reply is FlutterError { return }
      if (reply as? NSObject) === FlutterMethodNotImplemented { return }
      Task { await transaction.finish() }
    }
  }
}
