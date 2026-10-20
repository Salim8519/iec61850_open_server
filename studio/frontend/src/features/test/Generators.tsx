import { Pencil, Plus, Waves, X } from 'lucide-react';
import { useState } from 'react';
import type { Project } from '../../api/types';
import { shortRef } from '../../equipment/relayDisplay';
import { EmptyState } from '../../ui/EmptyState';
import { SourceEditor } from './SourceEditor';
import { modeLabel } from './sourceModes';

export function Generators({ project, perform }: {
  project: Project; perform: (task: () => Promise<unknown>) => Promise<void>;
}) {
  const [editing, setEditing] = useState<string | null>(null);
  const sources = project.sources.filter(item => item.spec.mode !== 'manual');
  const key = (item: { ied_id: string; ref: string }) => `${item.ied_id}:${item.ref}`;
  return <div className="stack">
    <div className="row" style={{ justifyContent: 'space-between' }}>
      <p className="muted" style={{ margin: 0 }}>Generators move measurements automatically while the project runs.</p>
      {(!!sources.length || editing === 'new') && <button className={editing === 'new' ? '' : 'primary'}
        onClick={() => setEditing(editing === 'new' ? null : 'new')}>{editing === 'new' ? <><X size={16} />Cancel</> : <><Plus size={16} />New generator</>}</button>}
    </div>
    {editing === 'new' && <div className="card"><div className="card-heading"><h2>New generator</h2></div>
      <div className="card-body"><SourceEditor project={project} perform={perform} done={() => setEditing(null)} /></div></div>}
    {!sources.length && editing !== 'new' && <div className="card"><EmptyState icon={<Waves size={24} />} title="No generators yet"
      hintId="test.generators.empty" action={<button className="primary" onClick={() => setEditing('new')}><Plus size={16} />New generator</button>} /></div>}
    {!!sources.length && <div className="card"><table><thead><tr><th>Measurement</th><th>Behaviour</th><th style={{ width: 90 }} /></tr></thead>
      <tbody>{sources.map(item => {
        const ied = project.ieds.find(entry => entry.id === item.ied_id);
        return [<tr key={key(item)}><td><strong>{shortRef(item.ref, ied?.name)}</strong><div className="small muted">{ied?.name}</div></td>
          <td>{modeLabel(item.spec.mode)}{item.spec.mode !== 'csv' && item.spec.mode !== 'constant' &&
            <span className="small muted"> · {item.spec.offset} ± {item.spec.amplitude}, every {item.spec.period}s</span>}</td>
          <td><button className="ghost" onClick={() => setEditing(editing === key(item) ? null : key(item))}><Pencil size={15} />Edit</button></td></tr>,
          editing === key(item) && <tr key={`${key(item)}-edit`}><td colSpan={3} style={{ background: 'var(--surface-2)' }}>
            <SourceEditor point={item} project={project} perform={perform} done={() => setEditing(null)} /></td></tr>];
      })}</tbody></table></div>}
  </div>;
}
