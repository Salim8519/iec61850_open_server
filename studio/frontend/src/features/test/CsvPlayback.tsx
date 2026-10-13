import { useState } from 'react';
import { upload } from '../../api/client';
import { configureSource } from './signalActions';

type Preview = { columns: string[]; rows: string[][]; row_count: number };

export function CsvPlayback({ iedId, reference, perform }: {
  iedId: string; reference: string; perform: (task: () => Promise<unknown>) => Promise<void>;
}) {
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [timeColumn, setTimeColumn] = useState('');
  const [valueColumn, setValueColumn] = useState('');
  const [interpolation, setInterpolation] = useState('hold');
  const [repeat, setRepeat] = useState(false);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState('');
  return <div className="stack">
    <label>CSV trace<input type="file" accept=".csv,text/csv" disabled={busy} onChange={async event => {
      const chosen = event.target.files?.[0];
      setFile(null); setPreview(null); setMessage('');
      if (!chosen) return;
      setBusy(true);
      await perform(async () => {
        if (chosen.size > 5 * 1024 * 1024) throw new Error('CSV trace must be at most 5 MB.');
        const result = await upload('/traces/preview', chosen) as Preview;
        setPreview(result); setFile(chosen);
        setTimeColumn(result.columns.includes('time') ? 'time' : result.columns[0]);
        setValueColumn(result.columns.includes('value') ? 'value' : result.columns[1]);
      });
      setBusy(false);
    }} /></label>
    <p className="small muted">UTF-8, comma-separated CSV with column headers. Time is seconds from 0,
      strictly increasing. Up to 100,000 rows / 5 MB. Values use the selected signal's units;
      no unit conversion is applied.</p>
    {preview && <>
      <div className="generator-grid">
        <label>Time column<select value={timeColumn} onChange={e => setTimeColumn(e.target.value)}>
          {preview.columns.map(column => <option key={column}>{column}</option>)}</select></label>
        <label>Value column<select value={valueColumn} onChange={e => setValueColumn(e.target.value)}>
          {preview.columns.map(column => <option key={column}>{column}</option>)}</select></label>
        <label>Between samples<select value={interpolation} onChange={e => setInterpolation(e.target.value)}>
          <option value="hold">Hold previous value</option><option value="linear">Linear interpolation</option></select></label>
        <label>After last sample<select value={String(repeat)} onChange={e => setRepeat(e.target.value === 'true')}>
          <option value="false">Hold final value</option><option value="true">Repeat trace</option></select></label>
      </div>
      <div className="table-scroll"><table><caption className="small muted">Preview: first {preview.rows.length} of {preview.row_count} rows</caption>
        <thead><tr>{preview.columns.map(column => <th key={column}>{column}</th>)}</tr></thead>
        <tbody>{preview.rows.map((row, index) => <tr key={index}>{row.map((cell, column) => <td key={column}>{cell}</td>)}</tr>)}</tbody></table></div>
      <div className="row"><button className="primary" disabled={!reference || !file || busy || timeColumn === valueColumn}
        onClick={async () => {
          if (!file) return;
          setBusy(true); setMessage('');
          await perform(async () => {
            await configureSource({ ied_id: iedId, ref: reference }, {
              mode: 'csv', filename: file.name, csv_text: await file.text(), time_column: timeColumn,
              value_column: valueColumn, interpolation, repeat });
            setMessage('Trace saved in the project. Run or resume automation to play it.');
          });
          setBusy(false);
        }}>Apply CSV playback</button><span className="small muted">{message}</span></div>
    </>}
  </div>;
}
