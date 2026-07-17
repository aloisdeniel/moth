import '@testing-library/jest-dom/vitest'
import { afterEach } from 'vitest'
import { cleanup } from '@testing-library/react'

// Node >= 22 exposes an experimental (non-functional without a flag)
// `localStorage` global that shadows jsdom's when vitest merges globals.
// Install a deterministic in-memory Web Storage for tests instead.
class TestStorage implements Storage {
  #map = new Map<string, string>()

  get length(): number {
    return this.#map.size
  }

  clear(): void {
    this.#map.clear()
  }

  getItem(key: string): string | null {
    return this.#map.get(key) ?? null
  }

  key(index: number): string | null {
    return [...this.#map.keys()][index] ?? null
  }

  removeItem(key: string): void {
    this.#map.delete(key)
  }

  setItem(key: string, value: string): void {
    this.#map.set(key, String(value))
  }
}

for (const name of ['localStorage', 'sessionStorage'] as const) {
  Object.defineProperty(globalThis, name, {
    value: new TestStorage(),
    configurable: true,
    writable: true,
  })
}

afterEach(() => {
  cleanup()
  window.localStorage.clear()
  window.sessionStorage.clear()
})
