// Title and end cards: the dark brand ground of scadabench.com with a gentle entrance.
import { AbsoluteFill, interpolate, spring, useCurrentFrame, useVideoConfig } from 'remotion';
import { COLORS, MONO, SANS } from './brand';
import { Logo } from './Logo';

const ground = `radial-gradient(1100px 620px at 82% -8%, rgb(43 228 176 / 16%), transparent 70%),
  radial-gradient(900px 560px at -8% 40%, rgb(56 189 248 / 9%), transparent 70%), ${COLORS.night}`;

function useRise(delay: number) {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const progress = spring({ frame: frame - delay, fps, config: { damping: 200 } });
  return { opacity: progress, transform: `translateY(${interpolate(progress, [0, 1], [36, 0])}px)` };
}

function Frame({ children }: { children: React.ReactNode }) {
  return (
    <AbsoluteFill style={{ background: ground, color: COLORS.ink, fontFamily: SANS, padding: '0 180px', justifyContent: 'center' }}>
      <div style={{ position: 'absolute', top: 90, left: 180, display: 'flex', alignItems: 'center', gap: 18,
        fontSize: 40, fontWeight: 650, letterSpacing: '-0.02em', ...useRise(0) }}>
        <Logo size={64} /><span>Scada<b style={{ color: COLORS.accent, fontWeight: 650 }}>Bench</b></span>
      </div>
      <div style={{ display: 'grid', gap: 34 }}>{children}</div>
      <div style={{ position: 'absolute', left: 180, right: 180, bottom: 120, height: 4, borderRadius: 4, opacity: 0.7,
        background: `linear-gradient(90deg, ${COLORS.accent}, ${COLORS.signal} 60%, transparent)` }} />
      <div style={{ position: 'absolute', left: 180, bottom: 60, fontFamily: MONO, fontSize: 28, color: COLORS.muted }}>
        scadabench.com</div>
    </AbsoluteFill>
  );
}

const eyebrow = { fontFamily: MONO, fontSize: 30, letterSpacing: '0.16em', textTransform: 'uppercase' as const, color: COLORS.accent };
const headline = { margin: 0, fontSize: 118, lineHeight: 1.02, letterSpacing: '-0.04em', fontWeight: 650, maxWidth: 1450 };

export function TitleCard({ number, title }: { number: number; title: string }) {
  return (
    <Frame>
      <span style={{ ...eyebrow, ...useRise(6) }}>Lesson {number} · Tutorial</span>
      <h1 style={{ ...headline, ...useRise(10) }}>{title}</h1>
      <p style={{ margin: 0, fontSize: 40, color: COLORS.muted, ...useRise(16) }}>IEC 61850 IED simulator &amp; SCADA test bench</p>
    </Frame>
  );
}

export function EndCard() {
  return (
    <Frame>
      <span style={{ ...eyebrow, ...useRise(4) }}>Try it yourself</span>
      <h1 style={{ ...headline, ...useRise(8) }}>Free for 7 days.<br />No card needed.</h1>
      <p style={{ margin: 0, fontSize: 40, color: COLORS.muted, ...useRise(14) }}>
        Download at scadabench.com · more lessons at scadabench.com/tutorials</p>
    </Frame>
  );
}
