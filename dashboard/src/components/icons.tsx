const common = { width: 16, height: 16, viewBox: '0 0 16 16', fill: 'none', stroke: 'currentColor' }

export function HashIcon() {
  return (
    <svg {...common}>
      <path d="M6 2 4.5 14M11.5 2 10 14M2.5 5.5h11M2 10.5h11" strokeWidth={1.5} strokeLinecap="round" />
    </svg>
  )
}

export function CheckIcon() {
  return (
    <svg {...common}>
      <path d="M3 8.5 6.5 12 13 4.5" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

export function XIcon() {
  return (
    <svg {...common}>
      <path d="M4 4l8 8M12 4l-8 8" strokeWidth={1.5} strokeLinecap="round" />
    </svg>
  )
}

export function PercentIcon() {
  return (
    <svg {...common}>
      <circle cx="4.5" cy="4.5" r="1.75" strokeWidth={1.5} />
      <circle cx="11.5" cy="11.5" r="1.75" strokeWidth={1.5} />
      <path d="M12.5 3.5 3.5 12.5" strokeWidth={1.5} strokeLinecap="round" />
    </svg>
  )
}
