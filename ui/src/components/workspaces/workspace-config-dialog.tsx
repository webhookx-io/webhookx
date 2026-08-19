import { useMemo, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, Download, Loader2, RotateCcw } from 'lucide-react'
import { toast } from 'sonner'
import { AppDialog } from '@/components/shared/app-dialog'
import { YamlCodeEditor } from '@/components/workspaces/yaml-code-editor'
import { Button } from '@/components/ui/button'
import { api } from '@/data/api'
import {
  analyzeDeclarativeYaml,
  DECLARATIVE_YAML_EXAMPLE,
  formatDeclarativeYaml,
} from '@/lib/declarative-yaml'
import { errorMessage } from '@/lib/utils'

export { summarizeDeclarativeYaml } from '@/lib/declarative-yaml'

function downloadYaml(value: string, workspaceName: string) {
  const href = URL.createObjectURL(new Blob([value], { type: 'application/yaml' }))
  const anchor = document.createElement('a')
  anchor.href = href
  anchor.download = `${workspaceName}-webhookx.yml`
  anchor.click()
  URL.revokeObjectURL(href)
}

export function WorkspaceConfigDialog({
  open,
  onClose,
  workspaceId,
  workspaceName,
  hasExistingConfiguration,
}: {
  open: boolean
  onClose: () => void
  workspaceId: string | undefined
  workspaceName: string
  hasExistingConfiguration: boolean
}) {
  const queryClient = useQueryClient()
  const [yaml, setYaml] = useState('')
  const [fileName, setFileName] = useState('')
  const [confirmed, setConfirmed] = useState(false)
  const summary = useMemo(() => analyzeDeclarativeYaml(yaml), [yaml])

  const updateYaml = (value: string, selectedFileName = '') => {
    setYaml(value)
    setFileName(selectedFileName)
    setConfirmed(false)
    sync.reset()
  }

  const close = () => {
    if (sync.isPending) return
    setYaml('')
    setFileName('')
    setConfirmed(false)
    sync.reset()
    onClose()
  }

  const backup = useMutation({
    mutationFn: () => api.dumpWorkspaceConfig(workspaceId!),
    onSuccess: (configuration) => {
      downloadYaml(configuration, workspaceName)
      toast.success('Current workspace configuration downloaded.')
    },
  })

  const sync = useMutation({
    mutationFn: () => api.syncWorkspaceConfig(workspaceId!, yaml),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['workspaces', workspaceId] })
      toast.success('Workspace configuration synced.')
      setYaml('')
      setFileName('')
      setConfirmed(false)
      onClose()
    },
  })

  const selectFile = async (file: File | undefined) => {
    if (!file) return
    try {
      updateYaml(await file.text(), file.name)
    } catch {
      toast.error('Could not read the selected YAML file.')
    }
  }

  const formatYaml = () => {
    try {
      updateYaml(formatDeclarativeYaml(yaml), fileName)
      toast.success('YAML formatted.')
    } catch (error) {
      toast.error(errorMessage(error, 'Could not format invalid YAML.'))
    }
  }

  return (
    <AppDialog
      open={open}
      onClose={close}
      title="Sync YAML configuration"
      description="Apply a declarative configuration to this workspace."
      className="sm:max-w-3xl"
    >
      <div className="space-y-4 p-5">
        <YamlCodeEditor
          value={yaml}
          onChange={(value) => updateYaml(value)}
          onSelectFile={(file) => void selectFile(file)}
          onFormat={formatYaml}
          onInsertExample={() => updateYaml(DECLARATIVE_YAML_EXAMPLE)}
          fileName={fileName}
          summary={summary.value}
          error={summary.error}
          disabled={sync.isPending}
        />

        <div className="flex gap-3 rounded-lg border border-amber-500/25 bg-amber-500/5 p-3">
          <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-600 dark:text-amber-400" />
          <div>
            <p className="text-xs font-semibold">This replaces the workspace configuration</p>
            <p className="mt-1 text-xs leading-5 text-muted-foreground">
              Existing sources and endpoints that are not present in this YAML will be deleted.
            </p>
            {hasExistingConfiguration && (
              <Button
                type="button"
                variant="link"
                size="sm"
                className="mt-1 h-auto p-0 text-xs"
                onClick={() => backup.mutate()}
                disabled={!workspaceId || backup.isPending || sync.isPending}
              >
                {backup.isPending ? (
                  <Loader2 className="size-3 animate-spin" />
                ) : (
                  <Download className="size-3" />
                )}
                Download current configuration first
              </Button>
            )}
            {backup.isError && (
              <p className="field-error">Could not download backup: {errorMessage(backup.error)}</p>
            )}
          </div>
        </div>

        <label className="flex cursor-pointer items-start gap-2.5 text-xs leading-5">
          <input
            type="checkbox"
            checked={confirmed}
            onChange={(event) => setConfirmed(event.target.checked)}
            className="mt-1 size-3.5 accent-primary"
            disabled={sync.isPending}
          />
          <span>I understand that resources missing from the YAML will be removed.</span>
        </label>

        {sync.isError && (
          <p className="field-error">Could not sync configuration: {errorMessage(sync.error)}</p>
        )}
      </div>

      <div className="flex flex-col-reverse gap-2 border-t border-border bg-muted/40 p-4 sm:flex-row sm:justify-end">
        <Button type="button" variant="outline" onClick={close} disabled={sync.isPending}>
          Cancel
        </Button>
        <Button
          type="button"
          onClick={() => sync.mutate()}
          disabled={!workspaceId || !summary.value || !confirmed || sync.isPending}
        >
          {sync.isPending ? <Loader2 className="size-3.5 animate-spin" /> : <RotateCcw />}
          {sync.isPending ? 'Syncing…' : 'Sync configuration'}
        </Button>
      </div>
    </AppDialog>
  )
}
