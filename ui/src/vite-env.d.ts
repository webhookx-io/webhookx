/// <reference types="vite/client" />

declare module '*.yml?json' {
  const document: unknown
  export default document
}

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL?: string
  readonly VITE_GATEWAY_BASE_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
