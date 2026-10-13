import { useEffect, useState } from 'react';
import { Box, Plus, Search, Upload, Download } from 'lucide-react';
import type { Ied, Signal } from '../../api/types';
import { api } from '../../api/client';
import { SignalTable } from './SignalTable';
import { DeleteIed } from './DeleteIed';
import { ExportScl } from './ExportScl';
import { BulkDeleteIeds } from './BulkDeleteIeds';

type Props = { ieds: Ied[]; running: boolean; values: Record<string, unknown>;
  addTemplate: (name: string) => void; importFile: () => void;
  prepare: () => Promise<void>;
  perform: (task: () => Promise<unknown>) => Promise<void> };

export function IedWorkspace({ ieds, running, values, addTemplate, importFile, perform, prepare }: Props) {
  const [checked, setChecked] = useState<string[]>([]);
  useEffect(() => { setChecked(current => current.filter(id => ieds.some(item => item.id === id))); }, [ieds]);
  const [selected, setSelected] = useState('');
  const [query, setQuery] = useState('');
  const [fc, setFc] = useState('');
  const [reportFilter, setReportFilter] = useState('');
  const [signals, setSignals] = useState<Signal[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [newName, setNewName] = useState('Feeder01');
  const [adding, setAdding] = useState(false);
  const [error, setError] = useState('');
  useEffect(() => { if (!ieds.some(item => item.id === selected)) setSelected(ieds[0]?.id || ''); }, [ieds, selected]);
  useEffect(() => { setReportFilter(''); }, [selected]);
  useEffect(() => { setOffset(0); }, [query, fc, selected, reportFilter]);
  useEffect(() => {
    if (!selected) { setSignals([]); setTotal(0); return; }
    let cancelled = false;
    const timer = setTimeout(() => {
      api<{ items: Signal[]; total: number }>(`/ieds/${selected}/signals?q=${encodeURIComponent(query)}&fc=${fc}&offset=${offset}&report=${encodeURIComponent(reportFilter)}`)
        .then(result => { if (!cancelled) { setSignals(result.items); setTotal(result.total); setError(''); } })
        .catch(reason => { if (!cancelled) setError(reason.message); });
    }, 150);
    return () => { cancelled = true; clearTimeout(timer); };
  }, [selected, query, fc, offset, running, reportFilter]);
  const ied = ieds.find(item => item.id === selected);
  return <div className="main-view">
    <div className="view-heading"><div><h1>IEDs & Signals</h1><p className="muted small">Your data models, reports and live values.</p></div>
      <div className="row"><button disabled={running} onClick={importFile}><Upload size={15} />Import CID / SCL</button>
        <button className="primary" disabled={running} onClick={() => setAdding(!adding)}><Plus size={15} />Add template</button></div></div>
    <div className="content-page" style={{ paddingTop: 0 }}>
      {adding && <div className="card" style={{ padding: 16, marginBottom: 20 }}><div className="row">
        <label style={{ flex: 1 }}>IED name<input value={newName} onChange={e => setNewName(e.target.value)} /></label>
        <button className="primary" onClick={() => { addTemplate(newName); setAdding(false); }}>Create bay controller</button>
      </div><p className="small muted" style={{ margin: '10px 0 0' }}>Three switch controls, two analog values, and buffered/unbuffered reports.</p></div>}
      {!ieds.length && <div className="intro-card"><Box size={30} /><h1 style={{ marginTop: 20 }}>Bring your own IED</h1>
        <p>Import a CID, ICD or SCD file to load its IEC 61850 data model. Or start with a laboratory bay-controller template.</p>
        <p className="small">Original engineering files are preserved. Simulation addresses are assigned separately.</p></div>}
      {!!ieds.length && <div className="row" style={{ marginBottom: 16, flexWrap: 'wrap' }}>
        <button disabled={running} onClick={() => setChecked(ieds.map(item => item.id))}>Select all</button>
        <button disabled={running || !checked.length} onClick={() => setChecked([])}>Clear selection</button>
        <BulkDeleteIeds ids={checked} disabled={running} prepare={prepare} perform={perform} clear={() => setChecked([])} />
        {running && <span className="small muted">Stop simulation before deleting IEDs.</span>}
      </div>}
      <div className="ied-cards">{ieds.map(item => <div key={item.id} className="ied-selection">
        <label className="row small"><input type="checkbox" disabled={running} checked={checked.includes(item.id)}
          onChange={e => setChecked(current => e.target.checked ? [...current, item.id] : current.filter(id => id !== item.id))} />Select {item.name}</label>
        <button className={`ied-card ${selected === item.id ? 'active' : ''}`}
        onClick={() => setSelected(item.id)}><div className="row" style={{ justifyContent: 'space-between' }}>
          <Box size={18} /><span className={`badge ${item.state === 'running' ? 'live' : ''}`}><span className="dot" />{item.state}</span></div>
        <div className="ied-name">{item.name}</div><code>{item.ip}:{item.port}</code>
        <div className="ied-info"><span>{item.signal_count.toLocaleString()} attributes</span><span>{item.reports.length} reports</span></div>
      </button></div>)}</div>
      {ied && <div className="card"><div className="card-heading"><div className="row"><h2>Data model</h2><span className="badge">{total.toLocaleString()}</span></div>
        <div className="row"><div className="search-field"><Search size={14} /><input aria-label="Search data references" placeholder="Search data references…"
          value={query} onChange={e => setQuery(e.target.value)} /></div>
          <select style={{ width: 85 }} aria-label="Functional constraint" value={fc} onChange={e => setFc(e.target.value)}>
            <option value="">All FCs</option>{['ST', 'MX', 'CF', 'SP', 'CO', 'DC'].map(item => <option key={item}>{item}</option>)}
          </select><a className="button" title="Download original SCL" href={`/api/v1/ieds/${ied.id}/source`}><Download size={14} /></a>
          <ExportScl key={`export-${ied.id}`} ied={ied} />
          <DeleteIed key={ied.id} iedId={ied.id} disabled={running} prepare={prepare} perform={perform} /></div></div>
        <div className="card-heading"><label>Filter by report<select value={reportFilter} onChange={e => setReportFilter(e.target.value)}>
          <option value="">All signals / not report-filtered</option>
          {ied.reports.map(report => <option key={report.ref} value={report.ref}>{report.ref} · {report.dataset_ref}</option>)}
        </select></label></div>
        {error && <div className="error-banner">{error}</div>}
        <SignalTable signals={signals} values={values} iedId={ied.id} running={running}
          update={(reference, value) => perform(() => api('/signals/update', 'POST', { ied_id: ied.id, ref: reference, value }))} />
        <div className="card-heading"><span className="small muted">{offset + 1}–{Math.min(offset + 100, total)} of {total} · Enter or Set applies a manual override</span>
          <div className="row"><button disabled={!offset} onClick={() => setOffset(Math.max(0, offset - 100))}>Previous</button>
            <button disabled={offset + 100 >= total} onClick={() => setOffset(offset + 100)}>Next</button></div></div>
        {!!ied.warnings.length && <div className="warning-list">{ied.warnings.map(warning => <div key={warning}>{warning}</div>)}</div>}
        {ied.validation && <details style={{ padding: '12px 20px' }}>
          <summary className="small">Source document schema: {ied.validation.status} ({ied.validation.schema || 'not bundled'})</summary>
          <p className="small muted">XSD validation checks document structure, not device behavior or interoperability.</p>
          {ied.validation.errors.map((issue, index) => <p className="small" key={index}>Line {issue.line}: {issue.message}</p>)}
        </details>}
      </div>}
    </div>
  </div>;
}
