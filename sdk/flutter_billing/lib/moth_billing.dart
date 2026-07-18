/// First-party native billing for moth: StoreKit 2 on iOS, Play Billing on
/// Android, behind `moth_auth`'s [MothBillingAdapter] interface.
///
/// Pass [MothStoreBilling] to `MothApp` and the paywall sells — the plugin
/// produces exactly the signed transaction / purchase token the moth server
/// validates.
library;

export 'src/moth_store_billing.dart';
