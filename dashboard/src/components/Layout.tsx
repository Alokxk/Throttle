import type { Client } from '../lib/api'
import { Logo } from './Logo'

export type Tab = 'overview' | 'rules' | 'exemptions'

interface LayoutProps {
  client: Client
  tab: Tab
  onTabChange: (tab: Tab) => void
  onLogout: () => void
  children: React.ReactNode
}

const TABS: { id: Tab; label: string }[] = [
  { id: 'overview', label: 'Overview' },
  { id: 'rules', label: 'Rules' },
  { id: 'exemptions', label: 'Exemptions' },
]

export function Layout({ client, tab, onTabChange, onLogout, children }: LayoutProps) {
  return (
    <div className="min-h-screen bg-neutral-50 dark:bg-neutral-950">
      <header className="border-b border-neutral-200 bg-white dark:border-neutral-800 dark:bg-neutral-900">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-3">
          <div className="flex items-center gap-2.5">
            <Logo />
            <div>
              <h1 className="text-sm font-semibold text-neutral-900 dark:text-neutral-100">
                Throttle
              </h1>
              <p className="text-xs text-neutral-500 dark:text-neutral-400">{client.name}</p>
            </div>
          </div>
          <button
            onClick={onLogout}
            className="text-xs font-medium text-neutral-500 transition-colors hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-neutral-100"
          >
            Sign out
          </button>
        </div>
        <nav className="mx-auto flex max-w-5xl gap-1 px-4">
          {TABS.map((t) => (
            <button
              key={t.id}
              onClick={() => onTabChange(t.id)}
              className={`border-b-2 px-3 py-2.5 text-sm font-medium transition-colors ${
                tab === t.id
                  ? 'border-teal-500 text-neutral-900 dark:text-neutral-100'
                  : 'border-transparent text-neutral-500 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-neutral-100'
              }`}
            >
              {t.label}
            </button>
          ))}
        </nav>
      </header>
      <main className="mx-auto max-w-5xl px-4 py-6">{children}</main>
    </div>
  )
}
