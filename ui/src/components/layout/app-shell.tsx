import { useEffect, useMemo, useState } from 'react'
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { siGithub } from 'simple-icons'
import {
  Activity,
  Check,
  ChevronDown,
  FileCode2,
  Gauge,
  Inbox,
  Layers3,
  Menu,
  Moon,
  Plus,
  PlugZap,
  RadioTower,
  Search,
  Settings,
  Sun,
  Webhook,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Toaster } from '@/components/ui/sonner'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Sheet, SheetContent, SheetDescription, SheetTitle } from '@/components/ui/sheet'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useAppStore } from '@/app/store'
import { useWorkspaceName, workspacePath } from '@/app/workspace'
import { api } from '@/data/api'
import { cn } from '@/lib/utils'
import { AdminStatus, AdminStatusRefresher } from '@/components/layout/admin-status'

const navigation = [
  { name: 'Overview', path: 'overview', icon: Gauge },
  { name: 'Sources', path: 'sources', icon: Inbox },
  { name: 'Endpoints', path: 'endpoints', icon: RadioTower },
  { name: 'Events', path: 'events', icon: Activity, badge: 'Live' },
  { name: 'Deliveries', path: 'deliveries', icon: Webhook },
  { name: 'Plugins', path: 'plugins', icon: PlugZap },
]

const openApiNavigation = { name: 'OpenAPI', path: 'openapi', icon: FileCode2 }
const settingsNavigation = { name: 'Settings', path: 'settings', icon: Settings }
const globalNavigation = [openApiNavigation, settingsNavigation]

export function workspaceSwitchSearch(search: string) {
  const params = new URLSearchParams(search)
  params.delete('after')
  params.delete('before')
  const nextSearch = params.toString()
  return nextSearch ? `?${nextSearch}` : ''
}

function versionLabel(version: string) {
  const normalizedVersion = version.trim() || 'unknown'
  return normalizedVersion === 'dev' || normalizedVersion.startsWith('v')
    ? normalizedVersion
    : `v${normalizedVersion}`
}

export function Brand() {
  const configQuery = useQuery({
    queryKey: ['dashboard-config'],
    queryFn: api.dashboardConfig,
    staleTime: Infinity,
    retry: false,
  })
  const version = configQuery.data?.version?.trim() || 'unknown'
  const commitHash = configQuery.data?.commit_hash?.trim() || 'unknown'

  return (
    <div className="flex h-16 items-center gap-2.5 border-b border-border px-5">
      <div className="grid size-8 place-items-center rounded-lg bg-primary text-primary-foreground shadow-lg shadow-violet-500/20">
        <Webhook className="size-4" strokeWidth={2.4} />
      </div>
      <span className="text-[15px] font-semibold tracking-tight">WebhookX</span>
      {configQuery.isSuccess ? (
        <Tooltip delayDuration={350}>
          <TooltipTrigger asChild>
            <span
              tabIndex={0}
              className="max-w-[108px] truncate rounded border border-border bg-muted px-1.5 py-0.5 font-mono text-[9px] font-medium text-muted-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
              aria-label={`Version ${version}, commit ${commitHash}`}
            >
              {versionLabel(version)}
            </span>
          </TooltipTrigger>
          <TooltipContent side="bottom" sideOffset={6}>
            <span>Commit</span>
            <code className="font-mono">{commitHash}</code>
          </TooltipContent>
        </Tooltip>
      ) : (
        <span
          className="max-w-[108px] truncate rounded border border-border bg-muted px-1.5 py-0.5 font-mono text-[9px] font-medium text-muted-foreground"
          aria-label={
            configQuery.isError
              ? 'Build information unavailable'
              : 'Loading build information'
          }
        >
          {configQuery.isError ? 'unavailable' : 'loading…'}
        </span>
      )}
    </div>
  )
}

function Sidebar({ mobile = false }: { mobile?: boolean }) {
  const setSidebarOpen = useAppStore((state) => state.setSidebarOpen)
  const workspaceName = useWorkspaceName()
  return (
    <aside
      className={cn(
        'flex h-screen w-[244px] flex-col border-r border-border bg-card/75 backdrop-blur',
        mobile ? '' : 'hidden lg:flex sticky top-0',
      )}
    >
      <Brand />
      <nav className="flex-1 space-y-1 p-3" aria-label="Main navigation">
        <p className="px-2 pb-2 pt-1 text-[10px] font-semibold uppercase tracking-[.14em] text-muted-foreground/70">
          Manage
        </p>
        {navigation.map(({ name, path, icon: Icon, badge }) => (
          <NavLink
            key={path}
            to={workspacePath(workspaceName, path)}
            onClick={() => setSidebarOpen(false)}
            className={({ isActive }) =>
              cn(
                'group flex h-9 items-center gap-3 rounded-md px-2.5 text-[13px] font-medium transition-colors',
                isActive
                  ? 'bg-primary/10 text-primary dark:text-violet-300'
                  : 'text-muted-foreground hover:bg-muted hover:text-foreground',
              )
            }
          >
            <Icon className="size-4" strokeWidth={1.8} />
            <span className="flex-1">{name}</span>
            {badge && (
              <span className="flex items-center gap-1 text-[9px] uppercase tracking-wider text-emerald-600 dark:text-emerald-400">
                <span className="size-1.5 rounded-full bg-emerald-500" />
                {badge}
              </span>
            )}
          </NavLink>
        ))}
      </nav>
      <AdminStatus />
      <div className="space-y-1 border-t border-border p-3">
        <NavLink
          to="/openapi"
          end
          onClick={() => setSidebarOpen(false)}
          className={({ isActive }) =>
            cn(
              'group flex h-9 items-center gap-3 rounded-md px-2.5 text-[13px] font-medium transition-colors',
              isActive
                ? 'bg-primary/10 text-primary dark:text-violet-300'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground',
            )
          }
        >
          <FileCode2 className="size-4" strokeWidth={1.8} />
          <span className="flex-1">OpenAPI</span>
        </NavLink>
        <NavLink
          to="/settings"
          onClick={() => setSidebarOpen(false)}
          className={({ isActive }) =>
            cn(
              'group flex h-9 items-center gap-3 rounded-md px-2.5 text-[13px] font-medium transition-colors',
              isActive
                ? 'bg-primary/10 text-primary dark:text-violet-300'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground',
            )
          }
        >
          <Settings className="size-4" strokeWidth={1.8} />
          <span className="flex-1">Settings</span>
        </NavLink>
        {/*<Button asChild variant="ghost" className="w-full justify-start" size="sm">*/}
        {/*  <a*/}
        {/*    href="https://docs.webhookx.io/docs/"*/}
        {/*    target="_blank"*/}
        {/*    rel="noreferrer"*/}
        {/*  >*/}
        {/*    <CircleHelp className="size-4" />*/}
        {/*    Docs*/}
        {/*  </a>*/}
        {/*</Button>*/}
      </div>
    </aside>
  )
}

function WorkspaceSwitcher() {
  const workspaceName = useWorkspaceName()
  const location = useLocation()
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const {
    data: workspaces = [],
    isLoading,
    isError,
    refetch,
  } = useQuery({ queryKey: ['workspaces'], queryFn: api.workspaces })
  const namedWorkspaces = useMemo(
    () =>
      workspaces
        .filter((workspace) => Boolean(workspace.name))
        .sort((left, right) => (left.name ?? '').localeCompare(right.name ?? '')),
    [workspaces],
  )
  const normalizedQuery = query.trim().toLocaleLowerCase()
  const filteredWorkspaces = namedWorkspaces.filter((workspace) => {
    if (!normalizedQuery) return true
    return `${workspace.name ?? ''} ${workspace.description ?? ''}`
      .toLocaleLowerCase()
      .includes(normalizedQuery)
  })
  const visibleWorkspaces = filteredWorkspaces.slice(0, 50)
  const hasCurrentWorkspace = namedWorkspaces.some((workspace) => workspace.name === workspaceName)

  const switchWorkspace = (nextWorkspace: string) => {
    const pagePath = location.pathname.startsWith('/workspaces/')
      ? location.pathname.split('/').slice(3).join('/') || 'overview'
      : 'overview'
    setOpen(false)
    setQuery('')
    void navigate(
      `${workspacePath(nextWorkspace, pagePath)}${workspaceSwitchSearch(location.search)}`,
    )
  }

  return (
    <Popover
      open={open}
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen)
        if (!nextOpen) setQuery('')
      }}
    >
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="outline"
          aria-label={`Current workspace: ${workspaceName}. Switch workspace`}
          className="h-8 max-w-[150px] justify-start gap-2 px-2.5 text-xs sm:max-w-[190px]"
        >
          <Layers3 className="size-3.5 shrink-0 text-primary" strokeWidth={1.9} />
          <span className="truncate">{workspaceName}</span>
          <ChevronDown
            className={cn(
              'ml-auto size-3 shrink-0 text-muted-foreground transition-transform',
              open && 'rotate-180',
            )}
          />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        className="w-[calc(100vw-2rem)] max-w-[320px] overflow-hidden p-0"
      >
        <div className="px-3 pb-1.5 pt-3">
          <p className="text-[10px] font-semibold uppercase tracking-[.12em] text-muted-foreground">
            Switch workspace
          </p>
        </div>

        {namedWorkspaces.length > 7 && (
          <div className="px-2 pb-1.5">
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 size-3 -translate-y-1/2 text-muted-foreground" />
              <Input
                autoFocus
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="Search workspaces…"
                aria-label="Search workspaces"
                className="h-8 pl-7 text-xs"
              />
            </div>
          </div>
        )}

        <div
          role="listbox"
          aria-label="Switch workspace"
          className="max-h-64 overflow-y-auto px-2 pb-1.5"
        >
          {!hasCurrentWorkspace && !isLoading && !isError && (
            <Button
              type="button"
              role="option"
              aria-selected="true"
              onClick={() => setOpen(false)}
              variant="ghost"
              className="h-auto w-full justify-start gap-2.5 border border-primary/30 bg-primary/10 px-2.5 py-2 text-left"
            >
              <span className="min-w-0 flex-1">
                <span className="block truncate text-[13px] font-medium">{workspaceName}</span>
                <span className="mt-0.5 block truncate text-[11px] text-muted-foreground">
                  Current workspace
                </span>
              </span>
              <Check className="size-3.5 shrink-0 text-primary" />
            </Button>
          )}

          {isLoading && (
            <div className="space-y-1">
              {Array.from({ length: 3 }).map((_, index) => (
                <div key={index} className="h-12 animate-pulse rounded-md bg-muted/60" />
              ))}
            </div>
          )}

          {isError && (
            <div className="px-3 py-6 text-center">
              <p className="text-xs text-muted-foreground">Could not load workspaces.</p>
              <Button variant="outline" size="sm" className="mt-3" onClick={() => void refetch()}>
                Try again
              </Button>
            </div>
          )}

          {!isLoading &&
            !isError &&
            visibleWorkspaces.map((workspace) => {
              const selected = workspace.name === workspaceName
              return (
                <Button
                  key={workspace.id}
                  type="button"
                  role="option"
                  aria-selected={selected}
                  onClick={() => switchWorkspace(workspace.name!)}
                  variant="ghost"
                  className={cn(
                    'h-auto w-full justify-start gap-2.5 border border-transparent px-2.5 py-2 text-left',
                    selected && 'border-primary/30 bg-primary/10 hover:bg-primary/15',
                  )}
                >
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-[13px] font-medium">{workspace.name}</span>
                    {workspace.description && (
                      <span className="mt-0.5 block truncate text-[11px] text-muted-foreground">
                        {workspace.description}
                      </span>
                    )}
                  </span>
                  {selected && <Check className="size-3.5 shrink-0 text-primary" />}
                </Button>
              )
            })}

          {!isLoading && !isError && !filteredWorkspaces.length && (
            <p className="px-3 py-8 text-center text-xs text-muted-foreground">
              No workspaces match “{query}”.
            </p>
          )}
          {!isLoading && !isError && filteredWorkspaces.length > visibleWorkspaces.length && (
            <p className="px-3 py-2 text-center text-[11px] text-muted-foreground">
              Showing the first 50 results. Refine your search to narrow the list.
            </p>
          )}
        </div>

        <div className="border-t border-border p-1.5">
          <Button
            type="button"
            variant="ghost"
            onClick={() => {
              setOpen(false)
              setQuery('')
              void navigate('/settings/workspaces/create')
            }}
            className="w-full justify-start gap-2.5 px-2.5 text-xs text-primary hover:bg-primary/10 hover:text-primary"
          >
            <Plus className="size-3.5" />
            Create workspace
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  )
}

export function AppShell() {
  const { theme, setTheme, sidebarOpen, setSidebarOpen } = useAppStore()
  const location = useLocation()
  const navigate = useNavigate()
  const current = useMemo(() => {
    const globalItem = globalNavigation.find(
      (item) =>
        location.pathname === `/${item.path}` || location.pathname.startsWith(`/${item.path}/`),
    )
    if (globalItem) return globalItem.name
    const pagePath = location.pathname.split('/')[3]
    return navigation.find((item) => item.path === pagePath)?.name ?? 'Overview'
  }, [location.pathname])

  useEffect(() => {
    document.documentElement.classList.toggle('dark', theme === 'dark')
    document.documentElement.style.colorScheme = theme
  }, [theme])

  return (
    <div className="flex min-h-screen bg-background text-foreground">
      <AdminStatusRefresher />
      <Sidebar />
      <Sheet open={sidebarOpen} onOpenChange={setSidebarOpen}>
        <SheetContent
          side="left"
          className="w-[244px] gap-0 p-0 sm:max-w-[244px] lg:hidden"
          showCloseButton={false}
        >
          <SheetTitle className="sr-only">Main navigation</SheetTitle>
          <SheetDescription className="sr-only">
            Navigate between WebhookX administration pages.
          </SheetDescription>
          <Sidebar mobile />
        </SheetContent>
      </Sheet>
      <div className="min-w-0 flex-1">
        <header className="sticky top-0 z-30 flex h-16 items-center gap-2.5 border-b border-border bg-background/85 px-4 backdrop-blur-xl sm:px-6">
          <Button
            variant="ghost"
            size="icon"
            className="lg:hidden"
            onClick={() => setSidebarOpen(true)}
            aria-label="Open menu"
          >
            <Menu className="size-5" />
          </Button>
          <WorkspaceSwitcher />
          <div className="hidden min-w-0 items-center text-sm sm:flex">
            <span className="pr-2 text-muted-foreground/40">/</span>
            <span className="font-medium">{current}</span>
          </div>
          <a
            className="ml-auto inline-flex h-9 items-center gap-2 rounded-md border border-border bg-background px-2 text-xs font-medium text-muted-foreground shadow-sm transition hover:border-foreground/25 hover:bg-muted hover:text-foreground sm:px-2.5"
            href="https://github.com/webhookx-io/webhookx"
            target="_blank"
            rel="noreferrer"
            aria-label="Star WebhookX on GitHub"
          >
            <svg aria-hidden="true" viewBox="0 0 24 24" className="size-4" fill="currentColor">
              <path d={siGithub.path} />
            </svg>
            <span className="hidden sm:inline">Star</span>
          </a>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
            aria-label={`Use ${theme === 'dark' ? 'light' : 'dark'} theme`}
          >
            {theme === 'dark' ? <Sun className="size-4" /> : <Moon className="size-4" />}
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="rounded-full bg-gradient-to-br from-violet-500 to-indigo-700 text-[11px] font-semibold text-white hover:text-white"
                aria-label="Open account menu"
              >
                LY
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-44">
              <DropdownMenuLabel>Local administrator</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem onSelect={() => void navigate('/settings')}>
                <Settings />
                Settings
              </DropdownMenuItem>
              <DropdownMenuItem onSelect={() => void navigate('/openapi')}>
                <FileCode2 />
                OpenAPI
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </header>
        <main className="mx-auto max-w-[1500px] p-4 sm:p-6 lg:p-8">
          <Outlet />
        </main>
      </div>
      <Toaster theme={theme} closeButton richColors position="bottom-right" />
    </div>
  )
}
