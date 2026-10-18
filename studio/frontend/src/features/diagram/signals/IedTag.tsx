import type { PlacedTag } from '../../../equipment/signalLayout';

export type TagProblem = { text: string; tone: 'problem' | 'waiting' };

/** A small grey IED name beside its equipment. Problems stay as words with a dashed border;
 * IEDs outside the project are dashed and say so; readers show as "◉ 2 read" with details on hover. */
export function IedTag({ tag, scale, label, problems, readers, readersTitle, operated, selected, select }: {
  tag: PlacedTag; scale: number; label: string; problems: TagProblem[]; readers: string; readersTitle: string;
  operated?: { key: number; label: string }; selected: boolean; select: () => void;
}) {
  const tone = problems.some(item => item.tone === 'problem') ? ' problem' : problems.length ? ' waiting' : '';
  const where = tag.outside ? (tag.remote ? 'not in any project file' : 'not in this project') : '';
  return <button className={`ied-tag nodrag nopan${tag.outside ? ' outside' : ''}${tone}${selected ? ' selected' : ''}`}
    style={{ transform: `translate(${tag.x}px, ${tag.y}px) scale(${scale})` }} onClick={select}
    title={`${tag.name}${where ? ` (${where})` : ''}${readersTitle ? `\n${readersTitle}` : ''}`}
    aria-label={`IED ${tag.name}${where ? `, ${where}` : ''}${problems.length ? `, ${problems.map(item => item.text).join(', ')}` : ''}`}>
    <span className="ied-tag-name">{label}</span>
    {tag.outside && <span className="ied-tag-note">outside project</span>}
    {readers && <span className="ied-tag-readers">{readers}</span>}
    {problems.slice(0, 2).map(item => <span key={item.text} className={`ied-tag-problem ${item.tone}`}>{item.text}</span>)}
    {problems.length > 2 && <span className="ied-tag-problem">+{problems.length - 2} more</span>}
    {operated && <span key={operated.key} className="ied-tag-operated">{operated.label}</span>}
  </button>;
}
