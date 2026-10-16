// One tutorial: title card, the recording with camera and captions, end card.
import { AbsoluteFill, Sequence, interpolate, useCurrentFrame } from 'remotion';
import { END_SECONDS, FONTS_CSS, FPS, TITLE_SECONDS, type Lesson as LessonData } from './brand';
import { Captions } from './Captions';
import { EndCard, TitleCard } from './Cards';
import { Footage } from './Footage';

const FADE = 12;

function Fade({ length, children }: { length: number; children: React.ReactNode }) {
  const frame = useCurrentFrame();
  const opacity = interpolate(frame, [0, FADE, length - FADE, length], [0, 1, 1, 0], {
    extrapolateLeft: 'clamp', extrapolateRight: 'clamp' });
  return <AbsoluteFill style={{ opacity }}>{children}</AbsoluteFill>;
}

export const lessonFrames = (lesson: LessonData) =>
  Math.round((TITLE_SECONDS + lesson.seconds + END_SECONDS) * FPS);

export function Lesson(lesson: LessonData) {
  const title = Math.round(TITLE_SECONDS * FPS);
  const body = Math.round(lesson.seconds * FPS);
  const end = Math.round(END_SECONDS * FPS);
  return (
    <AbsoluteFill style={{ backgroundColor: '#070d1a' }}>
      <style>{FONTS_CSS}</style>
      <Sequence durationInFrames={title}><Fade length={title}><TitleCard number={lesson.number} title={lesson.title} /></Fade></Sequence>
      <Sequence from={title} durationInFrames={body}>
        <Fade length={body}><Footage slug={lesson.slug} clicks={lesson.clicks} /><Captions captions={lesson.captions} /></Fade>
      </Sequence>
      <Sequence from={title + body} durationInFrames={end}><Fade length={end}><EndCard /></Fade></Sequence>
    </AbsoluteFill>
  );
}
