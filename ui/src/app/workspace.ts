import { useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { useAppStore } from '@/app/store'

export function workspacePath(workspaceName: string, path = 'overview') {
  const normalizedPath = path.replace(/^\/+/, '')
  return `/workspaces/${encodeURIComponent(workspaceName)}/${normalizedPath}`
}

export function useWorkspaceName() {
  const { workspaceName } = useParams<{ workspaceName: string }>()
  const activeWorkspaceName = useAppStore((state) => state.activeWorkspaceName)
  const setActiveWorkspaceName = useAppStore((state) => state.setActiveWorkspaceName)

  useEffect(() => {
    if (workspaceName) setActiveWorkspaceName(workspaceName)
  }, [setActiveWorkspaceName, workspaceName])

  return workspaceName ?? activeWorkspaceName
}
