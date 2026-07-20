import 'push_device_id.dart';

/// Web (and other io-less platforms) keep the id in memory: the stable
/// installation id for Web Push lands with the browser flow in `@moth/react`;
/// on Flutter Web a fresh id per session is safe — the server's upsert
/// semantics replace by `(target, token)` too, so no rows accumulate.
MothPushDeviceIdStore createPushDeviceIdStore(String publishableKey) =>
    MothMemoryPushDeviceIdStore();
