import '@testing-library/jest-dom/vitest'

const storageValues = new Map<string, string>()

Object.defineProperty(globalThis, 'localStorage', {
  configurable: true,
  value: {
    get length() {
      return storageValues.size
    },
    clear: () => storageValues.clear(),
    getItem: (key: string) => storageValues.get(key) ?? null,
    key: (index: number) => [...storageValues.keys()][index] ?? null,
    removeItem: (key: string) => storageValues.delete(key),
    setItem: (key: string, value: string) => storageValues.set(key, value),
  } satisfies Storage,
})
