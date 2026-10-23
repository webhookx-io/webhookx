import { toast } from 'sonner'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { api } from '@/data/api'
import type { Workspace } from '@/types'
import {
  MetadataEditor,
  metadataEntriesSchema,
  metadataEntriesToRecord,
} from '@/components/shared/metadata-editor'
import { Button } from '@/components/ui/button'
import { AppDialog } from '@/components/shared/app-dialog'
import { Input } from '@/components/ui/input'
import { errorMessage } from '@/lib/utils'

export const workspaceSchema = z.object({
  name: z
    .string()
    .min(1, 'Enter a workspace name.')
    .max(64, 'Use 64 characters or fewer.')
    .regex(/^[a-zA-Z0-9][a-zA-Z0-9._-]*$/, 'Use letters, numbers, dots, hyphens, or underscores.'),
  description: z.string().max(240, 'Use 240 characters or fewer.'),
  metadata: metadataEntriesSchema,
})

export type WorkspaceForm = z.infer<typeof workspaceSchema>

interface CreateWorkspaceDialogProps {
  open: boolean
  onClose: () => void
  onCreated?: (workspace: Workspace) => void
}

export function CreateWorkspaceDialog({ open, onClose, onCreated }: CreateWorkspaceDialogProps) {
  const queryClient = useQueryClient()
  const form = useForm<WorkspaceForm>({
    resolver: zodResolver(workspaceSchema),
    defaultValues: { name: '', description: '', metadata: [] },
  })

  const createWorkspace = useMutation({
    mutationFn: (values: WorkspaceForm) =>
      api.createWorkspace({
        name: values.name,
        description: values.description || undefined,
        metadata: metadataEntriesToRecord(values.metadata),
      }),
    onSuccess: (workspace) => {
      queryClient.setQueryData<Workspace[]>(['workspaces'], (current = []) => [
        ...current,
        workspace,
      ])
      void queryClient.invalidateQueries({ queryKey: ['workspace-list'] })
      form.reset()
      onClose()
      onCreated?.(workspace)
      toast.success(`${workspace.name} workspace created.`)
    },
  })

  const close = () => {
    createWorkspace.reset()
    form.reset({ name: '', description: '', metadata: [] })
    onClose()
  }

  return (
    <AppDialog
      open={open}
      onClose={close}
      title="Create workspace"
      description="Resources created in this workspace are isolated from every other workspace."
    >
      <form
        className="space-y-4 p-5"
        onSubmit={form.handleSubmit((values) => createWorkspace.mutate(values))}
      >
        <div>
          <label className="label" htmlFor="new-workspace-name">
            Name
          </label>
          <Input
            id="new-workspace-name"
            autoFocus
            placeholder="staging"
            {...form.register('name')}
          />
          {form.formState.errors.name && (
            <p className="field-error">{form.formState.errors.name.message}</p>
          )}
        </div>
        <div>
          <label className="label" htmlFor="new-workspace-description">
            Description
          </label>
          <Input
            id="new-workspace-description"
            placeholder="Staging webhook infrastructure"
            {...form.register('description')}
          />
          {form.formState.errors.description && (
            <p className="field-error">{form.formState.errors.description.message}</p>
          )}
        </div>
        <MetadataEditor
          idPrefix="new-workspace"
          value={form.watch('metadata')}
          onChange={(metadata) =>
            form.setValue('metadata', metadata, {
              shouldDirty: true,
              shouldValidate: true,
            })
          }
        />
        {createWorkspace.isError && (
          <p className="field-error">{errorMessage(createWorkspace.error)}</p>
        )}
        <div className="flex justify-end gap-2 border-t border-border pt-4">
          <Button type="button" variant="outline" onClick={close}>
            Cancel
          </Button>
          <Button type="submit" disabled={createWorkspace.isPending}>
            {createWorkspace.isPending ? 'Creating…' : 'Create workspace'}
          </Button>
        </div>
      </form>
    </AppDialog>
  )
}
