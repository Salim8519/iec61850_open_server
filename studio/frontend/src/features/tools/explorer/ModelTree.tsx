import { ChevronDown, ChevronRight, Cpu, Loader2, Search } from 'lucide-react';
import { useEffect, useState } from 'react';
import { explorerApi } from '../../../api/tools';
import { EXPLORER_COPY } from '../../../copy/tools';
import { InfoTip } from '../../../ui/InfoTip';

/** Logical devices and their logical nodes; each device loads its nodes only when opened. */
export function ModelTree({ selected, select, fail }: {
  selected: string; select: (ref: string) => void; fail: (message: string) => void;
}) {
  const [devices, setDevices] = useState<string[] | null>(null);
  const [open, setOpen] = useState<Record<string, string[] | 'loading'>>({});
  const [filter, setFilter] = useState('');
  useEffect(() => {
    explorerApi.devices().then(value => {
      setDevices(value.names);
      if (value.names.length === 1) toggle(value.names[0]);
    }).catch(error => fail(error.message));
  }, []);
  const toggle = (device: string) => {
    if (open[device]) { setOpen(({ [device]: _, ...rest }) => rest); return; }
    setOpen(current => ({ ...current, [device]: 'loading' }));
    explorerApi.nodes(device).then(value => setOpen(current => ({ ...current, [device]: value.names })))
      .catch(error => { fail(error.message); setOpen(({ [device]: _, ...rest }) => rest); });
  };
  const needle = filter.trim().toLowerCase();
  return <div className="card explorer-tree">
    <div className="card-heading"><h3><InfoTip term="logical_device" label={EXPLORER_COPY.treeTitle} /></h3></div>
    <div className="guide-section">
      <label className="search-field"><Search size={16} />
        <input value={filter} onChange={event => setFilter(event.target.value)} placeholder={EXPLORER_COPY.filter}
          aria-label={EXPLORER_COPY.filter} /></label>
    </div>
    <ul className="tree" role="tree" aria-label={EXPLORER_COPY.treeTitle}>
      {devices === null && <li className="muted tree-loading"><Loader2 size={15} className="spin" />Loading…</li>}
      {devices?.map(device => {
        const nodes = open[device];
        const shown = Array.isArray(nodes) ? nodes.filter(name => !needle || name.toLowerCase().includes(needle)) : [];
        return <li key={device} role="treeitem" aria-expanded={!!nodes}>
          <button className="tree-row device" onClick={() => toggle(device)}>
            {nodes ? <ChevronDown size={15} /> : <ChevronRight size={15} />}<Cpu size={15} /><span>{device}</span>
          </button>
          {nodes === 'loading' && <div className="muted tree-loading"><Loader2 size={15} className="spin" />Loading…</div>}
          {Array.isArray(nodes) && <ul role="group">{shown.map(name => {
            const ref = `${device}/${name}`;
            return <li key={ref} role="treeitem" aria-selected={selected === ref}>
              <button className={`tree-row node${selected === ref ? ' active' : ''}`} onClick={() => select(ref)}>
                <span>{name}</span></button></li>;
          })}{shown.length === 0 && <li className="small muted tree-loading">No logical node matches.</li>}</ul>}
        </li>;
      })}
    </ul>
  </div>;
}
