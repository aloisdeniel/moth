import 'dart:async';

import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

import '../client.dart';
import '../customer_info.dart';
import '../exceptions.dart';
import '../offering.dart';
import '../paywall_cache.dart';
import '../purchase.dart';
import '../theme.dart';
import 'billing_adapter.dart';
import 'moth_logo.dart';
import 'moth_scope.dart';
import 'moth_theme_scope.dart';
import 'purchase_flow.dart';

/// Store account/subscription management deep links, opened by "Manage
/// subscription".
const _appleManageUrl = 'https://apps.apple.com/account/subscriptions';
const _googleManageUrl = 'https://play.google.com/store/account/subscriptions';

/// Batteries-included, Material paywall screen — the purchasing counterpart to
/// [MothLoginScreen].
///
/// Renders the project's default offering ([MothClient.getOfferings]) styled by
/// the admin-configured paywall copy ([MothClient.getPaywall]): a header
/// (logo + headline + subtitle), a benefit list, one card per tier (price,
/// trial badge, "most popular" highlight), a primary purchase button, restore,
/// and terms/privacy + manage-subscription links. It **consumes [MothTheme]
/// exclusively** (colors, typography, radius, spacing, logo), so it matches the
/// login screen and the brand with no hardcoded styling.
///
/// Drop it in wherever a feature is gated (or via
/// `MothApp(requiresEntitlement: 'pro')`). The building blocks
/// ([MothPaywallHeader], [MothTierCard], [MothPurchaseButton]) are exported for
/// custom paywalls; pass [config] to override the delivered copy, or [theme] to
/// pin a theme.
class MothPaywallScreen extends StatefulWidget {
  const MothPaywallScreen({
    super.key,
    this.client,
    this.adapter,
    this.config,
    this.paywallCache,
    this.theme,
    this.onPurchased,
    this.onClose,
  });

  /// Client override for standalone use; defaults to the enclosing
  /// [MothScope]'s client.
  final MothClient? client;

  /// Billing-adapter override; defaults to the adapter passed to [MothApp].
  final MothBillingAdapter? adapter;

  /// Paywall copy/layout override; when null it is fetched from the server
  /// (`GetPaywall`) and cached by revision.
  final MothPaywall? config;

  /// Revision cache override for the server-delivered paywall config
  /// (defaults to a device file cache; useful for tests). Ignored when
  /// [config] is supplied.
  final MothPaywallCache? paywallCache;

  /// Theme override: wins over the enclosing [MothThemeScope] / server theme.
  final MothTheme? theme;

  /// Called after a successful purchase. When gated by
  /// `MothApp(requiresEntitlement:)` the swap to the child happens
  /// automatically; use this for standalone use (e.g. to pop the route).
  final VoidCallback? onPurchased;

  /// Called when the user dismisses the paywall; when set, a close button is
  /// shown in the header.
  final VoidCallback? onClose;

  // Stable keys for widget tests.
  static const headlineKey = Key('moth-paywall-headline');
  static const restoreKey = Key('moth-paywall-restore');
  static const purchaseButtonKey = Key('moth-paywall-purchase');
  static const emptyStateKey = Key('moth-paywall-empty');
  static const errorStateKey = Key('moth-paywall-error');
  static const bannerKey = Key('moth-paywall-banner');
  static const manageKey = Key('moth-paywall-manage');
  static const closeKey = Key('moth-paywall-close');

  static Key tierCardKey(String identifier) =>
      Key('moth-paywall-tier-$identifier');

  @override
  State<MothPaywallScreen> createState() => _MothPaywallScreenState();
}

class _MothPaywallScreenState extends State<MothPaywallScreen> {
  static const _maxContentWidth = 480.0;

  MothClient? _client;
  MothPaywallCache? _cache;
  MothPaywall? _paywall;
  MothOffering? _offering;
  Map<String, MothStoreProduct> _storeProducts = const {};
  MothTheme? _serverTheme;
  bool _loading = true;
  bool _failed = false;
  String? _selectedId;
  bool _busy = false;
  String? _message;

  MothClient get client => _client!;

  MothBillingAdapter? get _adapter =>
      widget.adapter ?? MothScope.maybeOf(context)?.billingAdapter;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_client == null) {
      _client = widget.client ?? MothScope.maybeOf(context)?.client;
      if (_client == null) {
        throw FlutterError(
          'MothPaywallScreen has no MothClient.\n'
          'Place it under MothApp, or pass the client parameter.',
        );
      }
      _load();
    }
  }

  Future<void> _load() async {
    // Read anything context-dependent before the first await (BuildContext is
    // not safe to use across async gaps).
    final needsThemeFetch =
        widget.theme == null && MothThemeScope.maybeOf(context) == null;
    final adapter = _adapter;
    // The paywall config is cached client-side by revision (stale-while-
    // revalidate, like the theme): echo the cached revision so the server can
    // omit an unchanged body, and persist every fresh config. Skipped when a
    // config is supplied directly.
    final cache = widget.config == null
        ? (_cache ??=
              widget.paywallCache ??
              defaultPaywallCache(client.config.publishableKey))
        : null;
    setState(() {
      _loading = true;
      _failed = false;
    });
    try {
      MothPaywall? cached;
      if (cache != null) {
        try {
          cached = await cache.load();
        } on Object {
          // Broken cache — treat as a miss.
        }
      }
      final MothPaywall paywall;
      if (widget.config != null) {
        paywall = widget.config!;
      } else {
        final fetched = await client.getPaywall(
          knownPaywallRevision: cached?.revisionId ?? '',
        );
        // A null response means the cached revision still matches: keep it.
        paywall = fetched ?? cached ?? const MothPaywall();
        if (fetched != null && cache != null) {
          unawaited(_saveCache(cache, fetched));
        }
      }
      final offering = await client.getOfferings(offering: paywall.offering);
      // Store-localized prices: ask the adapter for the native store products
      // so the tier cards show the price the user will actually be charged.
      // Optional — an adapter that doesn't implement productsFor returns an
      // empty list and the cards fall back to the catalog price.
      var storeProducts = const <String, MothStoreProduct>{};
      if (adapter != null) {
        try {
          final list = await adapter.productsFor(offering);
          if (list.isNotEmpty) {
            storeProducts = {for (final p in list) p.productIdentifier: p};
          }
        } on Object {
          // Non-fatal: fall back to the catalog price/period.
        }
      }
      // Fetch the project theme only as a standalone fallback: under MothApp
      // (or any MothThemeScope) the theme is already provided.
      MothTheme? serverTheme;
      if (needsThemeFetch) {
        try {
          serverTheme = (await client.getProjectConfig()).theme;
        } on MothException {
          // Non-fatal: the fallback theme still renders.
        }
      }
      if (!mounted) return;
      setState(() {
        _paywall = paywall;
        _offering = offering;
        _storeProducts = storeProducts;
        _serverTheme = serverTheme;
        _selectedId = _defaultSelection(paywall, offering);
        _loading = false;
      });
    } on MothException {
      if (mounted) {
        setState(() {
          _loading = false;
          _failed = true;
        });
      }
    }
  }

  /// Persists the fetched config best-effort: a broken cache (unavailable
  /// storage) must never surface as an unhandled error.
  Future<void> _saveCache(MothPaywallCache cache, MothPaywall paywall) async {
    try {
      await cache.save(paywall);
    } on Object {
      // Best effort — the config is re-delivered next launch.
    }
  }

  static String? _defaultSelection(MothPaywall paywall, MothOffering offering) {
    if (offering.products.isEmpty) return null;
    final highlighted = paywall.highlightedProductIdentifier;
    if (highlighted.isNotEmpty && offering.productById(highlighted) != null) {
      return highlighted;
    }
    final flagged = offering.products.where((p) => p.highlighted).firstOrNull;
    return (flagged ?? offering.products.first).identifier;
  }

  Future<void> _purchase() async {
    final offering = _offering;
    final selectedId = _selectedId;
    if (offering == null || selectedId == null) return;
    final product = offering.productById(selectedId);
    final adapter = _adapter;
    if (product == null) return;
    if (adapter == null) {
      setState(
        () => _message =
            'In-app purchases are not wired up in this app: pass a '
            'MothBillingAdapter to MothApp or MothPaywallScreen.',
      );
      return;
    }
    setState(() {
      _busy = true;
      _message = null;
    });
    final result = await runMothPurchase(client, adapter, product);
    if (!mounted) return;
    setState(() => _busy = false);
    switch (result) {
      case MothPurchasePurchased():
        widget.onPurchased?.call();
      case MothPurchasePending():
        setState(
          () => _message =
              'Your purchase is pending approval. It will unlock once '
              'confirmed.',
        );
      case MothPurchaseAlreadyOwned():
        setState(
          () => _message =
              'You already own this subscription — tap Restore to re-link it.',
        );
      case MothPurchaseCancelled():
        break;
      case MothPurchaseError(:final message):
        setState(() => _message = message);
    }
  }

  Future<void> _restore() async {
    final adapter = _adapter;
    if (adapter == null) {
      setState(
        () => _message =
            'Restoring purchases needs a MothBillingAdapter — pass one to '
            'MothApp or MothPaywallScreen.',
      );
      return;
    }
    setState(() {
      _busy = true;
      _message = null;
    });
    try {
      final info = await runMothRestore(client, adapter);
      if (!mounted) return;
      setState(() {
        _busy = false;
        _message = info.activeEntitlements.isEmpty
            ? 'No previous purchases were found to restore.'
            : 'Your purchases were restored.';
      });
    } on MothException catch (err) {
      if (mounted) {
        setState(() {
          _busy = false;
          _message = err.message;
        });
      }
    } on Object catch (err) {
      if (mounted) {
        setState(() {
          _busy = false;
          _message = 'Could not restore purchases: $err';
        });
      }
    }
  }

  Future<void> _openLink(String url) async {
    final uri = Uri.tryParse(url);
    if (uri == null) return;
    try {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    } on Object {
      // No browser/handler available.
    }
  }

  @override
  Widget build(BuildContext context) {
    final moth =
        widget.theme ??
        MothThemeScope.maybeOf(context) ??
        _serverTheme ??
        MothTheme.fallback();
    final brightness =
        MediaQuery.maybePlatformBrightnessOf(context) ?? Brightness.light;
    return MothThemeScope(
      theme: moth,
      child: Theme(
        data: moth.toThemeData(brightness),
        child: Builder(builder: (context) => _buildScreen(context, moth)),
      ),
    );
  }

  Widget _buildScreen(BuildContext context, MothTheme moth) {
    final theme = Theme.of(context);
    final Widget content;
    if (_loading) {
      content = Padding(
        padding: EdgeInsets.all(moth.space(4)),
        child: const Center(child: CircularProgressIndicator()),
      );
    } else if (_failed) {
      content = _buildError(theme, moth);
    } else {
      content = _buildContent(theme, moth);
    }
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: EdgeInsets.all(moth.space(3)),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: _maxContentWidth),
              child: content,
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildContent(ThemeData theme, MothTheme moth) {
    final paywall = _paywall!;
    final offering = _offering!;
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (widget.onClose != null)
          Align(
            alignment: Alignment.centerRight,
            child: IconButton(
              key: MothPaywallScreen.closeKey,
              icon: const Icon(Icons.close),
              onPressed: widget.onClose,
            ),
          ),
        MothPaywallHeader(
          headline: paywall.headline,
          subtitle: paywall.subtitle,
          theme: moth,
        ),
        if (paywall.benefits.isNotEmpty) ...[
          SizedBox(height: moth.space(3)),
          ..._buildBenefits(theme, moth, paywall.benefits),
        ],
        SizedBox(height: moth.space(3)),
        if (offering.isEmpty)
          _buildEmpty(theme, moth)
        else ...[
          ..._buildTiers(theme, moth, paywall, offering),
          SizedBox(height: moth.space(1)),
          if (_message != null) _buildBanner(theme, moth, _message!),
          MothPurchaseButton(
            key: MothPaywallScreen.purchaseButtonKey,
            product: offering.productById(_selectedId ?? ''),
            busy: _busy,
            onPressed: _busy ? null : _purchase,
            theme: moth,
          ),
          TextButton(
            key: MothPaywallScreen.restoreKey,
            onPressed: _busy ? null : _restore,
            child: const Text('Restore purchases'),
          ),
        ],
        ..._buildFooter(theme, moth, paywall),
      ],
    );
  }

  /// The tier section, honoring [MothPaywall.layout]. `tiles` and `list` show
  /// every tier as a stacked, selectable card (a full-width row is the natural
  /// mobile form of both); `compact` collapses to a single selected tier with
  /// a period toggle to switch between the offering's products.
  List<Widget> _buildTiers(
    ThemeData theme,
    MothTheme moth,
    MothPaywall paywall,
    MothOffering offering,
  ) {
    if (paywall.layout == MothPaywallLayout.compact &&
        offering.products.length > 1) {
      return _buildCompactTiers(theme, moth, offering);
    }
    return [
      for (final product in offering.products)
        Padding(
          padding: EdgeInsets.only(bottom: moth.space(1.5)),
          child: MothTierCard(
            key: MothPaywallScreen.tierCardKey(product.identifier),
            product: product,
            storeProduct: _storeProducts[product.identifier],
            selected: product.identifier == _selectedId,
            onTap: _busy
                ? null
                : () => setState(() => _selectedId = product.identifier),
            theme: moth,
          ),
        ),
    ];
  }

  List<Widget> _buildCompactTiers(
    ThemeData theme,
    MothTheme moth,
    MothOffering offering,
  ) {
    final selectedId = _selectedId ?? offering.products.first.identifier;
    final selected =
        offering.productById(selectedId) ?? offering.products.first;
    return [
      Padding(
        padding: EdgeInsets.only(bottom: moth.space(1.5)),
        child: SegmentedButton<String>(
          showSelectedIcon: false,
          segments: [
            for (final product in offering.products)
              ButtonSegment<String>(
                value: product.identifier,
                label: Text(_compactSegmentLabel(product)),
              ),
          ],
          selected: {selected.identifier},
          onSelectionChanged: _busy
              ? null
              : (ids) => setState(() => _selectedId = ids.first),
        ),
      ),
      MothTierCard(
        key: MothPaywallScreen.tierCardKey(selected.identifier),
        product: selected,
        storeProduct: _storeProducts[selected.identifier],
        selected: true,
        theme: moth,
      ),
    ];
  }

  List<Widget> _buildBenefits(
    ThemeData theme,
    MothTheme moth,
    List<String> benefits,
  ) {
    return [
      for (final benefit in benefits)
        Padding(
          padding: EdgeInsets.only(bottom: moth.space(1)),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Icon(
                Icons.check_circle,
                size: moth.space(2.5),
                color: theme.colorScheme.primary,
              ),
              SizedBox(width: moth.space(1.5)),
              Expanded(child: Text(benefit, style: theme.textTheme.bodyMedium)),
            ],
          ),
        ),
    ];
  }

  Widget _buildEmpty(ThemeData theme, MothTheme moth) {
    return Column(
      key: MothPaywallScreen.emptyStateKey,
      children: [
        Icon(
          Icons.shopping_bag_outlined,
          size: moth.space(6),
          color: theme.colorScheme.onSurfaceVariant,
        ),
        SizedBox(height: moth.space(2)),
        Text(
          'Nothing to purchase yet',
          textAlign: TextAlign.center,
          style: theme.textTheme.titleLarge,
        ),
        SizedBox(height: moth.space(1)),
        Text(
          'There are no subscriptions available right now. Check back later.',
          textAlign: TextAlign.center,
          style: theme.textTheme.bodyMedium?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        SizedBox(height: moth.space(2)),
        TextButton(
          key: MothPaywallScreen.restoreKey,
          onPressed: _busy ? null : _restore,
          child: const Text('Restore purchases'),
        ),
      ],
    );
  }

  Widget _buildError(ThemeData theme, MothTheme moth) {
    return Column(
      key: MothPaywallScreen.errorStateKey,
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(
          Icons.cloud_off,
          size: moth.space(6),
          color: theme.colorScheme.onSurfaceVariant,
        ),
        SizedBox(height: moth.space(2)),
        Text(
          'Cannot reach the store',
          textAlign: TextAlign.center,
          style: theme.textTheme.titleLarge,
        ),
        SizedBox(height: moth.space(1)),
        Text(
          'Subscription options could not be loaded. Check your connection '
          'and try again.',
          textAlign: TextAlign.center,
          style: theme.textTheme.bodyMedium,
        ),
        SizedBox(height: moth.space(3)),
        FilledButton(onPressed: _load, child: const Text('Try again')),
      ],
    );
  }

  Widget _buildBanner(ThemeData theme, MothTheme moth, String text) {
    return Padding(
      padding: EdgeInsets.only(bottom: moth.space(1.5)),
      child: Material(
        key: MothPaywallScreen.bannerKey,
        color: theme.colorScheme.secondaryContainer,
        borderRadius: BorderRadius.circular(moth.cornerRadius),
        child: Padding(
          padding: EdgeInsets.all(moth.space(1.5)),
          child: Text(
            text,
            style: TextStyle(color: theme.colorScheme.onSecondaryContainer),
          ),
        ),
      ),
    );
  }

  List<Widget> _buildFooter(
    ThemeData theme,
    MothTheme moth,
    MothPaywall paywall,
  ) {
    final scope = MothScope.maybeOf(context);
    final hasSubscription =
        scope?.customerInfo.subscriptions.isNotEmpty ?? false;
    final manageStore = scope?.customerInfo.subscriptions.firstOrNull?.store;
    final links = <(Key, String, String)>[
      if (paywall.termsUrl != null)
        (
          const Key('moth-paywall-terms'),
          'Terms of Service',
          paywall.termsUrl!,
        ),
      if (paywall.privacyUrl != null)
        (
          const Key('moth-paywall-privacy'),
          'Privacy Policy',
          paywall.privacyUrl!,
        ),
    ];
    if (!hasSubscription && links.isEmpty) return const [];
    final linkStyle = TextButton.styleFrom(
      textStyle: theme.textTheme.bodySmall,
      foregroundColor: theme.colorScheme.onSurfaceVariant,
      minimumSize: Size(0, moth.space(4)),
      padding: EdgeInsets.symmetric(horizontal: moth.space(1)),
    );
    return [
      SizedBox(height: moth.space(2)),
      if (hasSubscription)
        TextButton(
          key: MothPaywallScreen.manageKey,
          style: linkStyle,
          onPressed: () => _openLink(
            manageStore == MothStore.google
                ? _googleManageUrl
                : _appleManageUrl,
          ),
          child: const Text('Manage subscription'),
        ),
      Wrap(
        alignment: WrapAlignment.center,
        crossAxisAlignment: WrapCrossAlignment.center,
        children: [
          for (final (index, (key, label, url)) in links.indexed) ...[
            if (index > 0) Text('·', style: theme.textTheme.bodySmall),
            TextButton(
              key: key,
              style: linkStyle,
              onPressed: () => _openLink(url),
              child: Text(label),
            ),
          ],
        ],
      ),
    ];
  }
}

/// The paywall header: the project logo (when set), the headline and the
/// subtitle, styled from [MothTheme]. Exported for custom paywalls.
class MothPaywallHeader extends StatelessWidget {
  const MothPaywallHeader({
    super.key,
    required this.headline,
    this.subtitle = '',
    this.theme,
  });

  final String headline;
  final String subtitle;

  /// Theme override; defaults to the enclosing [MothThemeScope].
  final MothTheme? theme;

  @override
  Widget build(BuildContext context) {
    final moth = theme ?? MothThemeScope.of(context);
    final data = Theme.of(context);
    final hasLogo = moth.logoLightUrl != null || moth.logoDarkUrl != null;
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (hasLogo) ...[
          MothLogo(theme: moth),
          SizedBox(height: moth.space(3)),
        ],
        Text(
          headline,
          key: MothPaywallScreen.headlineKey,
          textAlign: TextAlign.center,
          style: data.textTheme.headlineMedium,
        ),
        if (subtitle.isNotEmpty) ...[
          SizedBox(height: moth.space(1)),
          Text(
            subtitle,
            textAlign: TextAlign.center,
            style: data.textTheme.bodyMedium?.copyWith(
              color: data.colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ],
    );
  }
}

/// One selectable subscription tier card: name, price/period, a trial badge
/// and the "most popular" highlight. Styled from [MothTheme]. Exported for
/// custom paywalls.
class MothTierCard extends StatelessWidget {
  const MothTierCard({
    super.key,
    required this.product,
    this.storeProduct,
    this.selected = false,
    this.onTap,
    this.theme,
  });

  final MothOfferingProduct product;

  /// The native store product for [product], when the billing adapter supplied
  /// one: its localized, formatted price is authoritative and shown in place
  /// of the catalog price.
  final MothStoreProduct? storeProduct;

  final bool selected;
  final VoidCallback? onTap;

  /// Theme override; defaults to the enclosing [MothThemeScope].
  final MothTheme? theme;

  @override
  Widget build(BuildContext context) {
    final moth = theme ?? MothThemeScope.of(context);
    final data = Theme.of(context);
    final scheme = data.colorScheme;
    final highlighted = product.highlighted;
    final borderColor = selected
        ? scheme.primary
        : (highlighted
              ? scheme.primary.withValues(alpha: 0.5)
              : scheme.outlineVariant);
    return Material(
      color: selected ? scheme.primary.withValues(alpha: 0.08) : scheme.surface,
      borderRadius: BorderRadius.circular(moth.cornerRadius),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(moth.cornerRadius),
        child: Container(
          padding: EdgeInsets.all(moth.space(2)),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(moth.cornerRadius),
            border: Border.all(color: borderColor, width: selected ? 2 : 1),
          ),
          child: Row(
            children: [
              Icon(
                selected
                    ? Icons.radio_button_checked
                    : Icons.radio_button_unchecked,
                color: selected ? scheme.primary : scheme.outline,
                size: moth.space(3),
              ),
              SizedBox(width: moth.space(1.5)),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      crossAxisAlignment: CrossAxisAlignment.baseline,
                      textBaseline: TextBaseline.alphabetic,
                      children: [
                        Expanded(
                          child: Text(
                            product.displayName.isEmpty
                                ? product.identifier
                                : product.displayName,
                            style: data.textTheme.titleMedium,
                          ),
                        ),
                        SizedBox(width: moth.space(1)),
                        Text(
                          _priceLabel(product, storeProduct),
                          textAlign: TextAlign.end,
                          style: data.textTheme.titleMedium?.copyWith(
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ],
                    ),
                    if (highlighted || product.hasTrial) ...[
                      SizedBox(height: moth.space(0.75)),
                      Wrap(
                        spacing: moth.space(1),
                        runSpacing: moth.space(0.5),
                        children: [
                          if (highlighted)
                            _Badge(
                              label: 'Most popular',
                              background: scheme.primary,
                              foreground: scheme.onPrimary,
                              moth: moth,
                            ),
                          if (product.hasTrial)
                            _Badge(
                              label: _trialLabel(product.trialPeriod),
                              background: scheme.secondaryContainer,
                              foreground: scheme.onSecondaryContainer,
                              moth: moth,
                            ),
                        ],
                      ),
                    ],
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// The primary purchase button, labelled from the selected [product] (e.g.
/// "Start free trial" / "Subscribe"). Styled from [MothTheme].
class MothPurchaseButton extends StatelessWidget {
  const MothPurchaseButton({
    super.key,
    required this.product,
    this.busy = false,
    this.onPressed,
    this.label,
    this.theme,
  });

  final MothOfferingProduct? product;
  final bool busy;
  final VoidCallback? onPressed;

  /// Label override; defaults to a trial-aware label from [product].
  final String? label;

  /// Theme override; defaults to the enclosing [MothThemeScope].
  final MothTheme? theme;

  @override
  Widget build(BuildContext context) {
    final moth = theme ?? MothThemeScope.of(context);
    final text =
        label ??
        (product?.hasTrial ?? false ? 'Start free trial' : 'Subscribe');
    return FilledButton(
      onPressed: busy ? null : onPressed,
      child: busy
          ? SizedBox.square(
              dimension: moth.space(2.25),
              child: const CircularProgressIndicator(strokeWidth: 2),
            )
          : Text(text),
    );
  }
}

class _Badge extends StatelessWidget {
  const _Badge({
    required this.label,
    required this.background,
    required this.foreground,
    required this.moth,
  });

  final String label;
  final Color background;
  final Color foreground;
  final MothTheme moth;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: EdgeInsets.symmetric(
        horizontal: moth.space(1),
        vertical: moth.space(0.25),
      ),
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(moth.cornerRadius / 2),
      ),
      child: Text(
        label,
        style: Theme.of(context).textTheme.labelSmall?.copyWith(
          color: foreground,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

/// Formats a tier's price with the billing period suffix (e.g.
/// `$9.99 / month`). Prefers the native store's localized, formatted price
/// ([MothStoreProduct.price]) — the amount actually charged — and falls back
/// to the catalog micros + currency when no store product is available.
String _priceLabel(MothOfferingProduct product, [MothStoreProduct? store]) {
  final String price;
  if (store != null && store.price.isNotEmpty) {
    price = store.price;
  } else if (product.priceAmountMicros <= 0) {
    return '—';
  } else {
    final amount = product.priceAmountMicros / 1000000;
    final symbol = _currencySymbol(product.currency);
    final formatted = amount == amount.roundToDouble()
        ? amount.toStringAsFixed(0)
        : amount.toStringAsFixed(2);
    price = symbol.isEmpty
        ? '$formatted ${product.currency}'.trim()
        : '$symbol$formatted';
  }
  final period = _periodSuffix(product.billingPeriod);
  return period.isEmpty ? price : '$price / $period';
}

/// A short label for a compact-layout period toggle: the capitalized period
/// (e.g. `Month`, `Year`) when known, otherwise the tier's display name.
String _compactSegmentLabel(MothOfferingProduct product) {
  final period = _periodSuffix(product.billingPeriod);
  if (period.isNotEmpty) {
    return period[0].toUpperCase() + period.substring(1);
  }
  return product.displayName.isEmpty ? product.identifier : product.displayName;
}

String _currencySymbol(String currency) => switch (currency.toUpperCase()) {
  'USD' || 'AUD' || 'CAD' || 'NZD' => r'$',
  'EUR' => '€',
  'GBP' => '£',
  'JPY' || 'CNY' => '¥',
  _ => '',
};

/// Maps an ISO-8601 recurrence (e.g. `P1M`, `P1Y`, `P1W`) to a short suffix.
String _periodSuffix(String period) => switch (period.toUpperCase()) {
  'P1W' => 'week',
  'P1M' => 'month',
  'P3M' => 'quarter',
  'P6M' => '6 months',
  'P1Y' => 'year',
  _ => '',
};

/// Human-readable trial badge (e.g. `P1W` → `1-week free trial`).
String _trialLabel(String period) => switch (period.toUpperCase()) {
  'P3D' => '3-day free trial',
  'P1W' || 'P7D' => '1-week free trial',
  'P2W' || 'P14D' => '2-week free trial',
  'P1M' => '1-month free trial',
  _ => 'Free trial',
};
