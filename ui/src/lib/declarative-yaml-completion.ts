import type {
  Completion,
  CompletionContext,
  CompletionResult,
  CompletionSource,
} from '@codemirror/autocomplete'
import { openApiSchemaDefinitions } from '@/data/openapi'
import { displayString, resolveSchema, schemaType } from '@/lib/json-schema'
import type { JsonSchema } from '@/types'

interface SchemaFrame {
  indent: number
  schema: JsonSchema
}

interface ParsedLine {
  rawIndent: number
  keyIndent: number
  sequenceItem: boolean
  key: string | null
  value: string | null
}

const configurationSchema = openApiSchemaDefinitions.Configuration
const propertyPattern = /^\s*(-\s*)?([A-Za-z_][\w-]*)\s*:\s*(.*)$/

function resolved(schema: JsonSchema | undefined) {
  return schema ? resolveSchema(schema, openApiSchemaDefinitions) : undefined
}

function parseLine(line: string): ParsedLine {
  const rawIndent = line.match(/^\s*/)?.[0].length ?? 0
  const content = line.slice(rawIndent)
  const sequenceMatch = content.match(/^-\s*(.*)$/)
  const sequenceItem = Boolean(sequenceMatch)
  const keyContent = sequenceMatch?.[1] ?? content
  const keyMatch = keyContent.match(/^([A-Za-z_][\w-]*)\s*:\s*(.*)$/)

  return {
    rawIndent,
    keyIndent: rawIndent + (sequenceItem ? 2 : 0),
    sequenceItem,
    key: keyMatch?.[1] ?? null,
    value: keyMatch?.[2] ?? null,
  }
}

function itemSchema(schema: JsonSchema | undefined) {
  const source = resolved(schema)
  return source && schemaType(source) === 'array' ? resolved(source.items) : source
}

function mappingSchema(schema: JsonSchema | undefined) {
  const source = itemSchema(schema)
  return source && schemaType(source) === 'object' ? source : undefined
}

function popForLine(frames: SchemaFrame[], line: ParsedLine) {
  while (
    frames.length > 1 &&
    (line.sequenceItem
      ? frames[frames.length - 1].indent >= line.keyIndent
      : frames[frames.length - 1].indent > line.keyIndent)
  ) {
    frames.pop()
  }
}

function prepareLine(frames: SchemaFrame[], line: ParsedLine) {
  popForLine(frames, line)
  let parent = mappingSchema(frames[frames.length - 1]?.schema)

  if (line.sequenceItem) {
    const sequenceItem = itemSchema(frames[frames.length - 1]?.schema)
    if (sequenceItem && schemaType(sequenceItem) === 'object') {
      frames.push({ indent: line.keyIndent, schema: sequenceItem })
      parent = sequenceItem
    }
  }

  return parent
}

function scanFrames(lines: string[]) {
  const frames: SchemaFrame[] = [{ indent: 0, schema: configurationSchema ?? {} }]

  for (const source of lines) {
    if (!source.trim() || source.trimStart().startsWith('#')) continue
    const line = parseLine(source)
    const parent = prepareLine(frames, line)
    if (!parent || !line.key || line.value === null) continue

    const property = resolved(parent.properties?.[line.key])
    if (!property || line.value.trim()) continue
    const type = schemaType(property)
    if (type === 'object' || type === 'array') {
      frames.push({ indent: line.keyIndent + 2, schema: property })
    }
  }

  return frames
}

function yamlScalar(value: unknown) {
  if (value === null) return 'null'
  if (typeof value !== 'string') return displayString(value)
  if (/^[A-Za-z_][\w./:-]*$/.test(value) && !/^(true|false|null|yes|no|on|off)$/i.test(value)) {
    return value
  }
  return JSON.stringify(value)
}

function schemaDetail(schema: JsonSchema, required = false) {
  const type = schemaType(schema) ?? 'value'
  return required ? `${type} · required` : type
}

function schemaInfo(schema: JsonSchema) {
  const details = [schema.description]
  if (schema.default !== undefined) details.push(`Default: ${yamlScalar(schema.default)}`)
  if (schema.minimum !== undefined) details.push(`Minimum: ${schema.minimum}`)
  if (schema.maximum !== undefined) details.push(`Maximum: ${schema.maximum}`)
  return details.filter(Boolean).join('\n') || undefined
}

function propertyInsert(name: string, schema: JsonSchema, indent: number) {
  const type = schemaType(schema)
  if (schema.default !== undefined && type !== 'object') {
    return `${name}: ${yamlScalar(schema.default)}`
  }
  if (type === 'object') return `${name}:\n${' '.repeat(indent + 2)}`
  if (type === 'array') return `${name}:\n${' '.repeat(indent + 2)}- `
  return `${name}: `
}

function propertyCompletions(schema: JsonSchema, indent: number): Completion[] {
  const required = new Set(schema.required ?? [])
  return Object.entries(schema.properties ?? {}).flatMap(([name, source]) => {
    const property = resolved(source)
    if (!property || property.readOnly) return []
    return [
      {
        label: name,
        apply: propertyInsert(name, property, indent),
        type: 'property',
        detail: schemaDetail(property, required.has(name)),
        info: schemaInfo(property),
        boost: required.has(name) ? 10 : 0,
      },
    ]
  })
}

function valueCandidates(source: JsonSchema): Array<{ value: unknown; detail?: string }> {
  const schema = resolved(source) ?? source
  const candidates: Array<{ value: unknown; detail?: string }> = []
  const values = schema.enum ?? []

  values.forEach((value) => candidates.push({ value, detail: 'allowed value' }))
  if (schemaType(schema) === 'boolean' && values.length === 0) {
    candidates.push({ value: true, detail: 'boolean' }, { value: false, detail: 'boolean' })
  }
  if (
    schema.default !== undefined &&
    !candidates.some((candidate) => candidate.value === schema.default)
  ) {
    candidates.push({ value: schema.default, detail: 'default' })
  }
  if (
    schema.example !== undefined &&
    !candidates.some((candidate) => candidate.value === schema.example)
  ) {
    candidates.push({ value: schema.example, detail: 'example' })
  }
  if (schema.nullable && !candidates.some((candidate) => candidate.value === null)) {
    candidates.push({ value: null, detail: 'nullable' })
  }

  return candidates
}

function valueCompletions(schema: JsonSchema): Completion[] {
  return valueCandidates(schema).map(({ value, detail }) => ({
    label: yamlScalar(value),
    apply: yamlScalar(value),
    type: schema.enum ? 'enum' : 'constant',
    detail,
    info: schemaInfo(schema),
  }))
}

function currentParent(frames: SchemaFrame[], line: ParsedLine) {
  const working = [...frames]
  return prepareLine(working, line)
}

export function declarativeYamlCompletions(
  text: string,
  position: number,
  explicit = false,
): CompletionResult | null {
  if (!configurationSchema) return null

  const before = text.slice(0, position)
  const lines = before.split('\n')
  const currentSource = lines.pop() ?? ''
  const lineStart = position - currentSource.length
  const frames = scanFrames(lines)
  const current = parseLine(currentSource)
  const propertyLine = currentSource.match(propertyPattern)

  if (propertyLine && current.key && current.value !== null) {
    const parent = currentParent(frames, current)
    let property = resolved(parent?.properties?.[current.key])
    if (!property) return null

    const valueOffset = currentSource.length - current.value.length
    const flowStart = Math.max(current.value.lastIndexOf('['), current.value.lastIndexOf(','))
    const flowArray =
      schemaType(property) === 'array' &&
      flowStart >= 0 &&
      !current.value.slice(flowStart).includes(']')
    if (flowArray) property = resolved(property.items)
    if (!property) return null

    let tokenStart = current.value.length
    while (tokenStart > 0 && ![' ', '\t', ',', '[', ']'].includes(current.value[tokenStart - 1])) {
      tokenStart -= 1
    }
    const token = current.value.slice(tokenStart)
    const options = valueCompletions(property)
    if (options.length === 0) return null

    return {
      from: lineStart + valueOffset + current.value.length - token.length,
      options,
      validFor: /[\w./:"'*-]*/,
    }
  }

  const parent = currentParent(frames, current)
  if (!parent) return null
  const prefixMatch = currentSource.match(/(?:^\s*(?:-\s*)?)([A-Za-z_][\w-]*)?$/)
  if (!prefixMatch) return null
  const prefix = prefixMatch[1] ?? ''
  if (!explicit && !prefix) return null

  return {
    from: position - prefix.length,
    options: propertyCompletions(parent, current.keyIndent),
    validFor: /[\w-]*/,
  }
}

export const declarativeYamlCompletionSource: CompletionSource = (context: CompletionContext) =>
  declarativeYamlCompletions(context.state.doc.toString(), context.pos, context.explicit)
