/**
 * A drop-in `fetch` wrapper that attaches `Authorization: Bearer <access
 * token>` — kept fresh via {@link MothClient.accessToken} — to every
 * request. Use it wherever the app calls its own backend; the backend
 * verifies the JWT against the project JWKS (and remains the authority).
 *
 * ```ts
 * const apiFetch = createMothFetch(moth)
 * const todos = await apiFetch('https://api.example.com/todos')
 * ```
 *
 * Throws the same "not signed in" error as `accessToken()` when signed out.
 */
export function createMothFetch(client, baseFetch = fetch) {
    return async (input, init) => {
        const token = await client.accessToken();
        const headers = new Headers(init?.headers ?? (input instanceof Request ? input.headers : undefined));
        headers.set('authorization', `Bearer ${token}`);
        return baseFetch(input, { ...init, headers });
    };
}
