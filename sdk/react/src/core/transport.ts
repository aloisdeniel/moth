import type {
  DescMessage,
  DescMethodStreaming,
  DescMethodUnary,
  MessageInitShape,
} from '@bufbuild/protobuf'
import type { ContextValues, Transport } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { mothSdkVersion } from '../version.js'
import { currentLocaleOf, type MothConfig } from './config.js'

/** The request headers the SDK attaches to every RPC. */
export interface MothHeaderValues {
  /** The Bearer access token to attach, or undefined while signed out. */
  accessToken?: string
}

/**
 * Creates the gRPC-Web transport for a moth endpoint. The server serves
 * gRPC-Web on the same port as everything else, so `endpoint` is simply the
 * instance's base URL.
 */
export function createMothTransport(config: MothConfig): Transport {
  return createConnectTransport({ baseUrl: config.endpoint })
}

/**
 * Wraps a transport so every call carries the moth metadata: `x-moth-key`
 * (the publishable key), `x-moth-platform: web`, `x-moth-sdk-version`,
 * `x-moth-language` (pinned locale or live browser language, read lazily per
 * call), and `authorization: Bearer ...` while signed in. Works with any
 * inner transport — including `createRouterTransport` fakes in tests.
 */
export function withMothHeaders(
  inner: Transport,
  config: MothConfig,
  values: () => MothHeaderValues,
): Transport {
  const attach = (header: HeadersInit | undefined): Headers => {
    const headers = new Headers(header)
    headers.set('x-moth-key', config.publishableKey)
    headers.set('x-moth-platform', 'web')
    headers.set('x-moth-sdk-version', mothSdkVersion)
    headers.set('x-moth-language', currentLocaleOf(config))
    const { accessToken } = values()
    if (accessToken !== undefined) {
      headers.set('authorization', `Bearer ${accessToken}`)
    }
    return headers
  }
  return {
    async unary<I extends DescMessage, O extends DescMessage>(
      method: DescMethodUnary<I, O>,
      signal: AbortSignal | undefined,
      timeoutMs: number | undefined,
      header: HeadersInit | undefined,
      input: MessageInitShape<I>,
      contextValues?: ContextValues,
    ) {
      const response = await inner.unary(
        method,
        signal,
        timeoutMs,
        attach(header),
        input,
        contextValues,
      )
      checkServerVersion(response.header.get('x-moth-version'))
      return response
    },
    stream<I extends DescMessage, O extends DescMessage>(
      method: DescMethodStreaming<I, O>,
      signal: AbortSignal | undefined,
      timeoutMs: number | undefined,
      header: HeadersInit | undefined,
      input: AsyncIterable<MessageInitShape<I>>,
      contextValues?: ContextValues,
    ) {
      return inner.stream(
        method,
        signal,
        timeoutMs,
        attach(header),
        input,
        contextValues,
      )
    },
  }
}

let versionWarned = false

/**
 * Compares the server's `x-moth-version` response header against the SDK
 * version and warns once on a major-version mismatch (the SDK is served BY
 * the instance, so a skew means a stale lockfile / npm cache). Dev builds
 * (0.x, -dev suffixes) are exempt.
 */
function checkServerVersion(serverVersion: string | null): void {
  if (versionWarned || serverVersion === null || serverVersion === '') return
  const majorOf = (v: string) => {
    const major = parseInt(v.replace(/^v/, ''), 10)
    return Number.isNaN(major) ? null : major
  }
  const server = majorOf(serverVersion)
  const sdk = majorOf(mothSdkVersion)
  if (server === null || sdk === null || server === 0 || sdk === 0) return
  if (server !== sdk) {
    versionWarned = true
    console.warn(
      `moth: SDK version ${mothSdkVersion} does not match server version ` +
        `${serverVersion}; reinstall @moth/react from your instance's /npm ` +
        'registry so they stay in lockstep.',
    )
  }
}
