// The spoken narration: one voice clip per caption, starting with its caption.
import { Audio, Sequence, staticFile, useVideoConfig } from 'remotion';
import type { Caption } from './brand';

export function Narration({ captions }: { captions: Caption[] }) {
  const { fps } = useVideoConfig();
  return (
    <>
      {captions.filter((caption) => caption.voice).map((caption) => (
        <Sequence key={`${caption.voice}-${caption.start}`} from={Math.round(caption.start * fps)} layout="none">
          <Audio src={staticFile(`voice/${caption.voice}.wav`)} />
        </Sequence>
      ))}
    </>
  );
}
