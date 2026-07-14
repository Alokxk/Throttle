import { useEffect, useRef, useState } from 'react'
import { api, type Client, type Stats } from '../lib/api'
import { StatTile } from '../components/StatTile'
import { Sparkline } from '../components/Sparkline'
import { HashIcon, CheckIcon, XIcon, PercentIcon } from '../components/icons'

const POLL_INTERVAL_MS = 3000
const SPARKLINE_POINTS = 20

interface OverviewProps {
  apiKey: string
  client: Client
}

export function Overview({ apiKey, client }: OverviewProps) {
  const [stats, setStats] = useState<Stats | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [rps, setRps] = useState<number[]>(Array(SPARKLINE_POINTS).fill(0))
  const prevTotal = useRef<number | null>(null)
  const prevTime = useRef<number>(Date.now())

  useEffect(() => {
    let cancelled = false

    async function poll() {
      try {
        const s = await api.stats(apiKey, client.client_id)
        if (cancelled) return

        const now = Date.now()
        if (prevTotal.current !== null) {
          const deltaChecks = s.total_checks - prevTotal.current
          const deltaSeconds = (now - prevTime.current) / 1000
          const rate = deltaSeconds > 0 ? Math.max(0, deltaChecks / deltaSeconds) : 0
          setRps((prev) => [...prev.slice(1), rate])
        }
        prevTotal.current = s.total_checks
        prevTime.current = now

        setStats(s)
        setError(null)
      } catch {
        if (!cancelled) setError('Lost connection to the API')
      }
    }

    poll()
    const id = setInterval(poll, POLL_INTERVAL_MS)
    return () => {
      cancelled = true
      clearInterval(id)
    }
  }, [apiKey, client.client_id])

  const rejectionRate = stats && stats.total_checks > 0 ? (stats.rejected / stats.total_checks) * 100 : 0

  return (
    <div className="space-y-5">
      {error && (
        <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-950 dark:text-red-400">
          {error}
        </p>
      )}

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <StatTile label="Total checks" value={stats?.total_checks ?? null} icon={<HashIcon />} />
        <StatTile
          label="Allowed"
          value={stats?.allowed ?? null}
          icon={<CheckIcon />}
          accent="green"
        />
        <StatTile
          label="Rejected"
          value={stats?.rejected ?? null}
          icon={<XIcon />}
          accent="red"
        />
        <StatTile
          label="Rejection rate"
          value={stats ? `${rejectionRate.toFixed(1)}%` : null}
          icon={<PercentIcon />}
        />
      </div>

      <div className="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm shadow-neutral-900/[0.02] dark:border-neutral-800 dark:bg-neutral-900">
        <div className="flex items-baseline justify-between">
          <p className="text-xs font-medium text-neutral-500 dark:text-neutral-400">
            Requests/sec (last {(SPARKLINE_POINTS * POLL_INTERVAL_MS) / 1000}s, polled every{' '}
            {POLL_INTERVAL_MS / 1000}s)
          </p>
          <p className="text-sm font-semibold tabular-nums text-neutral-900 dark:text-neutral-100">
            {rps[rps.length - 1]?.toFixed(1) ?? '0.0'}/s
          </p>
        </div>
        <div className="mt-3">
          <Sparkline values={rps} />
        </div>
      </div>

      <div className="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm shadow-neutral-900/[0.02] dark:border-neutral-800 dark:bg-neutral-900">
        <p className="text-xs font-medium text-neutral-500 dark:text-neutral-400">By algorithm</p>
        <div className="mt-3 space-y-3">
          {stats &&
            Object.entries(stats.by_algorithm).map(([algo, count]) => {
              const pct = stats.total_checks > 0 ? (count / stats.total_checks) * 100 : 0
              return (
                <div key={algo}>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-neutral-600 dark:text-neutral-400">{algo}</span>
                    <span className="font-medium tabular-nums text-neutral-900 dark:text-neutral-100">
                      {count}
                    </span>
                  </div>
                  <div className="mt-1 h-1.5 overflow-hidden rounded-full bg-neutral-100 dark:bg-neutral-800">
                    <div
                      className="h-full rounded-full bg-teal-500 transition-all duration-300"
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                </div>
              )
            })}
        </div>
      </div>
    </div>
  )
}
