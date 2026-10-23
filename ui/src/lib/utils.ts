import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export const compactNumber = new Intl.NumberFormat('en', {
  notation: 'compact',
  maximumFractionDigits: 1,
})

export type TimestampValue = string | number | Date | null | undefined

export function timestampToDate(value: TimestampValue) {
  if (value === null || value === undefined || value === '') return null

  if (value instanceof Date) {
    const date = new Date(value.getTime())
    return Number.isFinite(date.getTime()) ? date : null
  }

  if (typeof value === 'number') {
    if (!Number.isFinite(value)) return null
    const milliseconds = Math.abs(value) < 1_000_000_000_000 ? value * 1000 : value
    const date = new Date(milliseconds)
    return Number.isFinite(date.getTime()) ? date : null
  }

  const trimmed = value.trim()
  if (!trimmed) return null
  const numeric = /^-?\d+(?:\.\d+)?$/.test(trimmed) ? Number(trimmed) : null
  return timestampToDate(numeric ?? new Date(trimmed))
}

export function timeAgo(value: TimestampValue) {
  const date = timestampToDate(value)
  if (!date) return '—'
  const difference = Date.now() - date.getTime()
  if (difference < 0) return 'just now'
  const seconds = Math.max(1, Math.floor(difference / 1000))
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}

export function errorMessage(error: unknown, fallback = 'The request could not be completed.') {
  return error instanceof Error && error.message.trim() ? error.message : fallback
}

export function formatTimestamp(
  timestamp: TimestampValue,
  options: Intl.DateTimeFormatOptions = { dateStyle: 'medium', timeStyle: 'short' },
) {
  const date = timestampToDate(timestamp)
  if (!date) return '—'
  return new Intl.DateTimeFormat(undefined, options).format(date)
}

export async function copyText(value: string) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(value)
    return
  }

  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.setAttribute('readonly', '')
  textarea.style.position = 'fixed'
  textarea.style.opacity = '0'
  document.body.append(textarea)
  textarea.select()
  const copied = document.execCommand('copy')
  textarea.remove()
  if (!copied) throw new Error('Clipboard access is unavailable in this browser.')
}

export function latencyTone(value: number) {
  if (value < 300) return 'text-emerald-600 dark:text-emerald-400'
  if (value < 900) return 'text-amber-600 dark:text-amber-400'
  return 'text-red-600 dark:text-red-400'
}
