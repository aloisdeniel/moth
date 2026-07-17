import type { Transport } from '@connectrpc/connect';
import { type MothConfig } from './config.js';
/** The request headers the SDK attaches to every RPC. */
export interface MothHeaderValues {
    /** The Bearer access token to attach, or undefined while signed out. */
    accessToken?: string;
}
/**
 * Creates the gRPC-Web transport for a moth endpoint. The server serves
 * gRPC-Web on the same port as everything else, so `endpoint` is simply the
 * instance's base URL.
 */
export declare function createMothTransport(config: MothConfig): Transport;
/**
 * Wraps a transport so every call carries the moth metadata: `x-moth-key`
 * (the publishable key), `x-moth-platform: web`, `x-moth-sdk-version`,
 * `x-moth-language` (pinned locale or live browser language, read lazily per
 * call), and `authorization: Bearer ...` while signed in. Works with any
 * inner transport — including `createRouterTransport` fakes in tests.
 */
export declare function withMothHeaders(inner: Transport, config: MothConfig, values: () => MothHeaderValues): Transport;
