import type { PlacedTag } from '../../../equipment/signalLayout';

export type TagProblem = { text: string; tone: 'problem' | 'waiting' };

/** A small grey IED name beside its equipment, always the full IED name (change 0097).
 * A GOOSE problem adds one small ⚠ only; the words are in the hover text and the side sheet.
 * Readers show as "◉ 2 read" with details on hover. */
export function IedTag({ tag, scale, problems, readers, readersTitle, operated, stopped, selected, select }: {
  tag: PlacedTag; scale: number; problems: TagProblem[]; readers: string; readersTitle: string;
  operated?: { key: number; label: string }; stopped: boolean; selected: boolean; select: () => void;
}) {
  const tone = problems.some(item => item.tone === 'problem') ? ' problem' : problems.length ? ' waiting' : '';
  const words = problems.map(item => item.text).join('\n');
  return <button className={`ied-tag nodrag nopan${tone}${stopped ? ' stopped' : ''}${selected ? ' selected' : ''}`}
    style={{ transform: `translate(${tag.x}px, ${tag.y}px) scale(${scale})` }} onClick={select}
    title={[tag.name, stopped ? 'IED not running' : '', words, readersTitle, 'Click for its GOOSE links'].filter(Boolean).join('\n')}
    aria-label={`IED ${tag.name}${stopped ? ', not running' : ''}${problems.length ? `, ${words.replace(/\n/g, ', ')}` : ''}`}>
    <span className="ied-tag-line">
      <span className="ied-tag-name">{tag.name}</span>
      {problems.length > 0 && <span className={`ied-tag-warn${tone}`} aria-hidden>⚠{problems.length > 1 ? problems.length : ''}</span>}
      {stopped && <span className="ied-tag-note">not running</span>}
      {readers && <span className="ied-tag-readers">{readers}</span>}
    </span>
    {operated && <span key={operated.key} className="ied-tag-operated">{operated.label}</span>}
  </button>;
}
