import { describe, expect, it } from 'vitest'
import {
  createSchemaDefaults,
  displayString,
  resolveSchema,
  validateSchemaValue,
  type SchemaDefinitions,
} from '@/lib/json-schema'

const definitions: SchemaDefinitions = {
  Credentials: {
    type: 'object',
    required: ['token'],
    properties: {
      token: { type: 'string', minLength: 3 },
      timeout: { type: 'integer', default: 30, minimum: 1 },
    },
  },
}

describe('JSON schema helpers', () => {
  it('resolves references and allOf overlays without mutating definitions', () => {
    const resolved = resolveSchema(
      {
        allOf: [
          { $ref: '#/components/schemas/Credentials' },
          { properties: { region: { type: 'string' } } },
        ],
      },
      definitions,
    )
    expect(resolved.required).toEqual(['token'])
    expect(Object.keys(resolved.properties ?? {})).toEqual(['token', 'timeout', 'region'])
    expect(definitions.Credentials.properties?.region).toBeUndefined()
  })

  it('creates defaults only for required fields and explicit defaults', () => {
    expect(createSchemaDefaults({ $ref: '#/components/schemas/Credentials' }, definitions)).toEqual(
      {
        token: '',
        timeout: 30,
      },
    )
  })

  it('validates nested required, length, and numeric constraints', () => {
    expect(
      validateSchemaValue(
        { $ref: '#/components/schemas/Credentials' },
        { token: 'x', timeout: 0 },
        definitions,
      ),
    ).toEqual([
      'Configuration.token: use at least 3 character(s).',
      'Configuration.timeout: the minimum is 1.',
    ])
  })

  it('serializes unexpected object values without leaking [object Object] into fields', () => {
    expect(displayString({ provider: 'vault' })).toBe('{"provider":"vault"}')
  })
})
