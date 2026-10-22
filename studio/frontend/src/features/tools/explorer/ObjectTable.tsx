import { Plus } from 'lucide-react';
import type { Attribute, ReadValue } from '../../../api/tools';
import type { Kind } from '../../../points/pointKind';
import { InfoTip } from '../../../ui/InfoTip';
import { KindBadge, kindRow } from '../../../ui/KindBadge';
import { isCommandField, READ_SAFETY_COPY, selectsOnRead } from './readSafety';
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
      const sbo = selectsOnRead(attribute);  // reading it would select the control (change 0177)
      return <tr key={key} {...kindRow(kinds[index])}>
        <td className="mono">{attribute.path}</td>
        <td><KindBadge kind={kinds[index]} raw={attribute.type} /></td>
        <td><span className="fc-badge">{attribute.fc}</span></td>
        <td>{sbo ? <ValueText text={READ_SAFETY_COPY.sbo} status="warning" />
          : value ? <ValueText text={value.text} status={value.status} />
          : isCommandField(attribute) ? <span className="small muted" title={READ_SAFETY_COPY.commandHint}>{READ_SAFETY_COPY.command}</span>
          : <span className="small muted">{reading ? 'Reading…' : ''}</span>}</td>
        <td><button className="ghost icon-button" disabled={sbo || watched.has(key)} onClick={() => addWatch([attribute])}
          title={sbo ? READ_SAFETY_COPY.sboWatch : watched.has(key) ? 'Already watched' : 'Add to watch list'}
          aria-label={`Watch ${attribute.path}`}><Plus size={15} /></button></td></tr>;
    })}</tbody></table>;
}
