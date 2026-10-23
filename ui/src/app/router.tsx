import { lazy, Suspense, type ReactNode } from 'react'
import { createBrowserRouter, Navigate } from 'react-router-dom'
import { AppShell } from '@/components/layout/app-shell'

const DashboardPage = lazy(() =>
  import('@/pages/dashboard').then((module) => ({
    default: module.DashboardPage,
  })),
)
const SourcesPage = lazy(() =>
  import('@/pages/sources').then((module) => ({ default: module.SourcesPage })),
)
const EndpointsPage = lazy(() =>
  import('@/pages/endpoints').then((module) => ({
    default: module.EndpointsPage,
  })),
)
const EventsPage = lazy(() =>
  import('@/pages/events').then((module) => ({ default: module.EventsPage })),
)
const DeliveriesPage = lazy(() =>
  import('@/pages/deliveries').then((module) => ({
    default: module.DeliveriesPage,
  })),
)
const PluginsPage = lazy(() =>
  import('@/pages/plugins').then((module) => ({ default: module.PluginsPage })),
)
const OpenApiPage = lazy(() =>
  import('@/pages/openapi').then((module) => ({ default: module.OpenApiPage })),
)
const SettingsPage = lazy(() =>
  import('@/pages/settings').then((module) => ({
    default: module.SettingsPage,
  })),
)

function route(element: ReactNode) {
  return (
    <Suspense
      fallback={<div className="h-72 animate-pulse rounded-xl border border-border bg-muted/40" />}
    >
      {element}
    </Suspense>
  )
}

export const router = createBrowserRouter([
  {
    path: '/workspaces/:workspaceName',
    element: <AppShell />,
    children: [
      { index: true, element: <Navigate to="overview" replace /> },
      { path: 'overview', element: route(<DashboardPage />) },
      {
        path: 'sources',
        element: route(<SourcesPage />),
        children: [
          { path: 'create', element: <></> },
          { path: ':sourceId/edit', element: <></> },
        ],
      },
      {
        path: 'endpoints',
        element: route(<EndpointsPage />),
        children: [
          { path: 'create', element: <></> },
          { path: ':endpointId/edit', element: <></> },
        ],
      },
      {
        path: 'events',
        element: route(<EventsPage />),
        children: [{ path: ':eventId', element: <></> }],
      },
      {
        path: 'deliveries',
        element: route(<DeliveriesPage />),
        children: [{ path: ':attemptId', element: <></> }],
      },
      { path: 'plugins', element: route(<PluginsPage />) },
      { path: 'openapi', element: <Navigate to="/openapi" replace /> },
      { path: 'settings', element: <Navigate to="/settings" replace /> },
      { path: '*', element: <Navigate to="overview" replace /> },
    ],
  },
  {
    path: '/settings',
    element: <AppShell />,
    children: [
      { index: true, element: route(<SettingsPage />) },
      { path: ':section', element: route(<SettingsPage />) },
      { path: ':section/:action', element: route(<SettingsPage />) },
    ],
  },
  {
    path: '/openapi',
    element: <AppShell />,
    children: [{ index: true, element: route(<OpenApiPage />) }],
  },
  {
    path: '/',
    element: <Navigate to="/workspaces/default/overview" replace />,
  },
  {
    path: '*',
    element: <Navigate to="/workspaces/default/overview" replace />,
  },
])
