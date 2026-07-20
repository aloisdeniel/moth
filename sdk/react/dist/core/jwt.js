// Decodes a JWT payload WITHOUT verifying the signature. The SDK only reads
// claims for client-side gating; the developer's backend verifies tokens
// against the project JWKS and remains the authority.
/** The decoded JWT payload of `token`, or `{}` for anything malformed. */
export function decodeJwtPayload(token) {
    const parts = token.split('.');
    if (parts.length !== 3)
        return {};
    try {
        const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
        const padded = base64 + '='.repeat((4 - (base64.length % 4)) % 4);
        const bytes = atob(padded);
        const buf = new Uint8Array(bytes.length);
        for (let i = 0; i < bytes.length; i++)
            buf[i] = bytes.charCodeAt(i);
        const decoded = JSON.parse(new TextDecoder().decode(buf));
        return isRecord(decoded) ? decoded : {};
    }
    catch {
        return {};
    }
}
/** The custom-claims object moth embeds under the `claims` claim, or `{}`. */
export function customClaimsOf(accessToken) {
    const claims = decodeJwtPayload(accessToken)['claims'];
    return isRecord(claims) ? claims : {};
}
function isRecord(v) {
    return typeof v === 'object' && v !== null && !Array.isArray(v);
}
