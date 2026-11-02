import { ChevronDown, ChevronRight, Loader2, RefreshCw } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { explorerApi, type Attribute, type NodeView } from '../../../api/tools';
import { EXPLORER_COPY } from '../../../copy/tools';
import { hint } from '../../../copy/hints';
import { KINDS, countKinds, type Kind } from '../../../points/pointKind';
import { useListFilter } from '../../../state/useListFilter';
import { Disclosure } from '../../../ui/Disclosure';
import { FilterBar } from '../../../ui/FilterBar';
import { isFiltered } from '../../../ui/filterState';
import { HelpTip } from '../../../ui/HelpTip';
import { InfoTip } from '../../../ui/InfoTip';
import { KindBadge } from '../../../ui/KindBadge';
import { objectKinds } from './attributeKind';
import { CommandPanel } from './CommandPanel';
import { attributeShown, NODE_FILTER } from './explorerFilters';
import { ObjectTable } from './ObjectTable';
import { useAutoRefresh, useNodeReads } from './useNodeReads';

/** The kinds of point a data object holds (quality and time helpers left out), in the usual order. */
const summaryOf = (kinds: Kind[]) => KINDS.filter(kind => kind !== 'AT' && kinds.includes(kind));

/** One logical node: data objects (read when opened, then every 2 s while open), datasets and report
 *  blocks. A filter bar above the objects searches them and picks one kind of point; objects with
 *  nothing left disappear while it is on. "Read all now" reads every open object at once. */
export function NodeDetail({ reference, host, watched, addWatch, fail }: {
  reference: string; host: string; watched: Set<string>; addWatch: (items: Attribute[]) => void; fail: (message: string) => void;
}) {
  const [node, setNode] = useState<NodeView | null>(null);
  const [open, setOpen] = useState<Set<string>>(new Set());
  const { values, reading, paused, read } = useNodeReads(reference, fail);
  const filter = useListFilter('explorer-node', NODE_FILTER);
  useEffect(() => {
    setNode(null); setOpen(new Set());
    explorerApi.node(reference).then(setNode).catch(error => fail(error.message));
  }, [reference]);
  const view = useMemo(() => (node?.objects ?? []).map(object => {
    const kinds = objectKinds(object.attributes);
    const shown = object.attributes.map((attribute, index) => attributeShown(object.name, attribute, kinds[index], filter.values));
    return { object, kinds, shown, count: shown.filter(Boolean).length };
  }), [node, filter.values]);
  const counts = useMemo(() => ({ kind: countKinds(view.flatMap(item => item.kinds), kind => kind) }), [view]);
  const openGroups = () => (node?.objects ?? []).filter(object => open.has(object.name)).map(object => object.attributes);
  useAutoRefresh(() => read('', openGroups(), true), open.size > 0 && !paused);
  const toggle = (name: string, attributes: Attribute[]) => {
    const next = new Set(open);
    if (next.has(name)) next.delete(name); else { next.add(name); read(name, [attributes]); }
    setOpen(next);
  };
  if (!node) return <div className="card explorer-detail fill-grow"><div className="guide-empty">
    <Loader2 size={22} className="spin" /><p>Reading {reference}…</p></div></div>;
  const filtered = isFiltered(NODE_FILTER, filter.values);
  const total = view.reduce((sum, item) => sum + item.kinds.length, 0);
  const matching = view.reduce((sum, item) => sum + item.count, 0);
  return <div className="card explorer-detail fill-grow fill-panel">
    <div className="panel-head"><h3>{node.ref}</h3>
      <span className="small muted">{node.objects.length} data objects · {node.leaf_count} values</span>
      <HelpTip text={hint('tools.explorer.objects')} label="How do I read values?" />
      <button className="ghost explorer-read-all" disabled={!open.size || reading === '*'} title={EXPLORER_COPY.readAllHint}
        onClick={() => read('*', openGroups())}>
        {reading === '*' ? <Loader2 size={15} className="spin" /> : <RefreshCw size={15} />}{EXPLORER_COPY.readAll}</button></div>
    {paused && <p className="small muted explorer-paused" role="status">{EXPLORER_COPY.autoPaused}</p>}
    <FilterBar label={EXPLORER_COPY.filterAttributes} spec={NODE_FILTER} values={filter.values} set={filter.set}
      clear={filter.clear} shown={matching} total={total} counts={counts} />
    <div className="panel-body fill-scroll explorer-detail-body"><div className="object-list">{view.map(({ object, kinds, shown, count }) => {
      if (filtered && count === 0) return null;
      const isOpen = open.has(object.name);
      return <div key={object.name} className="object">
        <div className="object-row">
          <button className="ghost object-toggle" aria-expanded={isOpen} onClick={() => toggle(object.name, object.attributes)}>
            {isOpen ? <ChevronDown size={15} /> : <ChevronRight size={15} />}<strong>{object.name}</strong>
            <span className="fc-list">{object.fcs.map(fc => <span key={fc} className="fc-badge">{fc}</span>)}</span>
            <span className="object-kinds">{summaryOf(kinds).map(kind => <KindBadge key={kind} kind={kind} />)}</span>
          </button>
        </div>
        {isOpen && <CommandPanel reference={`${node.ref}.${object.name}`} name={object.name} host={host}
          attributes={object.attributes} values={values} fail={fail} operated={() => read(object.name, [object.attributes])} />}
        {isOpen && <ObjectTable attributes={object.attributes} kinds={kinds} shown={shown} values={values}
          reading={reading === object.name || reading === '*'} watched={watched} addWatch={addWatch} />}
      </div>;
    })}</div>
    {filtered && matching === 0 && <p className="muted watch-empty">{EXPLORER_COPY.noAttributes}</p>}
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
  </div></div>;
}
