import { create, fromBinary, toBinary } from '@bufbuild/protobuf'
import { describe, expect, it } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import { CacheEnvelopeSchema } from '../gen/moth/projectconfig/v1/projectconfig_pb.js'
import { CopySchema, ThemeSchema } from '../gen/moth/auth/v1/config_pb.js'
import { fakeClient } from '../test/fake.js'
import {
  base64Decode,
  base64Encode,
  BlobCache,
  cacheNamespace,
  MemoryStorage,
} from './cache.js'
import { MothConfigController } from './configController.js'
import { loadPaywall } from './paywallLoader.js'
import { PaywallSchema } from '../gen/moth/billing/v1/billing_pb.js'
import { sha256Hex } from './sha256.js'

describe('sha256 / namespace', () => {
  it('matches known SHA-256 vectors', () => {
    expect(sha256Hex('')).toBe(
      'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
    )
    expect(sha256Hex('abc')).toBe(
      'ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad',
    )
  })

  it('namespaces caches by the first 16 hex chars of sha256(pk)', () => {
    expect(cacheNamespace('pk_test')).toBe(sha256Hex('pk_test').slice(0, 16))
    expect(cacheNamespace('pk_test')).toHaveLength(16)
  })
})

describe('BlobCache envelope', () => {
  it('round-trips a protobuf payload through a CacheEnvelope (never JSON)', () => {
    const storage = new MemoryStorage()
    const cache = new BlobCache(storage, 'slot')
    const theme = create(ThemeSchema, { revisionId: 'rev-1', fontFamily: 'Inter' })
    const payload = toBinary(ThemeSchema, theme)
    cache.save({ payload, revision: 'rev-1', locale: '', fetchedAtMs: 12345 })

    // The stored value is a base64 CacheEnvelope wire message.
    const stored = storage.getItem('slot')!
    const envelope = fromBinary(CacheEnvelopeSchema, base64Decode(stored))
    expect(envelope.revision).toBe('rev-1')
    expect(Number(envelope.fetchedAtUnixMs)).toBe(12345)
    expect(envelope.payload).toEqual(payload)
    expect(() => JSON.parse(stored)).toThrow() // definitely not JSON

    const loaded = cache.load()!
    expect(loaded.revision).toBe('rev-1')
    expect(loaded.fetchedAtMs).toBe(12345)
    expect(fromBinary(ThemeSchema, loaded.payload).fontFamily).toBe('Inter')
  })

  it('treats corrupt entries as a miss and drops them', () => {
    const storage = new MemoryStorage()
    storage.setItem('slot', 'not base64 protobuf!!')
    const cache = new BlobCache(storage, 'slot')
    expect(cache.load()).toBeNull()
    expect(storage.getItem('slot')).toBeNull()
  })

  it('touch re-stamps the fetch time and keeps the payload', () => {
    const storage = new MemoryStorage()
    const cache = new BlobCache(storage, 'slot')
    cache.save({
      payload: new Uint8Array([1, 2, 3]),
      revision: 'r',
      locale: 'fr',
      fetchedAtMs: 1,
    })
    cache.touch(999)
    const blob = cache.load()!
    expect(blob.fetchedAtMs).toBe(999)
    expect(blob.revision).toBe('r')
    expect(blob.locale).toBe('fr')
    expect([...blob.payload]).toEqual([1, 2, 3])
  })

  it('swallows storage failures', () => {
    const throwing = {
      getItem: () => {
        throw new Error('nope')
      },
      setItem: () => {
        throw new Error('nope')
      },
      removeItem: () => {
        throw new Error('nope')
      },
    }
    const cache = new BlobCache(throwing, 'slot')
    expect(cache.load()).toBeNull()
    expect(() =>
      cache.save({ payload: new Uint8Array([1]), revision: '', locale: '', fetchedAtMs: 0 }),
    ).not.toThrow()
  })

  it('base64 helpers round-trip binary data', () => {
    const bytes = new Uint8Array(300).map((_, i) => i % 256)
    expect(base64Decode(base64Encode(bytes))).toEqual(bytes)
  })
})

describe('config controller download-once TTL', () => {
  function copyMessage() {
    return create(CopySchema, {
      copyRevision: 'copy-1',
      locale: 'en',
      messages: { 'sign_in.title': 'Enter the app' },
    })
  }

  it('first start fetches, saves, and a fresh restart performs zero RPCs', async () => {
    const storage = new MemoryStorage()
    const { client, fake } = fakeClient()
    fake.state.projectConfig.theme = create(ThemeSchema, { revisionId: 'theme-1' })
    Object.assign(fake.state.projectConfig, { copy: copyMessage() })

    const controller = new MothConfigController(client, { storage })
    await controller.start()
    expect(fake.calls['getProjectConfig']).toBe(1)
    expect(controller.theme.revisionId).toBe('theme-1')
    expect(controller.copy.revisionId).toBe('copy-1')
    expect(controller.copy.value('sign_in.title')).toBe('Enter the app')

    // Second launch within the TTL: cached blobs, zero config RPCs.
    const second = new MothConfigController(client, { storage })
    await second.start()
    expect(fake.calls['getProjectConfig']).toBe(1)
    expect(second.theme.revisionId).toBe('theme-1')
    expect(second.copy.value('sign_in.title')).toBe('Enter the app')
  })

  it('a stale cache revalidates echoing revisions; a match touches, not saves', async () => {
    const storage = new MemoryStorage()
    let nowMs = 1_000_000
    const now = () => nowMs
    const { client, fake } = fakeClient()
    fake.state.projectConfig.theme = create(ThemeSchema, { revisionId: 'theme-1' })
    Object.assign(fake.state.projectConfig, { copy: copyMessage() })

    const first = new MothConfigController(client, { storage, now })
    await first.start()
    expect(fake.calls['getProjectConfig']).toBe(1)

    // Past the TTL: the next start revalidates, echoing the cached
    // revisions; the fake omits matched bodies (theme dropped, copy
    // messages emptied), so the controller touches instead of saving.
    nowMs += 2 * 60 * 60 * 1000
    const second = new MothConfigController(client, { storage, now })
    await second.start()
    expect(fake.calls['getProjectConfig']).toBe(2)
    expect(second.theme.revisionId).toBe('theme-1') // kept from cache
    expect(second.copy.value('sign_in.title')).toBe('Enter the app')

    // The touch restarted the window: a third start is silent again.
    nowMs += 30 * 60 * 1000
    const third = new MothConfigController(client, { storage, now })
    await third.start()
    expect(fake.calls['getProjectConfig']).toBe(2)
  })

  it('ensureProjectConfig single-flights concurrent callers', async () => {
    // React StrictMode mounts the login screen's effect twice, issuing two
    // concurrent ensureProjectConfig calls. They must share one fetch: two
    // independent fetches race the generation guard — the superseded one
    // never records the config and wrongly rejected with "project config
    // unavailable" even though its round-trip succeeded.
    const storage = new MemoryStorage()
    const { client, fake } = fakeClient()
    const controller = new MothConfigController(client, { storage })
    const [first, second] = await Promise.all([
      controller.ensureProjectConfig(),
      controller.ensureProjectConfig(),
    ])
    expect(fake.calls['getProjectConfig']).toBe(1)
    expect(first.signUpOpen).toBe(true)
    expect(second).toBe(first)

    // A later call reuses the recorded config without another round-trip.
    await controller.ensureProjectConfig()
    expect(fake.calls['getProjectConfig']).toBe(1)
  })

  it('a new revision replaces the cached body', async () => {
    const storage = new MemoryStorage()
    let nowMs = 1_000_000
    const now = () => nowMs
    const { client, fake } = fakeClient()
    fake.state.projectConfig.theme = create(ThemeSchema, { revisionId: 'theme-1' })
    Object.assign(fake.state.projectConfig, { copy: copyMessage() })
    const first = new MothConfigController(client, { storage, now })
    await first.start()

    // Admin edits the theme and the copy.
    fake.state.projectConfig.theme = create(ThemeSchema, {
      revisionId: 'theme-2',
      fontFamily: 'Sora',
    })
    Object.assign(fake.state.projectConfig, {
      copy: create(CopySchema, {
        copyRevision: 'copy-2',
        locale: 'en',
        messages: { 'sign_in.title': 'New title' },
      }),
    })
    nowMs += 2 * 60 * 60 * 1000
    const second = new MothConfigController(client, { storage, now })
    await second.start()
    expect(second.theme.revisionId).toBe('theme-2')
    expect(second.copy.revisionId).toBe('copy-2')
    expect(second.copy.value('sign_in.title')).toBe('New title')

    // And the new bodies are what a silent third launch serves.
    const third = new MothConfigController(client, { storage, now })
    await third.start()
    expect(fake.calls['getProjectConfig']).toBe(2)
    expect(third.theme.fontFamily).toBe('Sora')
    expect(third.copy.value('sign_in.title')).toBe('New title')
  })

  it('refresh always hits the server, even within the TTL', async () => {
    const storage = new MemoryStorage()
    const { client, fake } = fakeClient()
    fake.state.projectConfig.theme = create(ThemeSchema, { revisionId: 'theme-1' })
    const controller = new MothConfigController(client, { storage })
    await controller.start()
    await controller.refresh()
    expect(fake.calls['getProjectConfig']).toBe(2)
  })
})

describe('paywall loader', () => {
  function paywalledFake() {
    const { client, fake } = fakeClient()
    fake.state.paywall = create(PaywallSchema, {
      revisionId: 'pw-1',
      headline: 'Go pro',
      offering: 'main',
    })
    return { client, fake }
  }

  it('caches by revision with download-once TTL', async () => {
    const storage = new MemoryStorage()
    let nowMs = 1_000_000
    const now = () => nowMs
    const { client, fake } = paywalledFake()

    const first = await loadPaywall(client, { storage, now })
    expect(first.headline).toBe('Go pro')
    expect(fake.calls['getPaywall']).toBe(1)

    // Fresh: zero RPCs.
    const second = await loadPaywall(client, { storage, now })
    expect(second.headline).toBe('Go pro')
    expect(fake.calls['getPaywall']).toBe(1)

    // Stale: revalidates; the fake omits the matched body → touch.
    nowMs += 2 * 60 * 60 * 1000
    const third = await loadPaywall(client, { storage, now })
    expect(third.headline).toBe('Go pro')
    expect(fake.calls['getPaywall']).toBe(2)

    // Touched: silent again.
    const fourth = await loadPaywall(client, { storage, now })
    expect(fourth.revisionId).toBe('pw-1')
    expect(fake.calls['getPaywall']).toBe(2)
  })

  it('falls back to the stale cache when the network fails', async () => {
    const storage = new MemoryStorage()
    let nowMs = 1_000_000
    const now = () => nowMs
    const { client, fake } = paywalledFake()
    await loadPaywall(client, { storage, now })
    nowMs += 2 * 60 * 60 * 1000
    fake.failNext['getPaywall'] = [new ConnectError('down', Code.Unavailable)]
    const stale = await loadPaywall(client, { storage, now })
    expect(stale.headline).toBe('Go pro')
  })
})
