import type { PlacedTag } from '../../../equipment/signalLayout';

export type TagProblem = { text: string; tone: 'problem' | 'waiting' };

/** An IED on the SLD signal layers. Bound to equipment it is one small "G" mark inside that
 * equipment's top-right corner, so it never covers a neighbour (change 0105); unbound it is a
 * full-name tag left of the drawing (change 0097). The full name and every word are on hover.
 * A GOOSE problem shows ⚠ instead of G with a dashed border; readers add "◉n". */
export function IedTag({ tag, scale, problems, readers, readersTitle, operated, received, stopped, selected, select }: {
  tag: PlacedTag; scale: number; problems: TagProblem[]; readers: string; readersTitle: string;
  operated?: { key: number; label: string }; received?: { key: number; label: string };
  stopped: boolean; selected: boolean; select: () => void;
}) {
  const tone = problems.some(item => item.tone === 'problem') ? ' problem' : problems.length ? ' waiting' : '';
  const words = problems.map(item => item.text).join('\n');
  const count = readers.match(/\d+/)?.[0];
  const transform = tag.corner
    ? `translate(${tag.x}px, ${tag.y}px) scale(${scale}) translateX(-100%)`
    : `translate(${tag.x}px, ${tag.y}px) scale(${scale})`;
  return <button className={`ied-tag nodrag nopan${tag.corner ? ' ied-mark' : ''}${tone}${stopped ? ' stopped' : ''}${selected ? ' selected' : ''}`}
    style={{ transform }} onClick={select}
    title={[tag.name, stopped ? 'IED not running' : '', words, readersTitle, 'Click for its GOOSE links'].filter(Boolean).join('\n')}
    aria-label={`IED ${tag.name}${stopped ? ', not running' : ''}${problems.length ? `, ${words.replace(/\n/g, ', ')}` : ''}`}>
    {tag.corner ? <span className="ied-tag-line">
      {problems.length > 0 ? <span className={`ied-tag-warn${tone}`} aria-hidden>⚠{problems.length > 1 ? problems.length : ''}</span>
        : <span className="ied-mark-letter" aria-hidden>G</span>}
      {count && <span className="ied-tag-readers" aria-hidden>◉{count}</span>}
    </span> : <span className="ied-tag-line">
      <span className="ied-tag-name">{tag.name}</span>
      {problems.length > 0 && <span className={`ied-tag-warn${tone}`} aria-hidden>⚠{problems.length > 1 ? problems.length : ''}</span>}
      {stopped && <span className="ied-tag-note">not running</span>}
      {readers && <span className="ied-tag-readers">{readers}</span>}
    </span>}
    {operated && <span key={operated.key} className="ied-tag-operated">{operated.label}</span>}
    {received && <span key={received.key} className="ied-tag-received">{received.label}</span>}
  </button>;
}
