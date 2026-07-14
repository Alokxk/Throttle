import { useState, type FormEvent } from 'react'
import { api, ApiError } from '../lib/api'
import { Logo } from '../components/Logo'

interface LoginProps {
  onLogin: (apiKey: string) => void
}

export function Login({ onLogin }: LoginProps) {
  const [apiKey, setApiKey] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      await api.me(apiKey.trim())
      onLogin(apiKey.trim())
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not reach the API')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-neutral-50 px-4 dark:bg-neutral-950">
      <form
        onSubmit={handleSubmit}
        className="w-full max-w-sm rounded-2xl border border-neutral-200 bg-white p-7 shadow-lg shadow-neutral-900/5 dark:border-neutral-800 dark:bg-neutral-900"
      >
        <Logo className="h-9 w-9" />
        <h1 className="mt-4 text-lg font-semibold text-neutral-900 dark:text-neutral-100">
          Throttle
        </h1>
        <p className="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
          Enter your API key to view usage, rules, and exemptions.
        </p>

        <input
          type="text"
          value={apiKey}
          onChange={(e) => setApiKey(e.target.value)}
          placeholder="thr_..."
          autoFocus
          className="mt-5 w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm font-mono outline-none transition-colors focus:border-teal-500 focus:ring-2 focus:ring-teal-500/20 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-100"
        />

        {error && <p className="mt-2 text-sm text-red-600 dark:text-red-400">{error}</p>}

        <button
          type="submit"
          disabled={!apiKey.trim() || loading}
          className="mt-4 w-full rounded-lg bg-teal-500 px-3 py-2 text-sm font-semibold text-neutral-900 transition-colors hover:bg-teal-400 disabled:opacity-50 disabled:hover:bg-teal-500"
        >
          {loading ? 'Checking...' : 'Continue'}
        </button>

        <p className="mt-4 text-xs text-neutral-400">
          No key yet? Register with{' '}
          <code className="rounded bg-neutral-100 px-1 py-0.5 dark:bg-neutral-800">
            POST /register
          </code>
        </p>
      </form>
    </div>
  )
}
