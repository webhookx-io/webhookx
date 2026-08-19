import type { Diagnostic } from '@codemirror/lint'
import { LineCounter, parseDocument } from 'yaml'
import type { YAMLError } from 'yaml'

export interface DeclarativeSummary {
  sources: number
  endpoints: number
}

export interface DeclarativeYamlAnalysis {
  value: DeclarativeSummary | null
  error: string | null
  diagnostics: Diagnostic[]
}

export const DECLARATIVE_YAML_EXAMPLE = `sources:
  - name: my-source
    type: http
    config:
      http:
        path: /
        methods: [ "POST" ]
        response:
          code: 200
          content_type: application/json
          body: '{"message": "OK"}'

endpoints:
  - name: my-endpoint
    request:
      timeout: 10000
      url: https://httpbin.org/anything
      method: POST
    retry:
      strategy: fixed
      config:
        attempts: [ 0, 3600, 3600 ]
    events: [ "test.created" ]
    plugins:
      - name: webhookx-signature
        config:
          signing_secret: foo
`

function diagnosticRange(position: [number, number], length: number) {
  let from = Math.min(Math.max(position[0], 0), length)
  let to = Math.min(Math.max(position[1], from), length)

  if (from === to && length > 0) {
    if (from === length) from -= 1
    else to += 1
  }

  return { from, to }
}

function parserDiagnostic(error: YAMLError, lineCounter: LineCounter, length: number): Diagnostic {
  const range = diagnosticRange(error.pos, length)
  const location = lineCounter.linePos(error.pos[0])
  return {
    ...range,
    severity: error.name === 'YAMLWarning' ? 'warning' : 'error',
    source: 'YAML',
    message: location ? `Line ${location.line}: ${error.message}` : error.message,
  }
}

function configurationDiagnostic(message: string, length: number): Diagnostic {
  return {
    ...diagnosticRange([0, Math.min(length, 1)], length),
    severity: 'error',
    source: 'WebhookX',
    message,
  }
}

export function analyzeDeclarativeYaml(value: string): DeclarativeYamlAnalysis {
  if (!value.trim()) return { value: null, error: null, diagnostics: [] }

  const lineCounter = new LineCounter()
  const document = parseDocument(value, {
    lineCounter,
    prettyErrors: false,
    strict: true,
    uniqueKeys: true,
  })
  const parserDiagnostics = [...document.errors, ...document.warnings].map((error) =>
    parserDiagnostic(error, lineCounter, value.length),
  )
  const firstParserError = parserDiagnostics.find((diagnostic) => diagnostic.severity === 'error')

  if (firstParserError) {
    return { value: null, error: firstParserError.message, diagnostics: parserDiagnostics }
  }

  let configuration: unknown
  try {
    configuration = document.toJS({ maxAliasCount: 100 })
  } catch (error) {
    const message = error instanceof Error ? error.message : 'The YAML could not be parsed.'
    const diagnostic = configurationDiagnostic(message, value.length)
    return {
      value: null,
      error: message,
      diagnostics: [...parserDiagnostics, diagnostic],
    }
  }

  let configurationError: string | null = null
  if (!configuration || typeof configuration !== 'object' || Array.isArray(configuration)) {
    configurationError = 'The YAML must contain a workspace configuration object.'
  } else {
    const record = configuration as Record<string, unknown>
    if (!Array.isArray(record.sources) || !Array.isArray(record.endpoints)) {
      configurationError = 'The YAML must contain both sources and endpoints arrays.'
    }
  }

  if (configurationError) {
    const diagnostic = configurationDiagnostic(configurationError, value.length)
    return {
      value: null,
      error: configurationError,
      diagnostics: [...parserDiagnostics, diagnostic],
    }
  }

  const record = configuration as Record<string, unknown[]>
  return {
    value: { sources: record.sources.length, endpoints: record.endpoints.length },
    error: null,
    diagnostics: parserDiagnostics,
  }
}

export function summarizeDeclarativeYaml(value: string): DeclarativeSummary {
  const analysis = analyzeDeclarativeYaml(value)
  if (!analysis.value) {
    throw new Error(analysis.error ?? 'The YAML must not be empty.')
  }
  return analysis.value
}

export function formatDeclarativeYaml(value: string) {
  const document = parseDocument(value, { prettyErrors: false, strict: true, uniqueKeys: true })
  if (document.errors.length > 0) throw document.errors[0]
  return document.toString({ indent: 2, lineWidth: 0 })
}
