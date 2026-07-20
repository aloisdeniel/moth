import type { MothCopy } from '../core/copy.js';
/**
 * End-user copy for a {@link MothError} (or any error), used by the SDK's
 * built-in screens. Exposed so custom login UIs can reuse the same wording.
 *
 * Every mapped error resolves from the localized catalog — wrong
 * credentials and an already-registered email share the login-form keys,
 * the rest use the shared `error.*` group. The two cases that echo a
 * server-supplied message (a weak-password policy and the generic
 * MothError) are already localized by the server.
 */
export declare function friendlyMothErrorMessage(error: unknown, copy: MothCopy): string;
