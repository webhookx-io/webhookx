import type { JsonSchema, JsonSchemaType } from '@/types'

export type SchemaDefinitions = Record<string, JsonSchema>

function clone<T>(value: T): T {
  if (value === undefined) return value
  return JSON.parse(JSON.stringify(value)) as T
}

function referenceName(reference: string) {
  return decodeURIComponent(reference.split('/').at(-1) ?? reference)
}

function mergeSchemas(base: JsonSchema, overlay: JsonSchema): JsonSchema {
  return {
    ...base,
    ...overlay,
    properties:
      base.properties || overlay.properties
        ? { ...base.properties, ...overlay.properties }
        : undefined,
    required:
      base.required || overlay.required
        ? [...new Set([...(base.required ?? []), ...(overlay.required ?? [])])]
        : undefined,
  }
}

export function displayString(value: unknown) {
  if (value === undefined || value === null) return ''
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean' || typeof value === 'bigint') {
    return String(value)
  }
  if (typeof value === 'symbol') return value.description ?? ''
  try {
    return JSON.stringify(value) ?? ''
  } catch {
    return ''
  }
}

export function humanize(value: string) {
  return value
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .replace(/[-_]+/g, ' ')
    .replace(/^./, (character) => character.toUpperCase())
}

export function resolveSchema(
  schema: JsonSchema,
  definitions: SchemaDefinitions,
  resolving = new Set<string>(),
): JsonSchema {
  let resolved = schema

  if (schema.$ref) {
    const name = referenceName(schema.$ref)
    if (!resolving.has(name) && definitions[name]) {
      const nextResolving = new Set(resolving).add(name)
      const target = resolveSchema(definitions[name], definitions, nextResolving)
      const overlay = { ...schema }
      delete overlay.$ref
      resolved = mergeSchemas(target, overlay)
    }
  }

  if (resolved.allOf?.length) {
    const combined = resolved.allOf.reduce<JsonSchema>(
      (current, item) => mergeSchemas(current, resolveSchema(item, definitions, resolving)),
      {},
    )
    const overlay = { ...resolved }
    delete overlay.allOf
    resolved = mergeSchemas(combined, overlay)
  }

  return resolved
}

export function schemaType(schema: JsonSchema): JsonSchemaType | undefined {
  if (Array.isArray(schema.type)) {
    return schema.type.find((type) => type !== 'null') ?? schema.type[0]
  }
  if (schema.type) return schema.type
  if (schema.properties || schema.additionalProperties !== undefined) return 'object'
  return undefined
}

export function createSchemaDefaults(
  source: JsonSchema,
  definitions: SchemaDefinitions = {},
): unknown {
  const schema = resolveSchema(source, definitions)
  if (schema.default !== undefined) return clone(schema.default)
  if (schema.oneOf?.length || schema.anyOf?.length) return {}

  switch (schemaType(schema)) {
    case 'object': {
      const result: Record<string, unknown> = {}
      Object.entries(schema.properties ?? {}).forEach(([name, propertySchema]) => {
        const resolvedProperty = resolveSchema(propertySchema, definitions)
        const required = schema.required?.includes(name) ?? false
        const hasDefault = resolvedProperty.default !== undefined
        if (!required && !hasDefault) return
        const value = createSchemaDefaults(resolvedProperty, definitions)
        if (value !== undefined) result[name] = value
      })
      return result
    }
    case 'array':
      return []
    case 'boolean':
      return false
    case 'integer':
    case 'number':
      return schema.minimum ?? 0
    case 'string':
      return schema.enum?.[0] ?? ''
    case 'null':
      return null
    default:
      return undefined
  }
}

export function variantDetails(option: JsonSchema, index: number, schema: JsonSchema) {
  const mapping = schema.discriminator?.mapping ?? {}
  const mappingEntry = option.$ref
    ? Object.entries(mapping).find(([, reference]) => reference === option.$ref)
    : undefined
  const name =
    mappingEntry?.[0] ?? (option.$ref ? referenceName(option.$ref) : `option-${index + 1}`)
  return {
    value: name.replace(/ProviderConfig$/i, '').toLowerCase(),
    label: humanize(name.replace(/ProviderConfig$/i, '')),
  }
}

function missing(value: unknown) {
  return value === undefined || value === null || (typeof value === 'string' && !value.trim())
}

export function validateSchemaValue(
  source: JsonSchema,
  value: unknown,
  definitions: SchemaDefinitions = {},
  path = 'Configuration',
): string[] {
  const schema = resolveSchema(source, definitions)
  if (value === null && schema.nullable) return []

  const variants = schema.oneOf ?? schema.anyOf
  if (variants?.length) {
    const discriminator = schema.discriminator?.propertyName
    const discriminatorValue =
      discriminator && value && typeof value === 'object' && !Array.isArray(value)
        ? displayString((value as Record<string, unknown>)[discriminator])
        : ''
    const index = variants.findIndex(
      (option, optionIndex) =>
        variantDetails(option, optionIndex, schema).value === discriminatorValue,
    )
    if (index < 0) return [`${path}: select a configuration variant.`]
    return validateSchemaValue(variants[index], value, definitions, path)
  }

  if (schema.enum && !schema.enum.some((item) => item === value)) {
    return [`${path}: select one of the allowed values.`]
  }

  switch (schemaType(schema)) {
    case 'object': {
      if (!value || typeof value !== 'object' || Array.isArray(value)) {
        return [`${path}: enter an object.`]
      }
      const record = value as Record<string, unknown>
      const errors: string[] = []
      for (const name of schema.required ?? []) {
        if (missing(record[name])) errors.push(`${path}.${name}: this field is required.`)
      }
      Object.entries(schema.properties ?? {}).forEach(([name, propertySchema]) => {
        if (record[name] === undefined) return
        errors.push(
          ...validateSchemaValue(propertySchema, record[name], definitions, `${path}.${name}`),
        )
      })
      if (typeof schema.additionalProperties === 'object') {
        Object.entries(record).forEach(([name, item]) => {
          if (schema.properties?.[name]) return
          errors.push(
            ...validateSchemaValue(
              schema.additionalProperties as JsonSchema,
              item,
              definitions,
              `${path}.${name}`,
            ),
          )
        })
      }
      return errors
    }
    case 'array': {
      if (!Array.isArray(value)) return [`${path}: enter a list.`]
      const errors: string[] = []
      if (schema.minItems !== undefined && value.length < schema.minItems) {
        errors.push(`${path}: select at least ${schema.minItems} item(s).`)
      }
      if (schema.maxItems !== undefined && value.length > schema.maxItems) {
        errors.push(`${path}: select no more than ${schema.maxItems} item(s).`)
      }
      if (schema.items) {
        value.forEach((item, index) => {
          errors.push(...validateSchemaValue(schema.items!, item, definitions, `${path}[${index}]`))
        })
      }
      return errors
    }
    case 'string': {
      if (typeof value !== 'string') return [`${path}: enter text.`]
      const errors: string[] = []
      if (schema.minLength !== undefined && value.length < schema.minLength) {
        errors.push(`${path}: use at least ${schema.minLength} character(s).`)
      }
      if (schema.maxLength !== undefined && value.length > schema.maxLength) {
        errors.push(`${path}: use no more than ${schema.maxLength} character(s).`)
      }
      if (schema.format === 'json' && value) {
        try {
          JSON.parse(value)
        } catch {
          errors.push(`${path}: enter valid JSON.`)
        }
      }
      return errors
    }
    case 'integer':
    case 'number': {
      if (typeof value !== 'number' || Number.isNaN(value)) return [`${path}: enter a number.`]
      const errors: string[] = []
      if (schemaType(schema) === 'integer' && !Number.isInteger(value)) {
        errors.push(`${path}: enter a whole number.`)
      }
      if (schema.minimum !== undefined && value < schema.minimum) {
        errors.push(`${path}: the minimum is ${schema.minimum}.`)
      }
      if (schema.maximum !== undefined && value > schema.maximum) {
        errors.push(`${path}: the maximum is ${schema.maximum}.`)
      }
      return errors
    }
    case 'boolean':
      return typeof value === 'boolean' ? [] : [`${path}: choose on or off.`]
    default:
      return []
  }
}
