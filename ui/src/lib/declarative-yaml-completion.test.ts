import { describe, expect, it } from 'vitest'
import { declarativeYamlCompletions } from '@/lib/declarative-yaml-completion'

function complete(source: string, explicit = true) {
  return declarativeYamlCompletions(source, source.length, explicit)
}

function labels(source: string, explicit = true) {
  return complete(source, explicit)?.options.map((option) => option.label) ?? []
}

describe('declarative YAML completion', () => {
  it('suggests root configuration properties from the bundled OpenAPI document', () => {
    expect(labels('')).toEqual(expect.arrayContaining(['sources', 'endpoints']))
    expect(complete('sou', false)?.from).toBe(0)
  })

  it('resolves array items, references, and allOf schemas', () => {
    const sourceFields = labels('sources:\n  - ')

    expect(sourceFields).toEqual(
      expect.arrayContaining(['name', 'enabled', 'type', 'config', 'async', 'plugins']),
    )
    expect(sourceFields).not.toEqual(expect.arrayContaining(['id', 'created_at', 'updated_at']))
  })

  it('suggests nested object properties', () => {
    expect(labels('sources:\n  - config:\n      http:\n        ')).toEqual(
      expect.arrayContaining(['path', 'methods', 'response']),
    )
    expect(labels('endpoints:\n  - request:\n      ')).toEqual(
      expect.arrayContaining(['url', 'method', 'headers', 'timeout']),
    )
  })

  it('suggests enum, boolean, and flow-array values', () => {
    expect(labels('sources:\n  - enabled: ')).toEqual(['true', 'false'])
    expect(labels('sources:\n  - type: h', false)).toContain('http')
    expect(labels('endpoints:\n  - request:\n      method: P', false)).toEqual(
      expect.arrayContaining(['GET', 'POST', 'PUT', 'DELETE', 'PATCH']),
    )
    expect(labels('sources:\n  - config:\n      http:\n        methods: [P', false)).toEqual(
      expect.arrayContaining(['GET', 'POST', 'PUT', 'DELETE', 'PATCH']),
    )
  })

  it('builds useful insertion text from schema defaults and container types', () => {
    const root = complete('sou', false)
    const sources = root?.options.find((option) => option.label === 'sources')
    expect(sources?.apply).toBe('sources:\n  - ')

    const fields = complete('sources:\n  - en', false)
    const enabled = fields?.options.find((option) => option.label === 'enabled')
    expect(enabled?.apply).toBe('enabled: true')
  })
})
