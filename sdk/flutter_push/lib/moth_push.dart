/// First-party native push registration for moth: APNs on iOS, Firebase
/// Cloud Messaging on Android, behind `moth_auth`'s [MothPushAdapter]
/// interface.
///
/// Pass [MothNativePush] to `MothApp` and every signed-in device shows up in
/// the server's push registry — the plugin produces exactly the credential
/// `RegisterDevice` stores. Credentials only: notification display, tap
/// handling and routing stay in the app's hands.
library;

export 'src/moth_native_push.dart';
