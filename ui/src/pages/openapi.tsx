import { FileCode2 } from 'lucide-react'
import SwaggerUI from 'swagger-ui-react'
import 'swagger-ui-react/swagger-ui.css'
import '@/styles/openapi.css'
import { PageHeader } from '@/components/shared/page-header'
import { Card } from '@/components/ui/card'
import { openApiDocument, type OpenApiDocument } from '@/data/openapi'

const operationMethods = new Set([
  'get',
  'put',
  'post',
  'delete',
  'options',
  'head',
  'patch',
  'trace',
])

function countOperations(document: OpenApiDocument) {
  return Object.values(document.paths ?? {}).reduce(
    (total, path) =>
      total + Object.keys(path).filter((key) => operationMethods.has(key.toLowerCase())).length,
    0,
  )
}

export function OpenApiPage() {
  return (
    <>
      <PageHeader
        title="OpenAPI"
        description="Browse the WebhookX Admin API contract, operations, request bodies, and response schemas."
      />

      <Card className="openapi-browser overflow-hidden">
        <div className="flex flex-wrap items-center gap-3 border-b border-border bg-muted/20 px-5 py-3">
          <div className="grid size-8 place-items-center rounded-md bg-primary/10 text-primary dark:text-violet-300">
            <FileCode2 className="size-4" />
          </div>
          <div>
            <p className="text-xs font-semibold">
              {openApiDocument.info?.title ?? 'API specification'}
            </p>
            <p className="mt-0.5 text-[10px] text-muted-foreground">
              Bundled from the repository OpenAPI definition
            </p>
          </div>
          <div className="ml-auto flex items-center gap-2">
            <span className="rounded border border-border bg-background px-2 py-1 font-mono text-[10px] text-muted-foreground">
              OpenAPI {openApiDocument.openapi}
            </span>
            {openApiDocument.info?.version && (
              <span className="rounded border border-border bg-background px-2 py-1 font-mono text-[10px] text-muted-foreground">
                v{openApiDocument.info.version}
              </span>
            )}
            <span className="rounded border border-border bg-background px-2 py-1 text-[10px] text-muted-foreground">
              {countOperations(openApiDocument)} operations
            </span>
          </div>
        </div>
        <SwaggerUI
          spec={openApiDocument}
          deepLinking
          displayOperationId
          docExpansion="list"
          defaultModelsExpandDepth={1}
          defaultModelExpandDepth={2}
          filter
          showExtensions
          showCommonExtensions
          supportedSubmitMethods={[]}
        />
      </Card>
    </>
  )
}
