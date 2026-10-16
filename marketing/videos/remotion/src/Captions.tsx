// Captions as a floating pill: slide up on start, fade out at the end. English, no audio.
import { AbsoluteFill, interpolate, useCurrentFrame, useVideoConfig } from 'remotion';
import { COLORS, SANS, type Caption } from './brand';

export function Captions({ captions }: { captions: Caption[] }) {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const now = frame / fps;
  const active = captions.find((caption) => now >= caption.start && now < caption.end);
  if (!active) return null;
  const enter = interpolate(now - active.start, [0, 0.3], [0, 1], { extrapolateRight: 'clamp' });
  const leave = interpolate(active.end - now, [0, 0.25], [0, 1], { extrapolateRight: 'clamp' });
  const shown = Math.min(enter, leave);
  return (
    <AbsoluteFill style={{ justifyContent: 'flex-end', alignItems: 'center', paddingBottom: 64 }}>
      <div style={{ opacity: shown, transform: `translateY(${(1 - enter) * 24}px)`, maxWidth: 1500,
        display: 'flex', alignItems: 'center', gap: 18, padding: '20px 34px', borderRadius: 22,
        background: 'rgb(7 13 26 / 86%)', boxShadow: '0 20px 60px rgb(0 0 0 / 35%)',
        border: '1px solid rgb(255 255 255 / 10%)', color: COLORS.ink, fontFamily: SANS,
        fontSize: 42, fontWeight: 560, lineHeight: 1.3, letterSpacing: '-0.01em' }}>
        <span style={{ width: 10, height: 10, borderRadius: 5, background: COLORS.accent, flex: 'none' }} />
        <span>{active.text}</span>
      </div>
    </AbsoluteFill>
  );
}
