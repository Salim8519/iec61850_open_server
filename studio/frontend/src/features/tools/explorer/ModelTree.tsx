import { ChevronDown, ChevronRight, Cpu, Loader2 } from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { explorerApi } from '../../../api/tools';
import { EXPLORER_COPY } from '../../../copy/tools';
import { useListFilter } from '../../../state/useListFilter';
import { FilterBar } from '../../../ui/FilterBar';
import { HelpTip } from '../../../ui/HelpTip';
import { InfoTip } from '../../../ui/InfoTip';
import { SpacerBlock, useVirtualRows } from '../../../ui/VirtualRows';
import { nodeMatches, TREE_FILTER } from './explorerFilters';

const ROW_HEIGHT = 36;
type Row = { key: string; type: 'device'; device: string; open: boolean } | { key: string; type: 'node'; name: string }
  | { key: string; type: 'loading' | 'empty' };

/** Devices with their opened logical nodes (filtered) as one flat list of rows. */
function treeRows(devices: string[], open: Record<string, string[] | 'loading'>, matches: (name: string) => boolean): Row[] {
  const rows: Row[] = [];
  for (const device of devices) {
    const nodes = open[device];
    rows.push({ key: `device:${device}`, type: 'device', device, open: !!nodes });
    if (nodes === 'loading') rows.push({ key: `loading:${device}`, type: 'loading' });
    if (!Array.isArray(nodes)) continue;
    const shown = nodes.filter(matches);
    for (const name of shown) rows.push({ key: `${device}/${name}`, type: 'node', name });
    if (!shown.length) rows.push({ key: `empty:${device}`, type: 'empty' });
  }
  return rows;
}

/** Logical devices and their logical nodes in the left column; each device loads its nodes only when opened. */
export function ModelTree({ selected, select, fail }: {
  selected: string; select: (ref: string) => void; fail: (message: string) => void;
}) {
  const [devices, setDevices] = useState<string[] | null>(null);
  const [open, setOpen] = useState<Record<string, string[] | 'loading'>>({});
  const filter = useListFilter('explorer-tree', TREE_FILTER);
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
  const loaded = Object.values(open).flatMap(nodes => Array.isArray(nodes) ? nodes : []);
  // One flat list of rows, so a device with thousands of logical nodes draws only what is on screen.
  const rows = useMemo(() => treeRows(devices ?? [], open, name => nodeMatches(name, filter.values)), [devices, open, filter.values]);
  const box = useRef<HTMLUListElement>(null);
  const keyOf = useCallback((index: number) => rows[index]?.key ?? index, [rows]);
  const { range, spacer } = useVirtualRows(box, rows.length, keyOf, ROW_HEIGHT);
  return <div className="card fill-panel explorer-tree">
    <div className="panel-head"><h3><InfoTip term="logical_device" label={EXPLORER_COPY.treeTitle} /></h3>
      <HelpTip text={EXPLORER_COPY.treeHint} label="How does the tree load?" /></div>
    <div className="explorer-tree-filter">
      <FilterBar label="Filter logical nodes" spec={TREE_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
        shown={loaded.filter(name => nodeMatches(name, filter.values)).length} total={loaded.length} />
    </div>
    <ul className="tree fill-scroll panel-body" role="tree" aria-label={EXPLORER_COPY.treeTitle} ref={box}>
      {devices === null && <li className="muted tree-loading"><Loader2 size={15} className="spin" />Loading…</li>}
      <SpacerBlock item height={range.before} spacerRef={spacer} />
      {rows.slice(range.start, range.end).map(row => row.type === 'device'
        ? <li key={row.key} data-vrow={row.key} role="treeitem" aria-level={1} aria-expanded={row.open}>
          <button className="tree-row device" onClick={() => toggle(row.device)}>
            {row.open ? <ChevronDown size={15} /> : <ChevronRight size={15} />}<Cpu size={15} /><span>{row.device}</span>
          </button></li>
        : row.type === 'node' ? <li key={row.key} data-vrow={row.key} role="treeitem" aria-level={2} aria-selected={selected === row.key}
          className="tree-child">
          <button className={`tree-row node${selected === row.key ? ' active' : ''}`} onClick={() => select(row.key)}>
            <span>{row.name}</span></button></li>
        : <li key={row.key} data-vrow={row.key} className={`muted tree-loading tree-child${row.type === 'empty' ? ' small' : ''}`}>
          {row.type === 'loading' ? <><Loader2 size={15} className="spin" />Loading…</> : 'No logical node matches.'}</li>)}
      <SpacerBlock item height={range.after} />
    </ul>
  </div>;
}
