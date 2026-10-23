import { readFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import type { Plugin } from 'vite'
import { parse } from 'yaml'

export function openApiSpec(): Plugin {
  const sourcePath = fileURLToPath(new URL('../openapi.yml', import.meta.url))
  const moduleId = `${sourcePath}?json`

  return {
    name: 'webhookx-openapi-spec',
    enforce: 'pre',
    async load(id) {
      if (id !== moduleId) return null

      this.addWatchFile(sourcePath)
      const document = parse(await readFile(sourcePath, 'utf8')) as unknown
      if (!document || typeof document !== 'object') {
        this.error('The root openapi.yml must contain an object')
      }

      return `export default ${JSON.stringify(document)};`
    },
  }
}
