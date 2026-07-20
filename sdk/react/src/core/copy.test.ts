import { describe, expect, it } from 'vitest'
import { MothCopy, languageOf } from './copy.js'
import { bundledCopy, mothBundledLocales } from './i18n/bundledCopy.js'

describe('MothCopy resolution', () => {
  it('server override wins over the bundled catalog', () => {
    const copy = new MothCopy('en', 'rev', {
      'sign_in.title': 'Enter the lair',
    })
    expect(copy.value('sign_in.title')).toBe('Enter the lair')
  })

  it('falls back to the bundled locale, then English, then the key', () => {
    const fr = new MothCopy('fr', 'rev', {})
    expect(fr.value('sign_in.title')).toBe('Connexion')
    // A bundled locale mirrors the full English key set.
    expect(fr.value('paywall.most_popular')).toBe('Le plus populaire')
    // Unbundled locale: English.
    const xx = new MothCopy('xx')
    expect(xx.value('sign_in.title')).toBe('Sign in')
    // Unknown key: the key itself.
    expect(xx.value('does.not.exist')).toBe('does.not.exist')
  })

  it('region tags resolve through their language', () => {
    const frCa = MothCopy.bundled('fr-CA')
    expect(frCa.value('sign_in.title')).toBe('Connexion')
    expect(languageOf('fr-CA')).toBe('fr')
    expect(languageOf('zh_Hant_TW')).toBe('zh')
  })

  it('substitutes {name} placeholders literally, leaving unknown ones', () => {
    const copy = new MothCopy('en', 'rev', {
      greeting: 'Hello {name}, welcome to {app} {missing}',
    })
    expect(copy.value('greeting', { name: 'Ada', app: 'TestApp' })).toBe(
      'Hello Ada, welcome to TestApp {missing}',
    )
  })

  it('fills {app} in the bundled floor (the server interpolates its own)', () => {
    const copy = MothCopy.bundled('en')
    expect(copy.value('sign_in.subtitle', { app: 'TestApp' })).toBe(
      'Welcome back to TestApp.',
    )
    expect(
      copy.value('sign_up.password_too_short', { count: '8' }),
    ).toBe('Use at least 8 characters')
  })
})

describe('bundled catalog', () => {
  it('bundles the documented locales, English first', () => {
    expect(mothBundledLocales[0]).toBe('en')
    expect([...mothBundledLocales]).toEqual(
      expect.arrayContaining(['en', 'fr', 'de', 'es', 'pt', 'it', 'ja']),
    )
  })

  it('every locale resolves every English key non-empty', () => {
    const en = bundledCopy('en')
    const keys = Object.keys(en)
    expect(keys.length).toBeGreaterThan(70)
    for (const locale of mothBundledLocales) {
      const catalog = bundledCopy(locale)
      for (const key of keys) {
        expect(catalog[key], `${locale}:${key}`).toBeTruthy()
      }
    }
  })

  it('includes the web-only unavailable-on-web key', () => {
    expect(bundledCopy('en')['paywall.unavailable_web']).toContain('web')
    expect(bundledCopy('ja')['paywall.unavailable_web']).toBe(
      'ウェブでは購入できません',
    )
  })
})
