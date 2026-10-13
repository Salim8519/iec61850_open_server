import { Download } from 'lucide-react';
import type { Ied } from '../../api/types';
import { DeleteIed } from './DeleteIed';
import { ExportScl } from './ExportScl';

export function DeviceFiles({ ied, locked, perform, prepare }: {
  ied: Ied; locked: boolean; perform: (task: () => Promise<unknown>) => Promise<void>; prepare: () => Promise<void>;
}) {
  return <div className="card-body">
    <div className="list">
      <div className="row" style={{ justifyContent: 'space-between' }}><div><h3 style={{ margin: 0 }}>Original file</h3>
        <p className="small muted" style={{ margin: 0 }}>Exactly the file you imported, unchanged.</p></div>
        <a className="button" href={`/api/v1/ieds/${ied.id}/source`}><Download size={16} />Download</a></div>
      <hr style={{ margin: '8px 0' }} />
      <div className="row" style={{ justifyContent: 'space-between' }}><div><h3 style={{ margin: 0 }}>Edited copy</h3>
        <p className="small muted" style={{ margin: 0 }}>Download a checked copy with changed starting values.</p></div>
        <ExportScl ied={ied} /></div>
      <hr style={{ margin: '8px 0' }} />
      <div className="row" style={{ justifyContent: 'space-between' }}><div><h3 style={{ margin: 0 }}>Remove from project</h3>
        <p className="small muted" style={{ margin: 0 }}>A backup of the project is saved first.</p></div>
        <DeleteIed iedId={ied.id} disabled={locked} prepare={prepare} perform={perform} /></div>
    </div>
  </div>;
}
