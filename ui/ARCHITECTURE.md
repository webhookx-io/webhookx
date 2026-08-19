# WebhookX Admin information architecture

## Navigation and page hierarchy

- Overview: system health, delivery volume, latency, failure rate, recent failures.
- Endpoints: searchable inventory and endpoint creation.
- Events: live event stream, payload inspection, delivery history, retry action.
- Deliveries: attempt-level debugging with response details and retry state.
- Plugins: discover, enable, and configure gateway capabilities.
- API keys: create, copy, inspect, and revoke credentials.
- Settings: general workspace, delivery defaults, security, and notifications.

The active workspace is global application context. Resource details use a right-side
inspector so operators keep their place in dense lists.

## Component architecture

`AppShell` owns responsive navigation and global utilities. Route components own
query state and feature actions. Primitives in `components/ui` are installed from the
official shadcn registry; application-specific compositions (dialogs, status badges,
filters, and metadata editing) live in `components/shared`. Server state is represented
with TanStack Query, lightweight persisted preferences use Zustand, and validated forms
use React Hook Form with Zod.

## Folder structure

```text
src/
  app/          providers, routing, application store
  components/
    layout/     application shell and navigation
    shared/     domain-aware reusable components
    ui/         shadcn/ui-style primitives
  data/         typed API clients, query serialization, and isolated demo data
  lib/          domain-independent formatting and schema utilities
  pages/        route-level feature screens
  types/        domain models
```

The service boundary in `data/api.ts` owns HTTP and response normalization. Pages never
construct API URLs directly. Dashboard samples and the temporary API-key service remain
isolated behind the same asynchronous boundary.

## Quality gates

`npm run check` is the required local gate. It checks formatting, ESLint, TypeScript,
Vitest, and the production build. New reusable behavior should include a colocated
`*.test.ts` or `*.test.tsx` regression test.
