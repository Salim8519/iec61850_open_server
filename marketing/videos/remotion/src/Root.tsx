// One composition per lesson; lessons.json is written by marketing/videos/make.py.
import { Composition } from 'remotion';
import { FPS, HEIGHT, WIDTH, type Lesson as LessonData } from './brand';
import { Lesson, lessonFrames } from './Lesson';
import lessons from './lessons.json';

export function Root() {
  return (
    <>
      {(lessons as LessonData[]).map((lesson) => (
        <Composition key={lesson.slug} id={lesson.slug} component={Lesson} defaultProps={lesson}
          durationInFrames={lessonFrames(lesson)} fps={FPS} width={WIDTH} height={HEIGHT} />
      ))}
    </>
  );
}
