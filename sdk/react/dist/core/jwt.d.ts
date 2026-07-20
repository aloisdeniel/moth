/** The decoded JWT payload of `token`, or `{}` for anything malformed. */
export declare function decodeJwtPayload(token: string): Record<string, unknown>;
/** The custom-claims object moth embeds under the `claims` claim, or `{}`. */
export declare function customClaimsOf(accessToken: string): Record<string, unknown>;
