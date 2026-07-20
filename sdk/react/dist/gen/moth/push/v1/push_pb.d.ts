import type { GenEnum, GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { Message } from "@bufbuild/protobuf";
/**
 * Describes the file moth/push/v1/push.proto.
 */
export declare const file_moth_push_v1_push: GenFile;
/**
 * PushDeviceMetadata is display metadata about the installation, for the
 * admin device panel and sender-side locale targeting. All fields optional.
 *
 * @generated from message moth.push.v1.PushDeviceMetadata
 */
export type PushDeviceMetadata = Message<"moth.push.v1.PushDeviceMetadata"> & {
    /**
     * OS family ("ios", "android", "web", "macos", ...). Display only — the
     * API to call lives in PushDevice.target.
     *
     * @generated from field: string platform = 1;
     */
    platform: string;
    /**
     * Device model (e.g. "iPhone16,1", "Pixel 9").
     *
     * @generated from field: string model = 2;
     */
    model: string;
    /**
     * OS version (e.g. "18.2").
     *
     * @generated from field: string os_version = 3;
     */
    osVersion: string;
    /**
     * App version the registration came from (e.g. "2.4.1+87").
     *
     * @generated from field: string app_version = 4;
     */
    appVersion: string;
    /**
     * BCP-47 locale of the device (e.g. "fr-FR"), for locale targeting.
     *
     * @generated from field: string locale = 5;
     */
    locale: string;
};
/**
 * Describes the message moth.push.v1.PushDeviceMetadata.
 * Use `create(PushDeviceMetadataSchema)` to create a new message.
 */
export declare const PushDeviceMetadataSchema: GenMessage<PushDeviceMetadata>;
/**
 * PushDevice is one stored registration as the client sees it. The push
 * credential itself is NOT echoed back: tokens are returned only over the
 * secret-key surface (moth.server.v1), and the client already holds its own.
 *
 * @generated from message moth.push.v1.PushDevice
 */
export type PushDevice = Message<"moth.push.v1.PushDevice"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: moth.push.v1.PushTarget target = 2;
     */
    target: PushTarget;
    /**
     * The client-generated stable installation id it registered under.
     *
     * @generated from field: string device_id = 3;
     */
    deviceId: string;
    /**
     * @generated from field: moth.push.v1.PushPermission permission = 4;
     */
    permission: PushPermission;
    /**
     * @generated from field: moth.push.v1.PushDeviceMetadata metadata = 5;
     */
    metadata?: PushDeviceMetadata | undefined;
    /**
     * @generated from field: google.protobuf.Timestamp create_time = 6;
     */
    createTime?: Timestamp | undefined;
    /**
     * @generated from field: google.protobuf.Timestamp update_time = 7;
     */
    updateTime?: Timestamp | undefined;
    /**
     * Refreshed on every RegisterDevice call.
     *
     * @generated from field: google.protobuf.Timestamp last_seen_time = 8;
     */
    lastSeenTime?: Timestamp | undefined;
};
/**
 * Describes the message moth.push.v1.PushDevice.
 * Use `create(PushDeviceSchema)` to create a new message.
 */
export declare const PushDeviceSchema: GenMessage<PushDevice>;
/**
 * @generated from message moth.push.v1.RegisterDeviceRequest
 */
export type RegisterDeviceRequest = Message<"moth.push.v1.RegisterDeviceRequest"> & {
    /**
     * Which push service `token` belongs to. Required.
     *
     * @generated from field: moth.push.v1.PushTarget target = 1;
     */
    target: PushTarget;
    /**
     * The push credential: APNs device token, FCM registration token, or the
     * serialized Web Push subscription (endpoint + keys). Required.
     *
     * @generated from field: string token = 2;
     */
    token: string;
    /**
     * Client-generated stable installation id, so one physical device
     * re-registering replaces its own row instead of accumulating. Required.
     *
     * @generated from field: string device_id = 3;
     */
    deviceId: string;
    /**
     * The OS-level permission state the client observed.
     *
     * @generated from field: moth.push.v1.PushPermission permission = 4;
     */
    permission: PushPermission;
    /**
     * @generated from field: moth.push.v1.PushDeviceMetadata metadata = 5;
     */
    metadata?: PushDeviceMetadata | undefined;
};
/**
 * Describes the message moth.push.v1.RegisterDeviceRequest.
 * Use `create(RegisterDeviceRequestSchema)` to create a new message.
 */
export declare const RegisterDeviceRequestSchema: GenMessage<RegisterDeviceRequest>;
/**
 * @generated from message moth.push.v1.RegisterDeviceResponse
 */
export type RegisterDeviceResponse = Message<"moth.push.v1.RegisterDeviceResponse"> & {
    /**
     * The stored registration (created or upserted).
     *
     * @generated from field: moth.push.v1.PushDevice device = 1;
     */
    device?: PushDevice | undefined;
};
/**
 * Describes the message moth.push.v1.RegisterDeviceResponse.
 * Use `create(RegisterDeviceResponseSchema)` to create a new message.
 */
export declare const RegisterDeviceResponseSchema: GenMessage<RegisterDeviceResponse>;
/**
 * @generated from message moth.push.v1.UnregisterDeviceRequest
 */
export type UnregisterDeviceRequest = Message<"moth.push.v1.UnregisterDeviceRequest"> & {
    /**
     * The installation id passed to RegisterDevice.
     *
     * @generated from field: string device_id = 1;
     */
    deviceId: string;
};
/**
 * Describes the message moth.push.v1.UnregisterDeviceRequest.
 * Use `create(UnregisterDeviceRequestSchema)` to create a new message.
 */
export declare const UnregisterDeviceRequestSchema: GenMessage<UnregisterDeviceRequest>;
/**
 * @generated from message moth.push.v1.UnregisterDeviceResponse
 */
export type UnregisterDeviceResponse = Message<"moth.push.v1.UnregisterDeviceResponse"> & {};
/**
 * Describes the message moth.push.v1.UnregisterDeviceResponse.
 * Use `create(UnregisterDeviceResponseSchema)` to create a new message.
 */
export declare const UnregisterDeviceResponseSchema: GenMessage<UnregisterDeviceResponse>;
/**
 * PushTarget says which push service the credential belongs to — i.e. which
 * API the developer's backend must call to reach the device. Deliberately not
 * the platform: an iOS app using FCM registers as PUSH_TARGET_FCM; the OS
 * lives in PushDeviceMetadata.platform as display metadata.
 *
 * @generated from enum moth.push.v1.PushTarget
 */
export declare enum PushTarget {
    /**
     * @generated from enum value: PUSH_TARGET_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * Apple Push Notification service; token is the APNs device token.
     *
     * @generated from enum value: PUSH_TARGET_APNS = 1;
     */
    APNS = 1,
    /**
     * Firebase Cloud Messaging; token is the FCM registration token.
     *
     * @generated from enum value: PUSH_TARGET_FCM = 2;
     */
    FCM = 2,
    /**
     * Web Push; token is the serialized subscription (endpoint + keys).
     *
     * @generated from enum value: PUSH_TARGET_WEBPUSH = 3;
     */
    WEBPUSH = 3
}
/**
 * Describes the enum moth.push.v1.PushTarget.
 */
export declare const PushTargetSchema: GenEnum<PushTarget>;
/**
 * PushPermission is the OS-level notification-permission state the client
 * last reported. A registration with a token but DENIED permission is kept
 * (data pushes may still work) but flagged, so senders can skip alert pushes.
 *
 * @generated from enum moth.push.v1.PushPermission
 */
export declare enum PushPermission {
    /**
     * @generated from enum value: PUSH_PERMISSION_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * @generated from enum value: PUSH_PERMISSION_GRANTED = 1;
     */
    GRANTED = 1,
    /**
     * iOS provisional authorization (quiet notifications).
     *
     * @generated from enum value: PUSH_PERMISSION_PROVISIONAL = 2;
     */
    PROVISIONAL = 2,
    /**
     * @generated from enum value: PUSH_PERMISSION_DENIED = 3;
     */
    DENIED = 3,
    /**
     * The client could not determine the permission state.
     *
     * @generated from enum value: PUSH_PERMISSION_UNKNOWN = 4;
     */
    UNKNOWN = 4
}
/**
 * Describes the enum moth.push.v1.PushPermission.
 */
export declare const PushPermissionSchema: GenEnum<PushPermission>;
/**
 * PushService is the client-facing push-device registry, consumed by the moth
 * SDKs (milestone 21). Authenticated exactly like BillingService: every call
 * carries the project publishable key in `x-moth-key: pk_...` request metadata
 * AND a user access token in the `Authorization: Bearer <jwt>` header — a
 * registration always hangs off (project, signed-in user); there are no
 * anonymous device registrations.
 *
 * moth registers; the developer's backend sends. moth never talks to
 * APNs/FCM/Web Push — it tracks each registration's target, permission state
 * and liveness, and hands the current set to the developer's backend through
 * moth.server.v1.PushService, which delivers. Rate-limited like the other
 * credential-facing RPCs (milestone 02).
 *
 * @generated from service moth.push.v1.PushService
 */
export declare const PushService: GenService<{
    /**
     * RegisterDevice upserts the calling user's registration and returns the
     * stored row. Idempotent by design — the SDK calls it on every app launch,
     * token rotation and permission change, without bookkeeping:
     *   - same device_id → the device's existing row is replaced (a rotated
     *     token supersedes the old one, revoked `replaced`);
     *   - same (target, token) under a new user → the newest owner wins (the
     *     previous user's row is revoked `replaced`), which handles a device
     *     changing accounts on sign-in;
     *   - otherwise → a new registration is created.
     * Every call refreshes last_seen_time, so it doubles as the liveness
     * heartbeat feeding the staleness sweep.
     *
     * @generated from rpc moth.push.v1.PushService.RegisterDevice
     */
    registerDevice: {
        methodKind: "unary";
        input: typeof RegisterDeviceRequestSchema;
        output: typeof RegisterDeviceResponseSchema;
    };
    /**
     * UnregisterDevice revokes the calling user's registration for one
     * installation (`signed_out`); the SDKs call it on sign-out. Idempotent:
     * unknown or already-revoked device ids succeed.
     *
     * @generated from rpc moth.push.v1.PushService.UnregisterDevice
     */
    unregisterDevice: {
        methodKind: "unary";
        input: typeof UnregisterDeviceRequestSchema;
        output: typeof UnregisterDeviceResponseSchema;
    };
}>;
