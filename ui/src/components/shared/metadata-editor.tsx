import { Plus, Tag, Trash2 } from 'lucide-react'
import { z } from 'zod'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

export interface MetadataEntry {
  key: string
  value: string
}

const metadataEntrySchema = z.object({
  key: z.string(),
  value: z.string(),
})

export const metadataEntriesSchema = z
  .array(metadataEntrySchema)
  .superRefine((entries, context) => {
    const keys = new Map<string, number>()

    entries.forEach((entry, index) => {
      const key = entry.key.trim()
      const value = entry.value.trim()
      if (!key && value) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          path: [index, 'key'],
          message: 'Enter a key.',
        })
        return
      }
      if (!key) return

      const firstIndex = keys.get(key)
      if (firstIndex !== undefined) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          path: [index, 'key'],
          message: 'Keys must be unique.',
        })
        context.addIssue({
          code: z.ZodIssueCode.custom,
          path: [firstIndex, 'key'],
          message: 'Keys must be unique.',
        })
        return
      }
      keys.set(key, index)
    })
  })

export function metadataToEntries(
  metadata: Record<string, string> | null | undefined,
): MetadataEntry[] {
  return Object.entries(metadata ?? {}).map(([key, value]) => ({ key, value }))
}

export function metadataEntriesToRecord(entries: MetadataEntry[]): Record<string, string> {
  return Object.fromEntries(
    entries
      .map(({ key, value }) => [key.trim(), value.trim()] as const)
      .filter(([key]) => Boolean(key)),
  )
}

interface MetadataEditorProps {
  value: MetadataEntry[]
  onChange: (value: MetadataEntry[]) => void
  idPrefix: string
  label?: string
  description?: string
  className?: string
}

export function MetadataEditor({
  value,
  onChange,
  idPrefix,
  label = 'Metadata',
  description = 'Add optional key-value pairs for filtering, grouping, and automation.',
  className,
}: MetadataEditorProps) {
  const normalizedKeys = value.map((entry) => entry.key.trim())

  const updateEntry = (index: number, field: keyof MetadataEntry, nextValue: string) => {
    onChange(
      value.map((entry, entryIndex) =>
        entryIndex === index ? { ...entry, [field]: nextValue } : entry,
      ),
    )
  }

  const addEntry = () => onChange([...value, { key: '', value: '' }])

  return (
    <fieldset className={cn('rounded-lg border border-border', className)}>
      <legend className="sr-only">{label}</legend>
      <div className="flex items-start justify-between gap-4 border-b border-border px-4 py-3">
        <div>
          <div className="flex items-center gap-2">
            <Tag className="size-3.5 text-muted-foreground" />
            <p className="text-xs font-medium">{label}</p>
            {value.length > 0 && (
              <span className="rounded-full bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">
                {value.length}
              </span>
            )}
          </div>
          <p className="mt-1 text-[11px] leading-4 text-muted-foreground">{description}</p>
        </div>
        <Button type="button" variant="outline" size="sm" onClick={addEntry}>
          <Plus className="size-3.5" />
          Add field
        </Button>
      </div>

      {value.length === 0 ? (
        <Button
          type="button"
          variant="ghost"
          onClick={addEntry}
          className="group h-auto w-full rounded-none px-4 py-6 text-xs text-muted-foreground"
        >
          <span className="grid size-7 place-items-center rounded-md border border-dashed border-border group-hover:border-primary/50">
            <Plus className="size-3.5" />
          </span>
          Add your first metadata field
        </Button>
      ) : (
        <div className="space-y-2 p-3">
          <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_36px] gap-2 px-1 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
            <span>Key</span>
            <span>Value</span>
            <span className="sr-only">Actions</span>
          </div>
          {value.map((entry, index) => {
            const key = normalizedKeys[index]
            const duplicate =
              Boolean(key) &&
              normalizedKeys.some(
                (candidate, candidateIndex) => candidateIndex !== index && candidate === key,
              )
            const missingKey = !key && Boolean(entry.value.trim())
            const keyError = duplicate
              ? 'This key is already used.'
              : missingKey
                ? 'Enter a key for this value.'
                : undefined
            const keyId = `${idPrefix}-metadata-key-${index}`
            const valueId = `${idPrefix}-metadata-value-${index}`

            return (
              <div
                key={index}
                className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_36px] items-start gap-2"
              >
                <div>
                  <label className="sr-only" htmlFor={keyId}>
                    Metadata key {index + 1}
                  </label>
                  <Input
                    id={keyId}
                    value={entry.key}
                    onChange={(event) => updateEntry(index, 'key', event.target.value)}
                    placeholder="environment"
                    className={cn(
                      'font-mono text-xs',
                      keyError && 'border-red-500 focus:border-red-500 focus:ring-red-500/15',
                    )}
                    aria-invalid={Boolean(keyError)}
                    aria-describedby={keyError ? `${keyId}-error` : undefined}
                  />
                  {keyError && (
                    <p id={`${keyId}-error`} className="mt-1 text-[11px] text-red-500">
                      {keyError}
                    </p>
                  )}
                </div>
                <div>
                  <label className="sr-only" htmlFor={valueId}>
                    Metadata value {index + 1}
                  </label>
                  <Input
                    id={valueId}
                    value={entry.value}
                    onChange={(event) => updateEntry(index, 'value', event.target.value)}
                    placeholder="production"
                    className="font-mono text-xs"
                  />
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="text-muted-foreground hover:text-red-500"
                  onClick={() => onChange(value.filter((_, entryIndex) => entryIndex !== index))}
                  aria-label={`Remove metadata field ${index + 1}`}
                >
                  <Trash2 className="size-3.5" />
                </Button>
              </div>
            )
          })}
        </div>
      )}
    </fieldset>
  )
}
