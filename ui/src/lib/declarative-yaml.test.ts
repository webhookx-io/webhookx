import { describe, expect, it } from 'vitest'
import {
  analyzeDeclarativeYaml,
  DECLARATIVE_YAML_EXAMPLE,
  formatDeclarativeYaml,
  summarizeDeclarativeYaml,
} from '@/lib/declarative-yaml'

describe('declarative YAML analysis', () => {
  it('reports syntax errors at their source location', () => {
    const analysis = analyzeDeclarativeYaml(
      'sources:\n  - name: inbound\n   enabled: true\nendpoints: []',
    )

    expect(analysis.value).toBeNull()
    expect(analysis.error).toMatch(/^Line 3:/)
    expect(analysis.diagnostics).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ severity: 'error', source: 'YAML', from: 27 }),
      ]),
    )
  })

  it('reports missing workspace resource collections', () => {
    const analysis = analyzeDeclarativeYaml('sources: []')

    expect(analysis.value).toBeNull()
    expect(analysis.error).toBe('The YAML must contain both sources and endpoints arrays.')
    expect(analysis.diagnostics[0]).toMatchObject({ severity: 'error', source: 'WebhookX' })
  })

  it('summarizes and formats a valid configuration', () => {
    expect(summarizeDeclarativeYaml(DECLARATIVE_YAML_EXAMPLE)).toEqual({
      sources: 1,
      endpoints: 1,
    })

    const formatted = formatDeclarativeYaml('sources: []\nendpoints: []')
    expect(formatted).toBe('sources: []\nendpoints: []\n')
  })
})
