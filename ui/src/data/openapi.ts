import bundledOpenApiDocument from '../../../openapi.yml?json'
import type { SchemaDefinitions } from '@/lib/json-schema'

export type OpenApiDocument = {
  openapi?: string
  info?: {
    title?: string
    version?: string
  }
  paths?: Record<string, Record<string, unknown>>
  components?: {
    schemas?: SchemaDefinitions
  }
}

export const openApiDocument = bundledOpenApiDocument as OpenApiDocument
export const openApiSchemaDefinitions = openApiDocument.components?.schemas ?? {}
