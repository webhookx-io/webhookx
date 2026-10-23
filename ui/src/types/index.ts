export type DeliveryStatus = 'delivered' | 'failed' | 'pending' | 'retrying'

export interface Workspace {
  id: string
  name: string | null
  description: string | null
  metadata: Record<string, string>
  created_at: number
  updated_at: number
}

export interface WorkspaceInput {
  name: string
  description?: string
  metadata?: Record<string, string>
}

export interface License {
  id: string
  plan: string
  customer: string
  expired_at: string
  created_at: string
  version: string
  signature: string
}

export interface AdminInfo {
  version: string
}

export interface DashboardConfig {
  version: string
  commit_hash: string
}

export type SourceMethod = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'

export interface SourceResponse {
  code: number
  content_type: string
  body?: string
}

export interface HTTPSourceConfig {
  path?: string
  methods?: SourceMethod[]
  response?: SourceResponse | null
}

export interface RateLimit {
  quota: number
  period: number
}

export interface Source {
  id: string
  name: string | null
  enabled: boolean
  type: 'http'
  config: { http: HTTPSourceConfig }
  async: boolean
  metadata: Record<string, string>
  rate_limit: RateLimit | null
  created_at: number
  updated_at: number
}

export interface SourceInput {
  name?: string | null
  enabled: boolean
  type: 'http'
  config: { http: HTTPSourceConfig }
  async: boolean
  metadata: Record<string, string>
  rate_limit: RateLimit | null
}

export type ListSort = 'id.desc' | 'id.asc'
export type CreatedAtOperator = 'eq' | 'gt' | 'gte' | 'lt' | 'lte'

export type CreatedAtFilter = number | Partial<Record<CreatedAtOperator, number>>

export interface ListQueryParams {
  limit: number
  sort: ListSort
  after?: string
  before?: string
  name?: string
  enabled?: boolean
  created_at?: CreatedAtFilter
  event_type?: string
  unique_id?: string
  ingested_at?: CreatedAtFilter
  event_id?: string
  endpoint_id?: string
  status?: string
  attempted_at?: CreatedAtFilter
  metadata: Record<string, string>
}

export interface CursorPage<T> {
  data: T[]
  next: string | null
  prev: string | null
  total?: number
}

export interface QueryView {
  id: string
  name: string
  params: ListQueryParams
  createdAt: number
  updatedAt: number
}

export type SourceListParams = ListQueryParams
export type SourcePage = CursorPage<Source>
export type WorkspacePage = CursorPage<Workspace>

export type EndpointMethod = SourceMethod

export interface EndpointRequest {
  url: string
  method: EndpointMethod
  headers: Record<string, string> | null
  timeout: number
}

export interface EndpointRetry {
  strategy: 'fixed'
  config: {
    attempts: number[]
  }
}

export interface Endpoint {
  id: string
  name: string | null
  description: string | null
  enabled: boolean
  request: EndpointRequest
  retry: EndpointRetry
  events: string[]
  metadata: Record<string, string>
  rate_limit: RateLimit | null
  created_at: number
  updated_at: number
}

export interface EndpointInput {
  name?: string | null
  description?: string | null
  enabled: boolean
  request: EndpointRequest
  retry: EndpointRetry
  events: string[]
  metadata: Record<string, string>
  rate_limit: RateLimit | null
}

export type EndpointListParams = ListQueryParams
export type EndpointPage = CursorPage<Endpoint>

export interface WebhookEvent {
  id: string
  event_type: string
  data: Record<string, unknown>
  ingested_at: number
  unique_id: string | null
  created_at: number
  updated_at: number
}

export type EventListParams = ListQueryParams

export type EventPage = CursorPage<WebhookEvent>

export interface Delivery {
  id: string
  eventId: string
  endpoint: string
  attempt: number
  status: DeliveryStatus
  responseCode: number | null
  latency: number
  createdAt: string
}

export type AttemptStatus = 'INIT' | 'QUEUED' | 'SUCCESSFUL' | 'FAILED' | 'CANCELED'
export type AttemptTriggerMode = 'INITIAL' | 'MANUAL' | 'AUTOMATIC'
export type AttemptErrorCode = 'TIMEOUT' | 'UNKNOWN' | 'ENDPOINT_DISABLED' | 'ENDPOINT_NOT_FOUND'

export interface AttemptRequest {
  method: string
  url: string
  headers: Record<string, string> | null
  body: string | null
}

export interface AttemptResponse {
  status: number
  latency: number
  headers: Record<string, string> | null
  body: string | null
}

export interface Attempt {
  id: string
  event_id: string
  endpoint_id: string
  status: AttemptStatus
  attempt_number: number
  scheduled_at: number
  attempted_at: number | null
  trigger_mode: AttemptTriggerMode
  exhausted: boolean
  error_code: AttemptErrorCode | null
  request: AttemptRequest | null
  response: AttemptResponse | null
  created_at: number
}

export type AttemptListParams = ListQueryParams
export type AttemptPage = CursorPage<Attempt>

export type JsonSchemaType =
  'object' | 'array' | 'string' | 'number' | 'integer' | 'boolean' | 'null'

export interface JsonSchema {
  $ref?: string
  title?: string
  description?: string
  type?: JsonSchemaType | JsonSchemaType[]
  format?: string
  nullable?: boolean
  readOnly?: boolean
  default?: unknown
  example?: unknown
  enum?: Array<string | number | boolean | null>
  properties?: Record<string, JsonSchema>
  required?: string[]
  items?: JsonSchema
  additionalProperties?: boolean | JsonSchema
  oneOf?: JsonSchema[]
  anyOf?: JsonSchema[]
  allOf?: JsonSchema[]
  minLength?: number
  maxLength?: number
  minimum?: number
  maximum?: number
  minItems?: number
  maxItems?: number
  discriminator?: {
    propertyName: string
    mapping?: Record<string, string>
  }
}

export interface PluginCatalogItem {
  name: string
  type: 'inbound' | 'outbound'
  description: string
  schema: JsonSchema
}

export interface Plugin {
  id: string
  name: string
  enabled: boolean
  endpoint_id: string | null
  source_id: string | null
  config: Record<string, unknown>
  metadata: Record<string, string>
  created_at: number
  updated_at: number
}

export interface PluginInput {
  name: string
  enabled: boolean
  endpoint_id?: string | null
  source_id?: string | null
  config: Record<string, unknown>
  metadata: Record<string, string>
}

export interface ApiKey {
  id: string
  name: string
  prefix: string
  createdAt: string
  lastUsedAt: string | null
  scope: 'Admin' | 'Read & write' | 'Read only'
}
