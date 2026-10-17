import { ChevronDown, ChevronRight, Loader2, Plus, RefreshCw } from 'lucide-react';
import { useEffect, useState } from 'react';
import { explorerApi, type Attribute, type NodeView, type ReadValue } from '../../../api/tools';
import { EXPLORER_COPY } from '../../../copy/tools';
import { Disclosure } from '../../../ui/Disclosure';
import { Hint } from '../../../ui/Hint';
import { InfoTip } from '../../../ui/InfoTip';
import { CommandPanel } from './CommandPanel';
import { ValueText } from './ValueText';

/** One logical node: data objects (read when opened), datasets and report blocks. */
export function NodeDetail({ reference, host, watched, addWatch, fail }: {
  reference: string; host: string; watched: Set<string>; addWatch: (items: Attribute[]) => void; fail: (message: string) => void;
}) {
  const [node, setNode] = useState<NodeView | null>(null);
  const [open, setOpen] = useState<Set<string>>(new Set());
  const [values, setValues] = useState<Record<string, ReadValue>>({});
  const [reading, setReading] = useState<string>('');
  useEffect(() => {
    setNode(null); setOpen(new Set()); setValues({});
    explorerApi.node(reference).then(setNode).catch(error => fail(error.message));
  }, [reference]);
  const read = (name: string, attributes: Attribute[]) => {
    setReading(name);
    explorerApi.read(attributes.slice(0, 200)).then(answer => setValues(current => ({ ...current, ...answer.values })))
      .catch(error => fail(error.message)).finally(() => setReading(''));
  };
  const toggle = (name: string, attributes: Attribute[]) => {
    const next = new Set(open);
    if (next.has(name)) next.delete(name); else { next.add(name); read(name, attributes); }
    setOpen(next);
  };
  if (!node) return <div className="card explorer-detail"><div className="guide-empty">
    <Loader2 size={22} className="spin" /><p>Reading {reference}…</p></div></div>;
  return <div className="card explorer-detail">
    <div className="card-heading"><h3>{node.ref}</h3>
      <span className="small muted">{node.objects.length} data objects · {node.leaf_count} values</span></div>
    <div className="guide-section"><Hint id="tools.explorer.objects" compact /></div>
    <div className="object-list">{node.objects.map(object => {
      const isOpen = open.has(object.name);
      return <div key={object.name} className="object">
        <div className="object-row">
          <button className="ghost object-toggle" aria-expanded={isOpen} onClick={() => toggle(object.name, object.attributes)}>
            {isOpen ? <ChevronDown size={15} /> : <ChevronRight size={15} />}<strong>{object.name}</strong>
            <span className="fc-list">{object.fcs.map(fc => <span key={fc} className="fc-badge">{fc}</span>)}</span>
          </button>
          {isOpen && <button className="ghost" title="Read again" aria-label={`Read ${object.name} again`}
            onClick={() => read(object.name, object.attributes)}>
            {reading === object.name ? <Loader2 size={15} className="spin" /> : <RefreshCw size={15} />}</button>}
        </div>
        {isOpen && <CommandPanel reference={`${node.ref}.${object.name}`} host={host} attributes={object.attributes}
          fail={fail} />}
        {isOpen && <table className="attribute-table"><thead><tr><th>Attribute</th>
          <th><InfoTip term="fc" label="FC" /></th><th>Type</th><th>Value</th><th aria-label="Watch" /></tr></thead>
          <tbody>{object.attributes.map(attribute => {
            const value = values[attribute.ref];
            const key = `${attribute.ref}|${attribute.fc}`;
            return <tr key={key}><td className="mono">{attribute.path}</td><td><span className="fc-badge">{attribute.fc}</span></td>
              <td className="small muted">{attribute.type}</td>
              <td>{value ? <ValueText text={value.text} status={value.status} />
                : <span className="small muted">{reading === object.name ? 'Reading…' : ''}</span>}</td>
              <td><button className="ghost icon-button" disabled={watched.has(key)} onClick={() => addWatch([attribute])}
                title={watched.has(key) ? 'Already watched' : 'Add to watch list'} aria-label={`Watch ${attribute.path}`}>
                <Plus size={15} /></button></td></tr>;
          })}</tbody></table>}
      </div>;
    })}</div>
    {node.datasets.length > 0 && <Disclosure title={`${EXPLORER_COPY.datasets} (${node.datasets.length})`}>
      {node.datasets.map(dataset => <div key={dataset.ref} className="block-entry">
        <strong><InfoTip term="dataset" label={dataset.name} /></strong>
        <ul className="small mono">{dataset.members.map(member => <li key={member}>{member}</li>)}</ul></div>)}
    </Disclosure>}
    {node.reports.length > 0 && <Disclosure title={`${EXPLORER_COPY.reports} (${node.reports.length})`}>
      <p className="small muted">{EXPLORER_COPY.reportsHint}</p>
      <ul className="small mono">{node.reports.map(report => <li key={report.ref}>{report.name}
        <span className="muted"> · {report.buffered ? 'buffered' : 'unbuffered'}</span></li>)}</ul>
    </Disclosure>}
  </div>;
}
