// The real-app recording, scaled to 1080p, with the click-following camera.
import { AbsoluteFill, OffthreadVideo, staticFile, useCurrentFrame, useVideoConfig } from 'remotion';
import { FOOTAGE_WIDTH, WIDTH, type Click } from './brand';
import { camera } from './zoom';

export function Footage({ slug, clicks }: { slug: string; clicks: Click[] }) {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const ratio = WIDTH / FOOTAGE_WIDTH;
  const view = camera(clicks, frame / fps, FOOTAGE_WIDTH, FOOTAGE_WIDTH * 9 / 16);
  return (
    <AbsoluteFill style={{ backgroundColor: '#f5f7fa', overflow: 'hidden' }}>
      <AbsoluteFill style={{ transform: `scale(${view.scale})`,
        transformOrigin: `${view.originX * ratio}px ${view.originY * ratio}px` }}>
        <OffthreadVideo src={staticFile(`footage/${slug}.mp4`)} style={{ width: '100%', height: '100%' }} muted />
      </AbsoluteFill>
    </AbsoluteFill>
  );
}
