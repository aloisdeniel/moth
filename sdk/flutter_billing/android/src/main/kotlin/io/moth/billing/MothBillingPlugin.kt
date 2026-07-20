package io.moth.billing

import android.app.Activity
import android.os.Handler
import android.os.Looper
import com.android.billingclient.api.BillingClient
import com.android.billingclient.api.BillingClientStateListener
import com.android.billingclient.api.BillingFlowParams
import com.android.billingclient.api.BillingResult
import com.android.billingclient.api.PendingPurchasesParams
import com.android.billingclient.api.Purchase
import com.android.billingclient.api.PurchasesUpdatedListener
import com.android.billingclient.api.QueryProductDetailsParams
import com.android.billingclient.api.QueryPurchasesParams
import io.flutter.embedding.engine.plugins.FlutterPlugin
import io.flutter.embedding.engine.plugins.activity.ActivityAware
import io.flutter.embedding.engine.plugins.activity.ActivityPluginBinding
import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel

/**
 * moth's first-party Play Billing bridge. Auto-renewing subscriptions only.
 *
 * Purchases are NEVER acknowledged here: the moth server acknowledges after
 * validating the purchase token (SubmitPurchase), so an unvalidated purchase
 * auto-refunds after three days instead of being silently kept. Do not add
 * acknowledgePurchase calls.
 */
class MothBillingPlugin :
  FlutterPlugin, ActivityAware, MethodChannel.MethodCallHandler, PurchasesUpdatedListener {

  private var channel: MethodChannel? = null
  private var billingClient: BillingClient? = null
  private var activity: Activity? = null
  private val mainHandler = Handler(Looper.getMainLooper())

  /** The in-flight purchase launched by [purchase], resolved in [onPurchasesUpdated]. */
  private var pendingPurchase: MethodChannel.Result? = null
  private var pendingProductId: String? = null

  override fun onAttachedToEngine(binding: FlutterPlugin.FlutterPluginBinding) {
    channel = MethodChannel(binding.binaryMessenger, "moth_billing")
    channel?.setMethodCallHandler(this)
    billingClient =
      BillingClient.newBuilder(binding.applicationContext)
        .setListener(this)
        .enablePendingPurchases(
          PendingPurchasesParams.newBuilder().enableOneTimeProducts().build()
        )
        .build()
  }

  override fun onDetachedFromEngine(binding: FlutterPlugin.FlutterPluginBinding) {
    channel?.setMethodCallHandler(null)
    channel = null
    billingClient?.endConnection()
    billingClient = null
  }

  override fun onAttachedToActivity(binding: ActivityPluginBinding) {
    activity = binding.activity
  }

  override fun onDetachedFromActivityForConfigChanges() {
    activity = null
  }

  override fun onReattachedToActivityForConfigChanges(binding: ActivityPluginBinding) {
    activity = binding.activity
  }

  override fun onDetachedFromActivity() {
    activity = null
  }

  override fun onMethodCall(call: MethodCall, result: MethodChannel.Result) {
    when (call.method) {
      "getProducts" ->
        getProducts(call.argument<List<String>>("productIds") ?: emptyList(), result)
      "purchase" -> purchase(call.argument<String>("productId"), result)
      "restore" -> restore(result)
      else -> result.notImplemented()
    }
  }

  /**
   * Runs [onReady] with a connected client, retrying the connection twice
   * with backoff. A dropped connection is retried lazily: the next call
   * re-enters here and reconnects.
   */
  private fun connect(
    attempt: Int = 0,
    onError: (String) -> Unit,
    onReady: (BillingClient) -> Unit,
  ) {
    val client = billingClient ?: return onError("Billing is not available.")
    if (client.isReady) return onReady(client)
    client.startConnection(
      object : BillingClientStateListener {
        override fun onBillingSetupFinished(result: BillingResult) {
          when {
            result.responseCode == BillingClient.BillingResponseCode.OK -> onReady(client)
            attempt < 2 ->
              mainHandler.postDelayed(
                { connect(attempt + 1, onError, onReady) },
                (attempt + 1) * 500L,
              )
            else ->
              onError(
                result.debugMessage.ifEmpty {
                  "Billing setup failed (${result.responseCode})."
                }
              )
          }
        }

        override fun onBillingServiceDisconnected() {
          // Nothing to do: the next billing call reconnects.
        }
      }
    )
  }

  private fun subsParams(ids: List<String>): QueryProductDetailsParams =
    QueryProductDetailsParams.newBuilder()
      .setProductList(
        ids.map {
          QueryProductDetailsParams.Product.newBuilder()
            .setProductId(it)
            .setProductType(BillingClient.ProductType.SUBS)
            .build()
        }
      )
      .build()

  private fun getProducts(ids: List<String>, result: MethodChannel.Result) {
    if (ids.isEmpty()) return success(result, emptyList<Any>())
    connect(onError = { fail(result, "unavailable", it) }) { client ->
      client.queryProductDetailsAsync(subsParams(ids)) { billingResult, details ->
        if (billingResult.responseCode != BillingClient.BillingResponseCode.OK) {
          fail(result, "store-error", billingResult.debugMessage)
          return@queryProductDetailsAsync
        }
        success(
          result,
          details.map { d ->
            // First offer, base (last) pricing phase for the display price;
            // a zero-priced phase is the free trial.
            val phases =
              d.subscriptionOfferDetails?.firstOrNull()?.pricingPhases?.pricingPhaseList
                ?: emptyList()
            val base = phases.lastOrNull()
            val trial = phases.firstOrNull { it.priceAmountMicros == 0L }
            hashMapOf<String, Any?>(
              "productId" to d.productId,
              "price" to (base?.formattedPrice ?: ""),
              "currency" to (base?.priceCurrencyCode ?: ""),
              "title" to d.title,
              "description" to d.description,
              "introPeriod" to (trial?.billingPeriod ?: ""),
              "introIsFreeTrial" to (trial != null),
            )
          },
        )
      }
    }
  }

  private fun purchase(productId: String?, result: MethodChannel.Result) {
    if (productId.isNullOrEmpty()) {
      return fail(result, "store-error", "productId is required")
    }
    val act =
      activity
        ?: return fail(
          result,
          "unavailable",
          "No foreground activity to launch the billing flow from.",
        )
    if (pendingPurchase != null) {
      return fail(result, "store-error", "A purchase is already in progress.")
    }
    connect(onError = { fail(result, "unavailable", it) }) { client ->
      client.queryProductDetailsAsync(subsParams(listOf(productId))) { billingResult, details ->
        val product = details.firstOrNull()
        if (billingResult.responseCode != BillingClient.BillingResponseCode.OK || product == null) {
          fail(result, "not-found", "Product \"$productId\" was not found in the store.")
          return@queryProductDetailsAsync
        }
        val offerToken = product.subscriptionOfferDetails?.firstOrNull()?.offerToken
        if (offerToken == null) {
          fail(result, "not-found", "Product \"$productId\" has no subscription offer.")
          return@queryProductDetailsAsync
        }
        val flowParams =
          BillingFlowParams.newBuilder()
            .setProductDetailsParamsList(
              listOf(
                BillingFlowParams.ProductDetailsParams.newBuilder()
                  .setProductDetails(product)
                  .setOfferToken(offerToken)
                  .build()
              )
            )
            .build()
        mainHandler.post {
          pendingPurchase = result
          pendingProductId = productId
          val launch = client.launchBillingFlow(act, flowParams)
          if (launch.responseCode != BillingClient.BillingResponseCode.OK) {
            pendingPurchase = null
            pendingProductId = null
            result.error("store-error", launch.debugMessage, null)
          }
        }
      }
    }
  }

  override fun onPurchasesUpdated(result: BillingResult, purchases: MutableList<Purchase>?) {
    val reply = pendingPurchase
    val productId = pendingProductId
    pendingPurchase = null
    pendingProductId = null
    when (result.responseCode) {
      BillingClient.BillingResponseCode.OK -> {
        val purchase =
          purchases?.firstOrNull { productId == null || it.products.contains(productId) }
            ?: purchases?.firstOrNull()
        when {
          purchase == null ->
            reply?.let { fail(it, "store-error", "The store returned no purchase.") }
          purchase.purchaseState == Purchase.PurchaseState.PENDING ->
            // Slow payment methods: the completed purchase arrives through
            // this listener later and is forwarded as onTransactionUpdated.
            reply?.let { fail(it, "pending", "The purchase is awaiting payment.") }
          else -> {
            val payload =
              hashMapOf<String, Any?>(
                "productId" to (purchase.products.firstOrNull() ?: ""),
                "purchaseToken" to purchase.purchaseToken,
              )
            if (reply != null) {
              success(reply, payload)
            } else {
              // Out-of-band completion (pending payment confirmed, purchase
              // made outside a purchase() call).
              mainHandler.post { channel?.invokeMethod("onTransactionUpdated", payload) }
            }
          }
        }
      }
      BillingClient.BillingResponseCode.USER_CANCELED ->
        reply?.let { success(it, null) } // cancellation is a null receipt
      BillingClient.BillingResponseCode.ITEM_ALREADY_OWNED ->
        reply?.let { fail(it, "already-owned", "This subscription is already owned.") }
      BillingClient.BillingResponseCode.BILLING_UNAVAILABLE,
      BillingClient.BillingResponseCode.SERVICE_UNAVAILABLE,
      BillingClient.BillingResponseCode.SERVICE_DISCONNECTED ->
        reply?.let { fail(it, "unavailable", result.debugMessage) }
      else ->
        reply?.let {
          fail(it, "store-error", result.debugMessage.ifEmpty { "Purchase failed." })
        }
    }
  }

  private fun restore(result: MethodChannel.Result) {
    connect(onError = { fail(result, "unavailable", it) }) { client ->
      client.queryPurchasesAsync(
        QueryPurchasesParams.newBuilder()
          .setProductType(BillingClient.ProductType.SUBS)
          .build()
      ) { billingResult, purchases ->
        if (billingResult.responseCode != BillingClient.BillingResponseCode.OK) {
          fail(result, "store-error", billingResult.debugMessage)
          return@queryPurchasesAsync
        }
        success(
          result,
          purchases
            .filter { it.purchaseState == Purchase.PurchaseState.PURCHASED }
            .map { it.purchaseToken },
        )
      }
    }
  }

  // Billing callbacks may arrive off the platform thread; channel replies
  // must not.
  private fun success(result: MethodChannel.Result, value: Any?) =
    mainHandler.post { result.success(value) }

  private fun fail(result: MethodChannel.Result, code: String, message: String) =
    mainHandler.post { result.error(code, message, null) }
}
