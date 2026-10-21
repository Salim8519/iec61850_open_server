import { Plus } from 'lucide-react';
import type { Attribute, ReadValue } from '../../../api/tools';
import type { Kind } from '../../../points/pointKind';
import { InfoTip } from '../../../ui/InfoTip';
import { KindBadge, kindRow } from '../../../ui/KindBadge';
import { ValueText } from './ValueText';

/** The attributes of one opened data object, one line each: the type badge (what kind of point it is), the FC
 *  as a small tag, the value read, and the + that adds it to the watch list. `shown` follows the filter. */
export function ObjectTable({ attributes, kinds, shown, values, reading, watched, addWatch }: {
  attributes: Attribute[]; kinds: Kind[]; shown: boolean[]; values: Record<string, ReadValue>; reading: boolean;
  watched: Set<string>; addWatch: (items: Attribute[]) => void;
}) {
  return <table className="attribute-table compact-rows"><thead><tr><th>Attribute</th><th>Type</th>
    <th><InfoTip term="fc" label="FC" /></th><th>Value</th><th aria-label="Watch" /></tr></thead>
    <tbody>{attributes.map((attribute, index) => {
      if (!shown[index]) return null;
      const value = values[attribute.ref];
      const key = `${attribute.ref}|${attribute.fc}`;
      return <tr key={key} {...kindRow(kinds[index])}>
        <td className="mono">{attribute.path}</td>
        <td><KindBadge kind={kinds[index]} raw={attribute.type} /></td>
        <td><span className="fc-badge">{attribute.fc}</span></td>
        <td>{value ? <ValueText text={value.text} status={value.status} />
          : <span className="small muted">{reading ? 'Reading…' : ''}</span>}</td>
        <td><button className="ghost icon-button" disabled={watched.has(key)} onClick={() => addWatch([attribute])}
          title={watched.has(key) ? 'Already watched' : 'Add to watch list'} aria-label={`Watch ${attribute.path}`}>
          <Plus size={15} /></button></td></tr>;
    })}</tbody></table>;
}
