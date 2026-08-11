import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { toast } from 'sonner'
import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { useMatch, useNavigate } from 'react-router-dom'
import {
  Boxes,
  ChevronLeft,
  ChevronRight,
  CircleAlert,
  ExternalLink,
  MoreHorizontal,
  Pencil,
  Plus,
  Trash2,
} from 'lucide-react'
import { api } from '@/data/api'
import { cloneListQueryParams, createListQueryParams } from '@/data/list-query'
import type { ListQueryParams, Workspace } from '@/types'
import { useWorkspaceName, workspacePath } from '@/app/workspace'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { AppDialog } from '@/components/shared/app-dialog'
import { Input } from '@/components/ui/input'
import { NativeSelect } from '@/components/ui/native-select'
import {
  MetadataEditor,
  metadataEntriesToRecord,
  metadataToEntries,
} from '@/components/shared/metadata-editor'
import { QueryFilter, type QueryFilterConfig } from '@/components/shared/query-filter'
import {
  CreateWorkspaceDialog,
  workspaceSchema,
  type WorkspaceForm,
} from '@/components/workspaces/create-workspace-dialog'
import { errorMessage } from '@/lib/utils'
import { useAppStore } from '@/app/store'
import { Timestamp } from '@/components/shared/timestamp'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

const workspaceDefaultParams = createListQueryParams({
  limit: 20,
  sort: 'id.asc',
})

const workspaceFilterConfig: QueryFilterConfig = {
  showSort: true,
  fields: [
    {
      key: 'name',
      type: 'string',
      label: 'Name',
      placeholder: 'Filter by exact workspace name…',
      quickSearch: true,
    },
    { key: 'created_at', type: 'created_at', label: 'Created' },
    { key: 'metadata', type: 'metadata', label: 'Metadata' },
  ],
}

export function WorkspaceManagement() {
  const activeWorkspaceName = useWorkspaceName()
  const setActiveWorkspaceName = useAppStore((state) => state.setActiveWorkspaceName)
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const createOpen = Boolean(useMatch('/settings/workspaces/create'))
  const [editing, setEditing] = useState<Workspace | null>(null)
  const [deleting, setDeleting] = useState<Workspace | null>(null)
  const [listParams, setListParams] = useState<ListQueryParams>(() =>
    cloneListQueryParams(workspaceDefaultParams),
  )
  const workspaceListQueryKey = ['workspace-list', listParams] as const
  const {
    data: page,
    isLoading,
    isFetching,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: workspaceListQueryKey,
    queryFn: () => api.workspacePage(listParams),
    placeholderData: (previous) => previous,
  })
  const workspaces = page?.data ?? []

  const editForm = useForm<WorkspaceForm>({
    resolver: zodResolver(workspaceSchema),
    defaultValues: { name: '', description: '', metadata: [] },
  })

  const updateWorkspace = useMutation({
    mutationFn: ({ workspace, values }: { workspace: Workspace; values: WorkspaceForm }) =>
      api.updateWorkspace(workspace.id, {
        name: values.name,
        description: values.description,
        metadata: metadataEntriesToRecord(values.metadata),
      }),
    onSuccess: (updated, { workspace }) => {
      queryClient.setQueryData<Workspace[]>(['workspaces'], (current = []) =>
        current.map((item) => (item.id === updated.id ? updated : item)),
      )
      void queryClient.invalidateQueries({ queryKey: ['workspace-list'] })
      setEditing(null)
      toast.success(`${updated.name} workspace updated.`)
      if (
        workspace.name === activeWorkspaceName &&
        updated.name &&
        updated.name !== activeWorkspaceName
      ) {
        setActiveWorkspaceName(updated.name)
        void navigate('/settings/workspaces', { replace: true })
      }
    },
  })

  const deleteWorkspace = useMutation({
    mutationFn: (workspace: Workspace) => api.deleteWorkspace(workspace.id),
    onSuccess: (_, workspace) => {
      queryClient.setQueryData<Workspace[]>(['workspaces'], (current = []) =>
        current.filter((item) => item.id !== workspace.id),
      )
      void queryClient.invalidateQueries({ queryKey: ['workspace-list'] })
      setDeleting(null)
      toast.success(`${workspace.name} workspace deleted.`)
      if (workspace.name === activeWorkspaceName) {
        setActiveWorkspaceName('default')
        void navigate('/settings/workspaces', { replace: true })
      }
    },
  })

  const openCreate = () => {
    void navigate('/settings/workspaces/create')
  }

  const closeCreate = () => void navigate('/settings/workspaces', { replace: true })

  const openEdit = (workspace: Workspace) => {
    updateWorkspace.reset()
    editForm.reset({
      name: workspace.name ?? '',
      description: workspace.description ?? '',
      metadata: metadataToEntries(workspace.metadata),
    })
    setEditing(workspace)
  }

  const openDelete = (workspace: Workspace) => {
    deleteWorkspace.reset()
    setDeleting(workspace)
  }

  const moveToCursor = (link: string | null | undefined) => {
    if (!link) return
    const url = new URL(link, window.location.origin)
    setListParams((current) => ({
      ...current,
      after: url.searchParams.get('after') || undefined,
      before: url.searchParams.get('before') || undefined,
    }))
  }

  return (
    <>
      <Card className="overflow-hidden">
        <div className="flex flex-col gap-3 border-b border-border px-5 py-4 sm:flex-row sm:items-center">
          <div className="min-w-0 flex-1">
            <h2 className="text-sm font-semibold">Workspaces</h2>
            <p className="mt-1 text-xs text-muted-foreground">
              Create and manage isolated environments for WebhookX resources.
            </p>
          </div>
          <Button onClick={openCreate}>
            <Plus className="size-3.5" />
            Create workspace
          </Button>
        </div>

        <QueryFilter
          value={listParams}
          onChange={setListParams}
          config={workspaceFilterConfig}
          storageKey="webhookx:workspace-views"
          defaultParams={workspaceDefaultParams}
          resultCount={workspaces.length}
          total={page?.total}
          busy={isFetching && !isLoading}
        />

        {isLoading && (
          <div className="space-y-px p-1">
            {Array.from({ length: 3 }).map((_, index) => (
              <div key={index} className="h-16 animate-pulse rounded bg-muted/40" />
            ))}
          </div>
        )}

        {isError && (
          <div className="m-5 rounded-lg border border-red-500/20 bg-red-500/5 p-4">
            <div className="flex gap-3">
              <CircleAlert className="mt-0.5 size-4 shrink-0 text-red-500" />
              <div>
                <p className="text-xs font-medium">Could not load workspaces</p>
                <p className="mt-1 text-xs text-muted-foreground">{errorMessage(error)}</p>
              </div>
            </div>
            <Button variant="outline" size="sm" className="mt-3" onClick={() => refetch()}>
              Try again
            </Button>
          </div>
        )}

        {!isLoading && !isError && (
          <div className="overflow-x-auto">
            <Table className="data-table">
              <TableHeader>
                <TableRow>
                  <TableHead>Workspace</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead>Workspace ID</TableHead>
                  <TableHead>Updated</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {workspaces.map((workspace) => {
                  const isActive = workspace.name === activeWorkspaceName
                  const isDefault = workspace.name === 'default'
                  return (
                    <TableRow key={workspace.id} className="[&>td]:h-12 [&>td]:py-0">
                      <TableCell>
                        <div className="flex items-center gap-3">
                          <span className="grid size-8 shrink-0 place-items-center rounded-md border border-border bg-muted">
                            <Boxes className="size-4 text-muted-foreground" />
                          </span>
                          <div>
                            <div className="flex items-center gap-2">
                              <span className="font-medium">
                                {workspace.name ?? 'Unnamed workspace'}
                              </span>
                              {isActive && (
                                <Badge className="border-violet-500/20 bg-violet-500/10 text-violet-600 dark:text-violet-300">
                                  Current
                                </Badge>
                              )}
                              {isDefault && <Badge>Default</Badge>}
                            </div>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className="max-w-xs truncate text-muted-foreground">
                        {workspace.description || '—'}
                      </TableCell>
                      <TableCell>
                        <span className="mono-id">{workspace.id}</span>
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-muted-foreground">
                        <Timestamp value={workspace.updated_at} />
                      </TableCell>
                      <TableCell>
                        <div className="flex justify-end">
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button
                                variant="ghost"
                                size="icon"
                                aria-label={`Actions for ${workspace.name || 'workspace'}`}
                              >
                                <MoreHorizontal className="size-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end" className="w-36">
                              {!isActive && workspace.name && (
                                <DropdownMenuItem
                                  onSelect={() =>
                                    void navigate(workspacePath(workspace.name!, 'overview'))
                                  }
                                >
                                  <ExternalLink className="text-muted-foreground" />
                                  Open
                                </DropdownMenuItem>
                              )}
                              <DropdownMenuItem onSelect={() => openEdit(workspace)}>
                                <Pencil className="text-muted-foreground" />
                                Edit
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                variant="destructive"
                                disabled={isActive || isDefault}
                                title={
                                  isActive
                                    ? 'Switch workspaces before deleting this workspace'
                                    : isDefault
                                      ? 'The default workspace cannot be deleted'
                                      : 'Delete workspace'
                                }
                                onSelect={() => openDelete(workspace)}
                              >
                                <Trash2 />
                                Delete
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })}
                {!workspaces.length && (
                  <TableRow>
                    <TableCell colSpan={5} className="py-14 text-center text-muted-foreground">
                      No workspaces found.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        )}

        {!isLoading && !isError && (
          <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border px-3 py-2.5">
            <label className="flex items-center gap-2 text-xs text-muted-foreground">
              Rows
              <NativeSelect
                className="h-8 w-20"
                aria-label="Workspaces per page"
                value={String(listParams.limit)}
                onChange={(event) =>
                  setListParams((current) => ({
                    ...current,
                    limit: Number(event.target.value),
                    after: undefined,
                    before: undefined,
                  }))
                }
              >
                {[10, 20, 50, 100].map((limit) => (
                  <option key={limit} value={limit}>
                    {limit}
                  </option>
                ))}
              </NativeSelect>
            </label>
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => moveToCursor(page?.prev)}
                disabled={!page?.prev || isFetching}
              >
                <ChevronLeft className="size-3.5" />
                Previous
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => moveToCursor(page?.next)}
                disabled={!page?.next || isFetching}
              >
                Next
                <ChevronRight className="size-3.5" />
              </Button>
            </div>
          </div>
        )}
      </Card>

      <CreateWorkspaceDialog open={createOpen} onClose={closeCreate} />

      <AppDialog
        open={Boolean(editing)}
        onClose={() => setEditing(null)}
        title="Edit workspace"
        description="Changes are reflected in the workspace switcher immediately."
      >
        {editing && (
          <form
            className="space-y-4 p-5"
            onSubmit={editForm.handleSubmit((values) =>
              updateWorkspace.mutate({ workspace: editing, values }),
            )}
          >
            <div>
              <label className="label" htmlFor="edit-workspace-name">
                Name
              </label>
              <Input
                id="edit-workspace-name"
                {...editForm.register('name')}
                disabled={editing.name === 'default'}
              />
              {editing.name === 'default' && (
                <p className="mt-1 text-[11px] text-muted-foreground">
                  The default workspace cannot be renamed.
                </p>
              )}
              {editForm.formState.errors.name && (
                <p className="field-error">{editForm.formState.errors.name.message}</p>
              )}
            </div>
            <div>
              <label className="label" htmlFor="edit-workspace-description">
                Description
              </label>
              <Input id="edit-workspace-description" {...editForm.register('description')} />
              {editForm.formState.errors.description && (
                <p className="field-error">{editForm.formState.errors.description.message}</p>
              )}
            </div>
            <MetadataEditor
              idPrefix="edit-workspace"
              value={editForm.watch('metadata')}
              onChange={(metadata) =>
                editForm.setValue('metadata', metadata, {
                  shouldDirty: true,
                  shouldValidate: true,
                })
              }
            />
            {updateWorkspace.isError && (
              <p className="field-error">{errorMessage(updateWorkspace.error)}</p>
            )}
            <div className="flex justify-end gap-2 border-t border-border pt-4">
              <Button type="button" variant="outline" onClick={() => setEditing(null)}>
                Cancel
              </Button>
              <Button type="submit" disabled={updateWorkspace.isPending}>
                {updateWorkspace.isPending ? 'Saving…' : 'Save changes'}
              </Button>
            </div>
          </form>
        )}
      </AppDialog>

      <AppDialog
        open={Boolean(deleting)}
        onClose={() => setDeleting(null)}
        title="Delete workspace"
        description="This operation cannot be undone."
      >
        {deleting && (
          <div className="p-5">
            <div className="rounded-lg border border-red-500/20 bg-red-500/5 p-4">
              <p className="text-sm font-medium">Delete {deleting.name}?</p>
              <p className="mt-1 text-xs leading-5 text-muted-foreground">
                All resources associated with this workspace may become inaccessible. Make sure its
                configuration has been backed up.
              </p>
            </div>
            {deleteWorkspace.isError && (
              <p className="field-error mt-3">{errorMessage(deleteWorkspace.error)}</p>
            )}
            <div className="mt-5 flex justify-end gap-2">
              <Button variant="outline" onClick={() => setDeleting(null)}>
                Cancel
              </Button>
              <Button
                variant="destructive"
                onClick={() => deleteWorkspace.mutate(deleting)}
                disabled={deleteWorkspace.isPending}
              >
                {deleteWorkspace.isPending ? 'Deleting…' : 'Delete workspace'}
              </Button>
            </div>
          </div>
        )}
      </AppDialog>
    </>
  )
}
