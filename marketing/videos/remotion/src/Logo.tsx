// The ScadaBench "S-Line" mark (studio/branding/icon.svg).
export function Logo({ size }: { size: number }) {
  return (
    <svg viewBox="0 0 64 64" width={size} height={size}>
      <defs>
        <linearGradient id="sb-tile" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#16274a" /><stop offset="1" stopColor="#070d1a" />
        </linearGradient>
      </defs>
      <rect x="2" y="2" width="60" height="60" rx="14" fill="url(#sb-tile)" />
      <rect x="2.5" y="2.5" width="59" height="59" rx="13.5" fill="none" stroke="#fff" strokeOpacity=".12" />
      <path d="M45 17H25a7 7 0 0 0-7 7v1a7 7 0 0 0 7 7h14a7 7 0 0 1 7 7v1a7 7 0 0 1-7 7H19" fill="none"
        stroke="#e8eef7" strokeWidth="5.2" strokeLinecap="round" strokeLinejoin="round" />
      <rect x="26.5" y="26.5" width="11" height="11" rx="2.4" fill="#2be4b0" />
      <circle cx="45" cy="17" r="3.6" fill="#2be4b0" />
    </svg>
  );
}
