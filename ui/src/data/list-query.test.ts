import { describe, expect, it } from 'vitest'
import {
  createListQueryParams,
  listQueryFingerprint,
  listQueryParamsFromSearchParams,
  listQueryString,
  mergeListQueryParams,
  workspacePlaceholderData,
} from '@/data/list-query'

describe('list query parameters', () => {
  it('sanitizes limits, strings, cursors, and metadata', () => {
    expect(
      createListQueryParams({
        limit: 0,
        sort: 'id.asc',
        after: 'next',
        before: 'previous',
        name: '  orders  ',
        metadata: { ' tenant ': 'acme', ' ': 'ignored' },
      }),
    ).toEqual({
      limit: 1,
      sort: 'id.asc',
      after: 'next',
      name: 'orders',
      metadata: { tenant: 'acme' },
    })
  })

  it('round-trips every supported event filter without dropping created-at or metadata', () => {
    const params = createListQueryParams({
      limit: 50,
      sort: 'id.desc',
      event_type: 'invoice.paid',
      unique_id: 'external-42',
      created_at: { gte: 1_700_000_000_000, lt: 1_800_000_000_000 },
      ingested_at: { eq: 1_750_000_000_000, lte: 1_760_000_000_000 },
      metadata: { tenant: 'acme' },
    })

    const query = listQueryString(params)
    expect(query).toContain('created_at%5Bgte%5D=1700000000000')
    expect(query).toContain('metadata%5Btenant%5D=acme')
    expect(listQueryParamsFromSearchParams(new URLSearchParams(query))).toEqual(params)
  })

  it('canonicalizes a lone equality filter and preserves mixed equality ranges', () => {
    expect(createListQueryParams({ created_at: { eq: 123 } }).created_at).toBe(123)

    const parsed = listQueryParamsFromSearchParams(
      new URLSearchParams('created_at=123&created_at%5Bgte%5D=100'),
    )
    expect(parsed.created_at).toEqual({ eq: 123, gte: 100 })
  })

  it('merges metadata and produces stable fingerprints regardless of key order', () => {
    const merged = mergeListQueryParams(
      { metadata: { region: 'us', tenant: 'old' } },
      { metadata: { tenant: 'new' } },
    )
    expect(merged.metadata).toEqual({ region: 'us', tenant: 'new' })

    const left = createListQueryParams({ metadata: { region: 'us', tenant: 'acme' } })
    const right = createListQueryParams({ metadata: { tenant: 'acme', region: 'us' } })
    expect(listQueryFingerprint(left)).toBe(listQueryFingerprint(right))
  })

  it('keeps placeholder data only when the workspace is unchanged', () => {
    const page = { data: [{ id: 'source-a' }] }
    const previousQuery = { queryKey: ['workspaces', 'workspace-a', 'sources'] }

    expect(workspacePlaceholderData('workspace-a', page, previousQuery)).toBe(page)
    expect(workspacePlaceholderData('workspace-b', page, previousQuery)).toBeUndefined()
  })
})
