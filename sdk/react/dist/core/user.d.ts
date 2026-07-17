/** The signed-in user's own account, as exposed to the app. */
export interface MothUser {
    id: string;
    email: string;
    emailVerified: boolean;
    displayName?: string;
    avatarUrl?: string;
    /** When the account was created (UTC), when the server reported it. */
    createTime?: Date;
    /**
     * The project-assigned custom claims (roles, permissions, ...) decoded
     * from the access token's `claims` claim — without signature
     * verification. Use them for client-side gating only; the developer's
     * backend must verify the JWT against the project JWKS and remains the
     * authority.
     */
    claims: Record<string, unknown>;
}
/**
 * The authentication state of a `MothClient`, as a discriminated union:
 *
 * ```ts
 * switch (state.status) {
 *   case 'loading':   // session restore in progress
 *   case 'signedOut':
 *   case 'signedIn':  // state.user
 * }
 * ```
 */
export type MothAuthState = {
    status: 'loading';
} | {
    status: 'signedOut';
} | {
    status: 'signedIn';
    user: MothUser;
};
export declare const mothAuthLoading: MothAuthState;
export declare const mothSignedOut: MothAuthState;
