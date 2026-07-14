interface SparklineProps {
  values: number[]
}

// Deliberately hand-rolled instead of pulling in a charting library — this
// dashboard only ever needs one shape (a rolling bar sparkline), which is a
// few divs, not a dependency.
export function Sparkline({ values }: SparklineProps) {
  const max = Math.max(1, ...values)

  return (
    <div className="flex h-16 items-end gap-1">
      {values.map((v, i) => (
        <div
          key={i}
          className="flex-1 rounded-t-sm bg-gradient-to-t from-teal-500 to-teal-400 transition-all duration-300 dark:from-teal-600 dark:to-teal-400"
          style={{
            height: `${Math.max(3, (v / max) * 100)}%`,
            opacity: 0.25 + (i / values.length) * 0.75,
          }}
        />
      ))}
    </div>
  )
}
