import { Textarea } from '@/components/ui/textarea'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import type { JsonSchema } from '@/types'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { NativeSelect } from '@/components/ui/native-select'
import { Switch } from '@/components/ui/switch'
import {
  createSchemaDefaults,
  displayString,
  humanize,
  resolveSchema,
  schemaType,
  variantDetails,
  type SchemaDefinitions,
} from '@/lib/json-schema'

interface JsonSchemaFormProps {
  schema: JsonSchema
  definitions?: SchemaDefinitions
  value: Record<string, unknown>
  onChange: (value: Record<string, unknown>) => void
}

interface SchemaFieldProps {
  schema: JsonSchema
  definitions: SchemaDefinitions
  value: unknown
  onChange: (value: unknown) => void
  label?: string
  required?: boolean
  path: string
  depth?: number
}

function fieldLabel(label: string | undefined, required: boolean | undefined) {
  if (!label) return null
  return (
    <label className="label">
      {humanize(label)}
      {required && <span className="ml-1 text-red-500">*</span>}
    </label>
  )
}

function helpText(schema: JsonSchema) {
  if (!schema.description) return null
  return <p className="mt-1 text-[11px] leading-4 text-muted-foreground">{schema.description}</p>
}

function OneOfField({
  schema,
  definitions,
  value,
  onChange,
  label,
  required,
  path,
  depth = 0,
}: SchemaFieldProps) {
  const options = schema.oneOf ?? schema.anyOf ?? []
  const discriminator = schema.discriminator?.propertyName
  const objectValue =
    value && typeof value === 'object' && !Array.isArray(value)
      ? (value as Record<string, unknown>)
      : {}
  const selectedValue = discriminator ? displayString(objectValue[discriminator]) : ''
  const selectedIndex = options.findIndex(
    (option, index) => variantDetails(option, index, schema).value === selectedValue,
  )
  const selectedSchema =
    selectedIndex >= 0 ? resolveSchema(options[selectedIndex], definitions) : undefined
  const visibleSchema =
    selectedSchema && discriminator
      ? {
          ...selectedSchema,
          properties: Object.fromEntries(
            Object.entries(selectedSchema.properties ?? {}).filter(
              ([name]) => name !== discriminator,
            ),
          ),
          required: selectedSchema.required?.filter((name) => name !== discriminator),
        }
      : selectedSchema

  return (
    <div>
      {fieldLabel(label ?? 'Configuration variant', required)}
      <NativeSelect
        className="w-full"
        value={selectedValue}
        onChange={(event) => {
          const index = options.findIndex(
            (option, optionIndex) =>
              variantDetails(option, optionIndex, schema).value === event.target.value,
          )
          if (index < 0) {
            onChange({})
            return
          }
          const nextSchema = resolveSchema(options[index], definitions)
          const defaults = createSchemaDefaults(nextSchema, definitions)
          const nextValue =
            defaults && typeof defaults === 'object' && !Array.isArray(defaults)
              ? { ...(defaults as Record<string, unknown>) }
              : {}
          if (discriminator) nextValue[discriminator] = event.target.value
          onChange(nextValue)
        }}
      >
        <option value="">Select a provider…</option>
        {options.map((option, index) => {
          const details = variantDetails(option, index, schema)
          return (
            <option key={`${details.value}-${index}`} value={details.value}>
              {details.label}
            </option>
          )
        })}
      </NativeSelect>
      {helpText(schema)}
      {visibleSchema && (
        <div className="mt-4 rounded-lg border border-border bg-muted/15 p-4">
          <SchemaField
            schema={visibleSchema}
            definitions={definitions}
            value={objectValue}
            onChange={(next) => {
              const nextObject =
                next && typeof next === 'object' && !Array.isArray(next)
                  ? (next as Record<string, unknown>)
                  : {}
              onChange({
                ...nextObject,
                ...(discriminator ? { [discriminator]: selectedValue } : {}),
              })
            }}
            path={`${path}.variant`}
            depth={depth + 1}
          />
        </div>
      )}
    </div>
  )
}

function StringField({ schema, value, onChange, label, required, path }: SchemaFieldProps) {
  const stringValue = displayString(value)
  const inputId = path.replace(/[^a-z0-9_-]/gi, '-')
  const secret = /password|secret|(^|_)key$/i.test(label ?? '')
  const multiline =
    schema.format === 'json' || (schema.maxLength ?? 0) > 500 || label === 'function'

  return (
    <div>
      {fieldLabel(label, required)}
      {schema.enum ? (
        <NativeSelect
          id={inputId}
          className="w-full"
          value={stringValue}
          onChange={(event) => {
            const selected = schema.enum?.find((item) => String(item) === event.target.value)
            onChange(selected)
          }}
        >
          <option value="">Select a value…</option>
          {schema.enum.map((option) => (
            <option key={String(option)} value={String(option)}>
              {String(option)}
            </option>
          ))}
        </NativeSelect>
      ) : multiline ? (
        <Textarea
          id={inputId}
          rows={label === 'function' ? 8 : 4}
          className="font-mono text-xs"
          value={stringValue}
          placeholder={schema.format === 'json' ? '{"type":"object"}' : undefined}
          onChange={(event) => onChange(event.target.value || (required ? '' : undefined))}
        />
      ) : (
        <Input
          id={inputId}
          type={secret ? 'password' : 'text'}
          className={schema.format ? 'font-mono text-xs' : undefined}
          minLength={schema.minLength}
          maxLength={schema.maxLength}
          value={stringValue}
          onChange={(event) => onChange(event.target.value || (required ? '' : undefined))}
        />
      )}
      {helpText(schema)}
    </div>
  )
}

function NumberField({ schema, value, onChange, label, required, path }: SchemaFieldProps) {
  const inputId = path.replace(/[^a-z0-9_-]/gi, '-')
  return (
    <div>
      {fieldLabel(label, required)}
      <Input
        id={inputId}
        type="number"
        step={schemaType(schema) === 'integer' ? 1 : 'any'}
        min={schema.minimum}
        max={schema.maximum}
        value={typeof value === 'number' ? value : ''}
        onChange={(event) =>
          onChange(event.target.value === '' ? undefined : Number(event.target.value))
        }
      />
      {helpText(schema)}
    </div>
  )
}

function BooleanField({ schema, value, onChange, label, path }: SchemaFieldProps) {
  const checked = value === true
  return (
    <div className="flex items-center justify-between gap-4 rounded-lg border border-border p-3">
      <div>
        <label className="text-xs font-medium" htmlFor={path}>
          {humanize(label ?? path)}
        </label>
        {helpText(schema)}
      </div>
      <Switch
        id={path}
        checked={checked}
        onCheckedChange={onChange}
        aria-label={humanize(label ?? path)}
      />
    </div>
  )
}

function ArrayField({
  schema,
  definitions,
  value,
  onChange,
  label,
  required,
  path,
}: SchemaFieldProps) {
  const values: unknown[] = Array.isArray(value) ? value : []
  const itemSchema = resolveSchema(schema.items ?? {}, definitions)

  if (itemSchema.enum) {
    return (
      <fieldset>
        <legend className="label">
          {humanize(label ?? path)}
          {required && <span className="ml-1 text-red-500">*</span>}
        </legend>
        <div className="flex flex-wrap gap-2">
          {itemSchema.enum.map((option) => {
            const selected = values.some((item) => item === option)
            return (
              <Button
                key={String(option)}
                type="button"
                aria-pressed={selected}
                size="sm"
                variant={selected ? 'secondary' : 'outline'}
                onClick={() =>
                  onChange(
                    selected ? values.filter((item) => item !== option) : [...values, option],
                  )
                }
                className="font-mono text-[11px]"
              >
                {String(option)}
              </Button>
            )
          })}
        </div>
        {helpText(schema)}
      </fieldset>
    )
  }

  if (['string', 'number', 'integer'].includes(schemaType(itemSchema) ?? '')) {
    return (
      <div>
        {fieldLabel(label, required)}
        <Input
          id={path}
          className="font-mono text-xs"
          value={values.join(', ')}
          placeholder="value-one, value-two"
          onChange={(event) => {
            const next = event.target.value
              .split(',')
              .map((item) => item.trim())
              .filter(Boolean)
              .map((item) =>
                ['number', 'integer'].includes(schemaType(itemSchema) ?? '') ? Number(item) : item,
              )
            onChange(next)
          }}
        />
        <p className="mt-1 text-[11px] text-muted-foreground">
          {schema.description ?? 'Enter comma-separated values.'}
        </p>
      </div>
    )
  }

  return (
    <JsonValueField label={label} schema={schema} value={values} onChange={onChange} path={path} />
  )
}

function AdditionalPropertiesField({
  schema,
  definitions,
  value,
  onChange,
  label,
  required,
  path,
}: SchemaFieldProps) {
  const record =
    value && typeof value === 'object' && !Array.isArray(value)
      ? (value as Record<string, unknown>)
      : {}
  const [entries, setEntries] = useState<Array<{ id: string; key: string; value: unknown }>>(() =>
    Object.entries(record).map(([key, item]) => ({
      id: crypto.randomUUID(),
      key,
      value: item,
    })),
  )
  const recordSignature = JSON.stringify(record)
  const lastRecordSignatureRef = useRef(recordSignature)
  const valueSchema = resolveSchema(
    typeof schema.additionalProperties === 'object' ? schema.additionalProperties : {},
    definitions,
  )

  useEffect(() => {
    if (lastRecordSignatureRef.current === recordSignature) return
    lastRecordSignatureRef.current = recordSignature
    const nextRecord = JSON.parse(recordSignature) as Record<string, unknown>
    setEntries(
      Object.entries(nextRecord).map(([key, item]) => ({
        id: crypto.randomUUID(),
        key,
        value: item,
      })),
    )
  }, [recordSignature])

  const updateEntries = (nextEntries: Array<{ id: string; key: string; value: unknown }>) => {
    const nextRecord = Object.fromEntries(
      nextEntries
        .filter((entry) => entry.key.trim())
        .map((entry) => [entry.key.trim(), entry.value]),
    )
    lastRecordSignatureRef.current = JSON.stringify(nextRecord)
    setEntries(nextEntries)
    onChange(nextRecord)
  }

  return (
    <fieldset className="rounded-lg border border-border">
      <legend className="sr-only">{humanize(label ?? path)}</legend>
      <div className="flex items-start justify-between gap-3 border-b border-border px-4 py-3">
        <div>
          <p className="text-xs font-medium">
            {humanize(label ?? path)}
            {required && <span className="ml-1 text-red-500">*</span>}
          </p>
          <p className="mt-1 text-[11px] text-muted-foreground">
            {schema.description ?? 'Add key-value configuration entries.'}
          </p>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setEntries([...entries, { id: crypto.randomUUID(), key: '', value: '' }])}
        >
          <Plus className="size-3.5" />
          Add
        </Button>
      </div>
      {!entries.length ? (
        <Button
          type="button"
          variant="ghost"
          className="h-auto w-full rounded-none px-4 py-5 text-xs text-muted-foreground"
          onClick={() => setEntries([{ id: crypto.randomUUID(), key: '', value: '' }])}
        >
          No entries. Add one if this plugin needs it.
        </Button>
      ) : (
        <div className="space-y-2 p-3">
          {entries.map((entry, index) => (
            <div
              key={entry.id}
              className="grid grid-cols-[minmax(0,.8fr)_minmax(0,1.2fr)_36px] items-start gap-2"
            >
              <Input
                aria-label={`${label ?? path} key ${index + 1}`}
                className="font-mono text-xs"
                value={entry.key}
                placeholder="key"
                onChange={(event) => {
                  const next = [...entries]
                  next[index] = { ...entry, key: event.target.value }
                  updateEntries(next)
                }}
              />
              {schemaType(valueSchema) === 'string' || !schemaType(valueSchema) ? (
                valueSchema.format === 'json' ? (
                  <Textarea
                    aria-label={`${label ?? path} value ${index + 1}`}
                    rows={2}
                    className="font-mono text-xs"
                    value={displayString(entry.value)}
                    placeholder='{"type":"object"}'
                    onChange={(event) => {
                      const next = [...entries]
                      next[index] = { ...entry, value: event.target.value }
                      updateEntries(next)
                    }}
                  />
                ) : (
                  <Input
                    aria-label={`${label ?? path} value ${index + 1}`}
                    className="font-mono text-xs"
                    value={displayString(entry.value)}
                    placeholder="value"
                    onChange={(event) => {
                      const next = [...entries]
                      next[index] = { ...entry, value: event.target.value }
                      updateEntries(next)
                    }}
                  />
                )
              ) : (
                <JsonValueField
                  schema={valueSchema}
                  value={entry.value}
                  onChange={(nextValue) => {
                    const next = [...entries]
                    next[index] = { ...entry, value: nextValue }
                    updateEntries(next)
                  }}
                  path={`${path}.${entry.key || index}`}
                />
              )}
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label={`Remove ${label ?? path} entry ${index + 1}`}
                onClick={() =>
                  updateEntries(entries.filter((_, entryIndex) => entryIndex !== index))
                }
              >
                <Trash2 className="size-3.5" />
              </Button>
            </div>
          ))}
        </div>
      )}
    </fieldset>
  )
}

function ObjectField({
  schema,
  definitions,
  value,
  onChange,
  label,
  required,
  path,
  depth = 0,
}: SchemaFieldProps) {
  const objectValue =
    value && typeof value === 'object' && !Array.isArray(value)
      ? (value as Record<string, unknown>)
      : {}
  const properties = Object.entries(schema.properties ?? {})

  if (!properties.length && schema.additionalProperties !== false) {
    return (
      <AdditionalPropertiesField
        schema={schema}
        definitions={definitions}
        value={objectValue}
        onChange={onChange}
        label={label}
        required={required}
        path={path}
      />
    )
  }

  const fields = (
    <div className="space-y-4">
      {properties.map(([name, propertySchema]) => (
        <SchemaField
          key={name}
          schema={propertySchema}
          definitions={definitions}
          value={objectValue[name]}
          onChange={(nextValue) => {
            const next = { ...objectValue }
            if (nextValue === undefined) delete next[name]
            else next[name] = nextValue
            onChange(next)
          }}
          label={name}
          required={schema.required?.includes(name)}
          path={`${path}.${name}`}
          depth={depth + 1}
        />
      ))}
      {schema.additionalProperties && (
        <AdditionalPropertiesField
          schema={schema}
          definitions={definitions}
          value={objectValue}
          onChange={onChange}
          label="Additional fields"
          path={`${path}.additional`}
        />
      )}
      {!properties.length && (
        <p className="rounded-lg border border-dashed border-border px-4 py-6 text-center text-xs text-muted-foreground">
          This plugin has no configuration fields.
        </p>
      )}
    </div>
  )

  if (!label || depth === 0) return fields
  return (
    <fieldset className="rounded-lg border border-border p-4">
      <legend className="px-1 text-xs font-medium">
        {humanize(label)}
        {required && <span className="ml-1 text-red-500">*</span>}
      </legend>
      {schema.description && (
        <p className="mb-3 text-[11px] leading-4 text-muted-foreground">{schema.description}</p>
      )}
      {fields}
    </fieldset>
  )
}

function JsonValueField({
  schema,
  value,
  onChange,
  label,
  path,
}: Pick<SchemaFieldProps, 'schema' | 'value' | 'onChange' | 'label' | 'path'>) {
  const [text, setText] = useState(() => JSON.stringify(value ?? {}, null, 2))
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setText(JSON.stringify(value ?? {}, null, 2))
    setError(null)
  }, [value])

  return (
    <div>
      {fieldLabel(label ?? 'JSON configuration', false)}
      <Textarea
        id={path}
        rows={6}
        aria-invalid={Boolean(error)}
        className="font-mono text-xs"
        value={text}
        onChange={(event) => {
          const nextText = event.target.value
          setText(nextText)
          try {
            const parsed = JSON.parse(nextText) as unknown
            setError(null)
            onChange(parsed)
          } catch {
            setError('Enter valid JSON.')
          }
        }}
      />
      {error ? <p className="field-error">{error}</p> : helpText(schema)}
    </div>
  )
}

function SchemaField(props: SchemaFieldProps) {
  const schema = resolveSchema(props.schema, props.definitions)
  if (schema.oneOf?.length || schema.anyOf?.length) {
    return <OneOfField {...props} schema={schema} />
  }

  switch (schemaType(schema)) {
    case 'object':
      return <ObjectField {...props} schema={schema} />
    case 'array':
      return <ArrayField {...props} schema={schema} />
    case 'boolean':
      return <BooleanField {...props} schema={schema} />
    case 'integer':
    case 'number':
      return <NumberField {...props} schema={schema} />
    case 'string':
      return <StringField {...props} schema={schema} />
    default:
      return <JsonValueField {...props} schema={schema} />
  }
}

export function JsonSchemaForm({ schema, definitions = {}, value, onChange }: JsonSchemaFormProps) {
  const resolved = useMemo(() => resolveSchema(schema, definitions), [schema, definitions])
  return (
    <SchemaField
      schema={resolved}
      definitions={definitions}
      value={value}
      onChange={(next) =>
        onChange(
          next && typeof next === 'object' && !Array.isArray(next)
            ? (next as Record<string, unknown>)
            : {},
        )
      }
      path="plugin-config"
    />
  )
}
