import { toast } from 'sonner'
import { useState, type ReactNode } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  AlertCircle,
  BadgeCheck,
  Boxes,
  Building2,
  Check,
  Copy,
  ExternalLink,
  RefreshCw,
  Save,
} from 'lucide-react'
import { PageHeader } from '@/components/shared/page-header'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { cn, copyText, errorMessage } from '@/lib/utils'
import { WorkspaceManagement } from '@/components/workspaces/workspace-management'
import { Timestamp } from '@/components/shared/timestamp'
import { useWorkspaceName } from '@/app/workspace'
import { api } from '@/data/api'

const tabs = [
  { id: 'general', label: 'General', icon: Building2 },
  { id: 'workspaces', label: 'Workspaces', icon: Boxes },
  { id: 'license', label: 'License', icon: BadgeCheck },
  // { id: 'delivery', label: 'Delivery', icon: RotateCcw },
  // { id: 'security', label: 'Security', icon: LockKeyhole },
  // { id: 'notifications', label: 'Notifications', icon: Bell },
]

function WorkspaceInformation({ workspaceName }: { workspaceName: string }) {
  const workspaceQuery = useQuery({
    queryKey: ['workspaces'],
    queryFn: api.workspaces,
    staleTime: 5 * 60_000,
  })

  if (workspaceQuery.isPending) {
    return (
      <Card aria-label="Loading workspace information">
        <CardHeader>
          <Skeleton className="h-4 w-40" />
          <Skeleton className="h-3 w-72 max-w-full" />
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2">
          {Array.from({ length: 6 }).map((_, index) => (
            <div key={index} className="space-y-2 border-t border-border pt-4">
              <Skeleton className="h-3 w-20" />
              <Skeleton className="h-4 w-40 max-w-full" />
            </div>
          ))}
        </CardContent>
      </Card>
    )
  }

  if (workspaceQuery.isError) {
    return (
      <Card className="flex min-h-72 flex-col items-center justify-center px-6 py-12 text-center">
        <div className="grid size-11 place-items-center rounded-full bg-red-500/10 text-red-500">
          <AlertCircle className="size-5" />
        </div>
        <h2 className="mt-4 text-sm font-semibold">Could not load workspace information</h2>
        <p className="mt-1 max-w-lg text-xs leading-5 text-muted-foreground">
          {errorMessage(workspaceQuery.error)}
        </p>
        <Button
          className="mt-5"
          variant="outline"
          size="sm"
          onClick={() => workspaceQuery.refetch()}
          disabled={workspaceQuery.isFetching}
        >
          <RefreshCw className={cn('size-3.5', workspaceQuery.isFetching && 'animate-spin')} />
          Retry
        </Button>
      </Card>
    )
  }

  const workspace = workspaceQuery.data.find((item) => item.name === workspaceName)
  if (!workspace) {
    return (
      <Card className="flex min-h-56 flex-col items-center justify-center px-6 py-10 text-center">
        <AlertCircle className="size-5 text-amber-500" />
        <h2 className="mt-3 text-sm font-semibold">Workspace not found</h2>
        <p className="mt-1 text-xs text-muted-foreground">
          The “{workspaceName}” workspace is no longer available.
        </p>
      </Card>
    )
  }

  const metadata = Object.entries(workspace.metadata).sort(([left], [right]) =>
    left.localeCompare(right),
  )
  const details: Array<{ label: string; value: ReactNode; mono?: boolean }> = [
    { label: 'Name', value: workspace.name || 'Unnamed workspace' },
    { label: 'Workspace ID', value: workspace.id, mono: true },
    { label: 'Description', value: workspace.description || '—' },
    { label: 'Created at', value: <Timestamp value={workspace.created_at} /> },
    { label: 'Updated at', value: <Timestamp value={workspace.updated_at} /> },
  ]

  return (
    <Card>
      <CardHeader className="border-b border-border">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h2 className="text-sm font-semibold">Workspace information</h2>
            <p className="mt-1 text-xs text-muted-foreground">
              Details for the workspace currently selected in the Dashboard.
            </p>
          </div>
          <Badge variant="secondary">Current</Badge>
        </div>
      </CardHeader>
      <CardContent className="grid sm:grid-cols-2">
        {details.map(({ label, value, mono }, index) => (
          <div
            key={label}
            className={cn(
              'min-w-0 border-b border-border py-4 sm:[&:nth-last-child(-n+1)]:border-b-0',
              index % 2 === 0 ? 'sm:pr-5' : 'sm:border-l sm:pl-5',
            )}
          >
            <p className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
              {label}
            </p>
            <div className={cn('mt-1.5 break-words text-xs', mono && 'font-mono text-[11px]')}>
              {value}
            </div>
          </div>
        ))}
        <div className="min-w-0 py-4 sm:col-span-2 sm:border-t sm:border-border">
          <p className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
            Metadata
          </p>
          {metadata.length > 0 ? (
            <dl className="mt-2 flex flex-wrap gap-2">
              {metadata.map(([key, value]) => (
                <div
                  key={key}
                  className="flex min-w-0 items-center overflow-hidden rounded-md border border-border font-mono text-[11px]"
                >
                  <dt className="bg-muted px-2 py-1 text-muted-foreground">{key}</dt>
                  <dd className="min-w-0 break-all px-2 py-1">{value}</dd>
                </div>
              ))}
            </dl>
          ) : (
            <p className="mt-1.5 text-xs text-muted-foreground">No metadata</p>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

function LicenseSection() {
  const [rawCopied, setRawCopied] = useState(false)
  const licenseQuery = useQuery({
    queryKey: ['license'],
    queryFn: api.license,
    staleTime: 5 * 60_000,
  })

  if (licenseQuery.isPending) {
    return (
      <Card aria-label="Loading license information">
        <CardHeader>
          <Skeleton className="h-4 w-32" />
          <Skeleton className="h-3 w-72 max-w-full" />
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2">
          {Array.from({ length: 6 }).map((_, index) => (
            <div key={index} className="space-y-2 border-t border-border pt-4">
              <Skeleton className="h-3 w-20" />
              <Skeleton className="h-4 w-40 max-w-full" />
            </div>
          ))}
        </CardContent>
      </Card>
    )
  }

  if (licenseQuery.isError) {
    return (
      <Card className="flex min-h-72 flex-col items-center justify-center px-6 py-12 text-center">
        <div className="grid size-11 place-items-center rounded-full bg-red-500/10 text-red-500">
          <AlertCircle className="size-5" />
        </div>
        <h2 className="mt-4 text-sm font-semibold">Could not load license information</h2>
        <p className="mt-1 max-w-lg text-xs leading-5 text-muted-foreground">
          {errorMessage(licenseQuery.error)}
        </p>
        <Button
          className="mt-5"
          variant="outline"
          size="sm"
          onClick={() => licenseQuery.refetch()}
          disabled={licenseQuery.isFetching}
        >
          <RefreshCw className={cn('size-3.5', licenseQuery.isFetching && 'animate-spin')} />
          Retry
        </Button>
      </Card>
    )
  }

  const license = licenseQuery.data
  const rawLicense = JSON.stringify(license, null, 2)
  const details: Array<{ label: string; value: ReactNode; mono?: boolean }> = [
    { label: 'Customer', value: license.customer || '—' },
    { label: 'License ID', value: license.id || '—', mono: true },
    {
      label: 'Created at',
      value: <Timestamp value={license.created_at} />,
    },
    {
      label: 'Expires at',
      value: <Timestamp value={license.expired_at} />,
    },
    { label: 'License version', value: license.version || '—' },
    {
      label: 'Signature',
      value: license.signature || 'Not provided',
      mono: Boolean(license.signature),
    },
  ]

  const copyRawLicense = async () => {
    try {
      await copyText(rawLicense)
      setRawCopied(true)
      toast.success('Raw license JSON copied.')
      window.setTimeout(() => setRawCopied(false), 1500)
    } catch (error) {
      toast.error(errorMessage(error, 'Could not copy the raw license JSON.'))
    }
  }

  return (
    <div className="space-y-5">
      <Card>
        <CardHeader className="border-b border-border">
          <div>
            <h2 className="text-sm font-semibold">License setup</h2>
            <p className="mt-1 text-xs text-muted-foreground">
              Request a license, then provide its complete JSON when WebhookX starts.
            </p>
          </div>
        </CardHeader>
        <CardContent className="grid gap-5 sm:grid-cols-2">
          <section>
            <p className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
              1. Get a license
            </p>
            <p className="mt-2 text-xs leading-5 text-muted-foreground">
              Submit the license request form. The approved license is delivered as a JSON object.
            </p>
            <Button asChild variant="outline" size="sm" className="mt-3">
              <a
                href="https://form.jotform.com/webhookx/request-a-license"
                target="_blank"
                rel="noreferrer"
              >
                Request a license
                <ExternalLink />
              </a>
            </Button>
          </section>
          <section className="border-t border-border pt-5 sm:border-l sm:border-t-0 sm:pl-5 sm:pt-0">
            <p className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
              2. Configure WebhookX
            </p>
            <p className="mt-2 text-xs leading-5 text-muted-foreground">
              Set the complete license JSON as <code className="font-mono">WEBHOOKX_LICENSE</code>{' '}
              for every WebhookX process or container, then restart it.
            </p>
            <pre className="mt-3 overflow-x-auto rounded-md border border-border bg-zinc-950 px-3 py-2 font-mono text-[11px] leading-5 text-zinc-300">
              <code>WEBHOOKX_LICENSE=&apos;&lt;license-json&gt;&apos;</code>
            </pre>
          </section>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="border-b border-border">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 className="text-sm font-semibold">License information</h2>
              <p className="mt-1 text-xs text-muted-foreground">
                License details reported by this WebhookX instance.
              </p>
            </div>
            <Badge variant="secondary" className="capitalize">
              {license.plan || 'Unknown'} plan
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="grid sm:grid-cols-2">
          {details.map(({ label, value, mono }, index) => (
            <div
              key={label}
              className={cn(
                'min-w-0 border-b border-border py-4 last:border-b-0 sm:[&:nth-last-child(-n+2)]:border-b-0',
                index % 2 === 0 ? 'sm:pr-5' : 'sm:border-l sm:pl-5',
              )}
            >
              <p className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                {label}
              </p>
              <div className={cn('mt-1.5 break-all text-xs', mono && 'font-mono text-[11px]')}>
                {value}
              </div>
            </div>
          ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="border-b border-border">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 className="text-sm font-semibold">Raw license</h2>
              <p className="mt-1 text-xs text-muted-foreground">
                Complete JSON returned by the current WebhookX instance.
              </p>
            </div>
            <Button type="button" variant="outline" size="sm" onClick={() => void copyRawLicense()}>
              {rawCopied ? <Check className="text-emerald-500" /> : <Copy />}
              {rawCopied ? 'Copied' : 'Copy JSON'}
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <pre
            aria-label="Raw license JSON"
            className="max-h-96 overflow-auto rounded-lg border border-border bg-zinc-950 p-4 font-mono text-[11px] leading-5 text-zinc-300"
          >
            <code>{rawLicense}</code>
          </pre>
        </CardContent>
      </Card>
    </div>
  )
}

function ToggleRow({
  title,
  description,
  initial = false,
}: {
  title: string
  description: string
  initial?: boolean
}) {
  const [enabled, setEnabled] = useState(initial)
  return (
    <div className="flex items-center justify-between gap-4 border-b border-border py-4 last:border-0">
      <div>
        <p className="text-xs font-medium">{title}</p>
        <p className="mt-1 text-xs leading-5 text-muted-foreground">{description}</p>
      </div>
      <Switch checked={enabled} onCheckedChange={setEnabled} aria-label={title} />
    </div>
  )
}

export function SettingsPage() {
  const workspaceName = useWorkspaceName()
  const location = useLocation()
  const requestedTab = location.pathname.split('/')[2]
  const tab = tabs.some((item) => item.id === requestedTab) ? requestedTab : 'general'
  return (
    <>
      <PageHeader
        title="Settings"
        description={`View ${workspaceName} details and manage WebhookX workspaces.`}
      />
      <div className="grid gap-5 lg:grid-cols-[210px_minmax(0,1fr)]">
        <nav className="space-y-1" aria-label="Settings sections">
          {tabs.map(({ id, label, icon: Icon }) => (
            <NavLink
              key={id}
              to={id === 'general' ? '/settings' : `/settings/${id}`}
              className={cn(
                'flex h-9 w-full items-center gap-2.5 rounded-md px-3 text-left text-xs font-medium transition-colors',
                tab === id
                  ? 'bg-muted text-foreground'
                  : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground',
              )}
            >
              <Icon className="size-4" />
              {label}
            </NavLink>
          ))}
        </nav>
        <div className={tab === 'workspaces' ? 'min-w-0' : 'max-w-[720px]'}>
          {tab === 'general' && <WorkspaceInformation workspaceName={workspaceName} />}
          {tab === 'workspaces' && <WorkspaceManagement />}
          {tab === 'license' && <LicenseSection />}
          {tab === 'delivery' && (
            <Card>
              <CardHeader>
                <div>
                  <h2 className="text-sm font-semibold">Delivery defaults</h2>
                  <p className="mt-1 text-xs text-muted-foreground">
                    Applied when an endpoint does not specify an override.
                  </p>
                </div>
              </CardHeader>
              <CardContent>
                <div className="grid gap-4 sm:grid-cols-2">
                  <div>
                    <label className="label">Request timeout</label>
                    <Input defaultValue="15 seconds" />
                  </div>
                  <div>
                    <label className="label">Maximum attempts</label>
                    <Input type="number" defaultValue="5" />
                  </div>
                </div>
                <div className="mt-5">
                  <h3 className="text-xs font-semibold">Retry schedule</h3>
                  <div className="mt-2 flex flex-wrap gap-2">
                    {['Immediately', '1 min', '5 min', '30 min', '2 hours'].map((value, index) => (
                      <span
                        key={value}
                        className="flex items-center gap-2 rounded-md border border-border bg-muted px-2.5 py-1.5 font-mono text-[10px]"
                      >
                        <span className="text-muted-foreground">{index + 1}</span>
                        {value}
                      </span>
                    ))}
                  </div>
                </div>
                <div className="mt-5">
                  <ToggleRow
                    title="Exponential backoff"
                    description="Add jitter to prevent retry storms when a destination recovers."
                    initial
                  />
                  <ToggleRow
                    title="Respect Retry-After"
                    description="Use the destination's Retry-After header when present."
                    initial
                  />
                </div>
                <div className="mt-4 flex justify-end">
                  <Button onClick={() => toast.success('Delivery policy saved.')}>
                    <Save className="size-3.5" />
                    Save policy
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}
          {tab === 'security' && (
            <Card>
              <CardHeader>
                <div>
                  <h2 className="text-sm font-semibold">Workspace security</h2>
                  <p className="mt-1 text-xs text-muted-foreground">
                    Controls that apply to administrators and API clients.
                  </p>
                </div>
              </CardHeader>
              <CardContent>
                <ToggleRow
                  title="Require multi-factor authentication"
                  description="All workspace administrators must configure a second factor."
                  initial
                />
                <ToggleRow
                  title="Enforce signed webhook payloads"
                  description="Require an HMAC signing secret on every active endpoint."
                />
                <ToggleRow
                  title="Restrict dashboard by IP"
                  description="Allow browser sessions only from configured CIDR ranges."
                />
                <div className="mt-4 flex justify-end">
                  <Button onClick={() => toast.success('Security settings saved.')}>
                    <Save className="size-3.5" />
                    Save security
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}
          {tab === 'notifications' && (
            <Card>
              <CardHeader>
                <div>
                  <h2 className="text-sm font-semibold">Operational notifications</h2>
                  <p className="mt-1 text-xs text-muted-foreground">
                    Choose when WebhookX should alert your team.
                  </p>
                </div>
              </CardHeader>
              <CardContent>
                <ToggleRow
                  title="Endpoint disabled"
                  description="Alert when WebhookX automatically disables an unhealthy endpoint."
                  initial
                />
                <ToggleRow
                  title="Failure rate threshold"
                  description="Alert when failures exceed 5% over a five-minute window."
                  initial
                />
                <ToggleRow
                  title="Queue saturation"
                  description="Alert when pending deliveries exceed 80% of capacity."
                  initial
                />
                <div className="mt-4">
                  <label className="label">Alert email</label>
                  <Input type="email" defaultValue="platform-ops@acme.dev" />
                </div>
                <div className="mt-4 flex justify-end">
                  <Button onClick={() => toast.success('Notification preferences saved.')}>
                    <Save className="size-3.5" />
                    Save preferences
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </>
  )
}
