import { Download, Search, Share2 } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { api } from '../../api/client';
import { Disclosure } from '../../ui/Disclosure';
import { EmptyState } from '../../ui/EmptyState';
import { Hint } from '../../ui/Hint';
import { SegmentedControl } from '../../ui/SegmentedControl';
import { flowRows, LINK_STYLE, linkState, MANY_IEDS, wireVerdicts, type FlowEdge, type FlowGraph, type FlowType } from './signalFlow';
import './signal-flow.css';

type Source = { id: string; filename: string; ieds: string[] };

/** Who sends GOOSE and reports to whom, as declared in the SCD. Station-wide or for one IED. */
export function SignalFlowView({ iedName, running }: { iedName?: string; running: boolean }) {
  const [sources, setSources] = useState<Source[]>([]);
  const [source, setSource] = useState('');
  const [focus, setFocus] = useState(iedName ?? '');
  const [graph, setGraph] = useState<FlowGraph | null>(null);
  const [type, setType] = useState<FlowType | 'all'>('all');
  const [query, setQuery] = useState('');
  const [selected, setSelected] = useState<FlowEdge | null>(null);
  const [error, setError] = useState('');
  const [live, setLive] = useState<Record<string, { state: string; checked_at: string }>>({});
  useEffect(() => {
    if (!running) { setLive({}); return; }
    const load = () => api<{ ieds: { checks: { area: string; item: string; message: string; observed_at: string | null }[] }[] }>('/health')
      .then(value => setLive(wireVerdicts(value.ieds))).catch(() => {});
    load();
    const timer = setInterval(load, 3000);
    return () => clearInterval(timer);
  }, [running]);
  useEffect(() => {
    api<{ sources: Source[] }>('/signal-flow/sources').then(value => {
      setSources(value.sources);
      setSource(value.sources.find(item => !iedName || item.ieds.includes(iedName))?.id ?? '');
    }).catch(reason => setError(reason.message));
  }, [iedName]);
  useEffect(() => { setFocus(iedName ?? ''); }, [iedName]);
  useEffect(() => {
    if (!source) return;
    setSelected(null); setError('');
    const filter = focus ? `&ied=${encodeURIComponent(focus)}` : '';
    api<FlowGraph>(`/signal-flow?source=${source}${filter}`).then(value => {
      setGraph(value);
      if (!iedName && !focus && value.ieds.length > MANY_IEDS) setFocus(value.ieds[0].name);
    }).catch(reason => setError(reason.message));
  }, [source, focus, iedName]);
  const rows = useMemo(() => graph ? flowRows(graph, type, query) : [], [graph, type, query]);
  if (!graph) return <div className={iedName ? '' : 'card'}><EmptyState icon={<Share2 size={22} />}
    title="Signal flow" hintId={error ? 'flow.error' : 'flow.intro'} />{error && <p role="alert" className="import-unavailable">{error}</p>}</div>;
  const problems = graph.diagnostics.filter(item => item.level === 'problem');
  const notes = graph.diagnostics.filter(item => item.level === 'info');
  const names = Object.fromEntries(graph.ieds.map(item => [item.name, item]));
  const receiver = (edge: FlowEdge) => <button key={edge.id} type="button"
    className={`flow-chip line-${LINK_STYLE[edge.type].line}${selected?.id === edge.id ? ' active' : ''}`}
    onClick={() => setSelected(selected?.id === edge.id ? null : edge)}>
    {edge.subscriber}{names[edge.subscriber]?.loaded ? '' : ' · in SCD only'}</button>;
  return <div className={iedName ? 'flow' : 'card flow'}>
    {!iedName && <div className="card-heading"><h2>Signal flow</h2></div>}
    <div className="guide-section">
      <Hint id="flow.intro" />
      <div className="facts">
        <div className="fact"><span>IEDs</span><strong>{graph.summary.ieds}</strong></div>
        <div className="fact"><span>GOOSE links</span><strong>{graph.summary.goose_links}</strong></div>
        <div className="fact"><span>Report links</span><strong>{graph.summary.report_links}</strong></div>
        <div className="fact"><span>Problems in the file</span><strong>{graph.summary.problems}</strong></div>
      </div>
      <div className="row flow-filters">
        {sources.length > 1 && <label>File<select value={source} onChange={event => setSource(event.target.value)}>
          {sources.map(item => <option key={item.id} value={item.id}>{item.filename}</option>)}</select></label>}
        {!iedName && <label>IED<select value={focus} onChange={event => setFocus(event.target.value)}>
          {graph.ieds.length <= MANY_IEDS && <option value="">All IEDs</option>}
          {(focus && !graph.ieds.some(item => item.name === focus) ? [{ name: focus }] : graph.ieds).map(item =>
            <option key={item.name} value={item.name}>{item.name}</option>)}</select></label>}
        <SegmentedControl label="Link type" value={type} onChange={setType} options={[{ value: 'all', label: 'All' },
          { value: 'goose', label: 'GOOSE' }, { value: 'report', label: 'Reports' }]} />
        <div className="search-field"><Search size={16} /><input aria-label="Search IED or signal" placeholder="Search IED or signal"
          value={query} onChange={event => setQuery(event.target.value)} /></div>
      </div>
    </div>
    {selected && <div className="guide-section flow-detail">
      <div className="card-heading"><h3>{selected.publisher} → {selected.subscriber} · {LINK_STYLE[selected.type].label}</h3>
        <span className="row">
        <a className="button" href={`/api/v1/signal-flow/edges/${selected.id}.csv?source=${graph.source}`}>
          <Download size={14} />Download CSV</a>
        <button type="button" className="ghost" onClick={() => setSelected(null)}>Close</button></span></div>
      <p className="small muted">{linkState(selected, selected.type === 'goose' ? live[graph.blocks.find(block => block.id === selected.block)?.ref ?? ''] : undefined)} · {selected.match === 'named'
        ? 'The receiver names this message block.' : 'Matched by data: the receiver names the signals, not the block.'}</p>
      <table><thead><tr><th>Sent value</th><th>Receiver input</th><th>Description</th></tr></thead>
        <tbody>{selected.signals.map(signal => <tr key={signal.source + signal.int_addr}>
          <td className="reference-cell"><code>{signal.source}</code></td><td>{signal.int_addr || '—'}</td>
          <td className="muted">{signal.desc || '—'}</td></tr>)}</tbody></table>
    </div>}
    <div className="table-scroll"><table><thead><tr><th>Sender</th><th>Message</th><th>Receivers (declared in SCD)</th></tr></thead>
      <tbody>{rows.map(({ block, edges }) => <tr key={block.id}>
        <td><strong>{block.ied}</strong>{names[block.ied] && !names[block.ied].loaded && <div className="small muted">in SCD only</div>}</td>
        <td><span className={`badge flow-type line-${LINK_STYLE[block.type].line}`} title={LINK_STYLE[block.type].hint}>
          {LINK_STYLE[block.type].label}</span> <code title={block.ref}>{block.name}</code>
          <div className="small muted">Dataset {block.dataset || '—'} · {block.members.length} values</div></td>
        <td><div className="flow-chips">{edges.length ? edges.map(receiver) :
          <span className="small muted">No receiver in this file</span>}</div></td></tr>)}</tbody></table>
      {!rows.length && <EmptyState icon={<Share2 size={20} />} title="No links to show" hintId="flow.empty" />}
    </div>
    {problems.length > 0 && <div className="guide-section"><Disclosure title={`Problems in the file (${problems.length})`}>
      <Hint id="flow.problems" compact /><ul className="deletion-impact">{problems.map(item =>
        <li key={item.ied + item.message}><strong>{item.ied}</strong>: {item.message}</li>)}</ul></Disclosure></div>}
    {notes.length > 0 && <div className="guide-section"><Disclosure title={`Notes (${notes.length})`}>
      <ul className="deletion-impact">{notes.map(item => <li key={item.ied + item.message}><strong>{item.ied}</strong>: {item.message}</li>)}</ul>
    </Disclosure></div>}
  </div>;
}
