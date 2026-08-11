import { useMemo, useRef, useState } from 'react'
import { autocompletion, completionKeymap, startCompletion } from '@codemirror/autocomplete'
import { yaml } from '@codemirror/lang-yaml'
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language'
import { lintGutter, linter } from '@codemirror/lint'
import { Prec } from '@codemirror/state'
import { EditorView, keymap } from '@codemirror/view'
import { tags } from '@lezer/highlight'
import CodeMirror, { type Statistics } from '@uiw/react-codemirror'
import {
  AlignLeft,
  Braces,
  CheckCircle2,
  Circle,
  CircleAlert,
  FileCode2,
  Sparkles,
  Upload,
} from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { analyzeDeclarativeYaml, type DeclarativeSummary } from '@/lib/declarative-yaml'
import { declarativeYamlCompletionSource } from '@/lib/declarative-yaml-completion'
import { cn } from '@/lib/utils'

const yamlEditorTheme = EditorView.theme({
  '&': {
    backgroundColor: 'transparent',
    color: 'var(--foreground)',
    fontSize: '13px',
  },
  '&.cm-focused': { outline: 'none' },
  '.cm-scroller': {
    fontFamily:
      'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace',
    lineHeight: '1.65',
  },
  '.cm-content': { padding: '12px 0', caretColor: 'var(--primary)' },
  '.cm-line': { padding: '0 16px 0 8px' },
  '.cm-gutters': {
    backgroundColor: 'color-mix(in oklch, var(--muted) 45%, transparent)',
    borderRight: '1px solid var(--border)',
    color: 'var(--muted-foreground)',
  },
  '.cm-lineNumbers .cm-gutterElement': { minWidth: '36px', padding: '0 9px 0 5px' },
  '.cm-activeLine, .cm-activeLineGutter': {
    backgroundColor: 'color-mix(in oklch, var(--primary) 7%, transparent)',
  },
  '.cm-selectionBackground, &.cm-focused .cm-selectionBackground, ::selection': {
    backgroundColor: 'color-mix(in oklch, var(--primary) 22%, transparent) !important',
  },
  '.cm-foldGutter .cm-gutterElement': { padding: '0 3px' },
  '.cm-tooltip': {
    backgroundColor: 'var(--popover)',
    border: '1px solid var(--border)',
    color: 'var(--popover-foreground)',
  },
  '.cm-tooltip-lint': { borderRadius: '8px', overflow: 'hidden' },
})

const yamlHighlightStyle = HighlightStyle.define([
  { tag: [tags.propertyName, tags.attributeName], color: 'var(--primary)', fontWeight: '600' },
  { tag: [tags.string, tags.special(tags.string)], color: 'var(--foreground)' },
  {
    tag: [tags.bool, tags.number, tags.null],
    color: 'color-mix(in oklch, var(--chart-2) 80%, var(--foreground))',
  },
  { tag: [tags.comment, tags.meta], color: 'var(--muted-foreground)', fontStyle: 'italic' },
  { tag: [tags.punctuation, tags.separator], color: 'var(--muted-foreground)' },
])

const yamlLinter = linter((view) => analyzeDeclarativeYaml(view.state.doc.toString()).diagnostics, {
  delay: 250,
})

const yamlCompletionNavigationKeymap = Prec.highest(
  keymap.of(completionKeymap.filter((binding) => binding.run !== startCompletion)),
)

interface YamlCodeEditorProps {
  value: string
  onChange: (value: string) => void
  onSelectFile: (file: File | undefined) => void
  onFormat: () => void
  onInsertExample: () => void
  fileName: string
  summary: DeclarativeSummary | null
  error: string | null
  disabled?: boolean
}

function isYamlFile(file: File) {
  return /\.ya?ml$/i.test(file.name) || ['application/yaml', 'text/yaml'].includes(file.type)
}

export function YamlCodeEditor({
  value,
  onChange,
  onSelectFile,
  onFormat,
  onInsertExample,
  fileName,
  summary,
  error,
  disabled = false,
}: YamlCodeEditorProps) {
  const fileInputRef = useRef<HTMLInputElement>(null)
  const editorViewRef = useRef<EditorView | null>(null)
  const [cursor, setCursor] = useState({ line: 1, column: 1 })
  const errorCount = useMemo(
    () =>
      value.trim()
        ? analyzeDeclarativeYaml(value).diagnostics.filter(
            (diagnostic) => diagnostic.severity === 'error',
          ).length
        : 0,
    [value],
  )
  const extensions = useMemo(
    () => [
      yaml(),
      yamlLinter,
      lintGutter(),
      EditorView.lineWrapping,
      yamlEditorTheme,
      syntaxHighlighting(yamlHighlightStyle),
      autocompletion({
        override: [declarativeYamlCompletionSource],
        activateOnTyping: true,
        selectOnOpen: true,
        defaultKeymap: false,
      }),
      yamlCompletionNavigationKeymap,
      EditorView.contentAttributes.of({
        'aria-label': 'YAML',
        'aria-describedby': 'workspace-yaml-status workspace-yaml-error',
        autocapitalize: 'off',
        autocorrect: 'off',
        spellcheck: 'false',
        'data-enable-grammarly': 'false',
        'data-gramm': 'false',
        'data-gramm_editor': 'false',
      }),
    ],
    [],
  )

  const updateCursor = (statistics: Statistics) => {
    const next = {
      line: statistics.line.number,
      column: statistics.selection.main.head - statistics.line.from + 1,
    }
    setCursor((current) =>
      current.line === next.line && current.column === next.column ? current : next,
    )
  }

  return (
    <div className="space-y-2">
      <div
        className={cn(
          'overflow-hidden rounded-xl border border-border bg-background shadow-xs transition-[border-color,box-shadow]',
          'focus-within:border-primary focus-within:ring-3 focus-within:ring-ring/20',
          error && 'border-destructive/50',
        )}
        onDragOver={(event) => {
          if (event.dataTransfer.types.includes('Files')) event.preventDefault()
        }}
        onDrop={(event) => {
          const file = event.dataTransfer.files[0]
          if (!file) return
          event.preventDefault()
          if (disabled || !isYamlFile(file)) return
          onSelectFile(file)
        }}
      >
        <div className="flex min-h-11 flex-wrap items-center justify-between gap-2 border-b border-border bg-muted/30 px-2.5 py-2">
          <div className="flex min-w-0 items-center gap-2 px-1 text-xs font-medium">
            <FileCode2 className="size-3.5 shrink-0 text-primary" />
            <span className="truncate">{fileName || 'configuration.yaml'}</span>
            <Badge variant="secondary" className="h-5 rounded-md px-1.5 text-[10px]">
              YAML
            </Badge>
          </div>
          <div className="flex flex-wrap items-center gap-1">
            <input
              ref={fileInputRef}
              type="file"
              accept=".yaml,.yml,application/yaml,text/yaml"
              className="sr-only"
              tabIndex={-1}
              onChange={(event) => {
                onSelectFile(event.target.files?.[0])
                event.target.value = ''
              }}
              disabled={disabled}
              aria-hidden="true"
            />
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => fileInputRef.current?.click()}
              disabled={disabled}
            >
              <Upload />
              Upload
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={onFormat}
              disabled={disabled || !value.trim() || Boolean(error)}
            >
              <AlignLeft />
              Format
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => {
                const view = editorViewRef.current
                if (!view) return
                view.focus()
                startCompletion(view)
              }}
              disabled={disabled}
            >
              <Sparkles />
              Suggestions
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={onInsertExample}
              disabled={disabled}
            >
              <Braces />
              Insert example
            </Button>
          </div>
        </div>

        <CodeMirror
          value={value}
          aria-invalid={Boolean(error)}
          height="360px"
          theme="none"
          extensions={extensions}
          onCreateEditor={(view) => {
            editorViewRef.current = view
          }}
          onChange={onChange}
          onStatistics={updateCursor}
          placeholder={'sources:\n  - name: inbound\n\nendpoints:\n  - name: destination'}
          basicSetup={{
            autocompletion: false,
            bracketMatching: true,
            closeBrackets: true,
            completionKeymap: false,
            foldGutter: true,
            highlightActiveLine: true,
            highlightActiveLineGutter: true,
            highlightSelectionMatches: true,
            indentOnInput: true,
            lineNumbers: true,
          }}
          indentWithTab
          editable={!disabled}
          readOnly={disabled}
        />

        <div
          id="workspace-yaml-status"
          className="flex min-h-9 flex-wrap items-center justify-between gap-2 border-t border-border bg-muted/20 px-3 py-2 text-[11px] text-muted-foreground"
        >
          <div className="flex flex-wrap items-center gap-3">
            {!value.trim() ? (
              <span className="inline-flex items-center gap-1.5">
                <Circle className="size-3" /> Ready for YAML
              </span>
            ) : error ? (
              <span className="inline-flex items-center gap-1.5 font-medium text-destructive">
                <CircleAlert className="size-3" /> {errorCount}{' '}
                {errorCount === 1 ? 'issue' : 'issues'}
              </span>
            ) : (
              <span className="inline-flex items-center gap-1.5 font-medium text-emerald-600 dark:text-emerald-400">
                <CheckCircle2 className="size-3" /> Valid YAML
              </span>
            )}
            {summary && (
              <>
                <span>
                  {summary.sources} {summary.sources === 1 ? 'source' : 'sources'}
                </span>
                <span>
                  {summary.endpoints} {summary.endpoints === 1 ? 'endpoint' : 'endpoints'}
                </span>
              </>
            )}
          </div>
          <span className="font-mono tabular-nums">
            Ln {cursor.line}, Col {cursor.column}
          </span>
        </div>
      </div>

      <div id="workspace-yaml-error" aria-live="polite">
        {error && (
          <p className="flex items-start gap-1.5 text-xs leading-5 text-destructive">
            <CircleAlert className="mt-0.5 size-3.5 shrink-0" />
            <span>{error}</span>
          </p>
        )}
      </div>
    </div>
  )
}
