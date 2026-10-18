import type { Ied, Project } from '../../api/types';
import { linkStatus } from '../../copy/signalLayers';
import { InfoTip } from '../../ui/InfoTip';
import { StatusPill } from '../../ui/StatusPill';
import { useGooseLinks } from './useGooseLinks';
import './goose-partners.css';

/** Overview table of everyone this IED exchanges GOOSE with (spec signal-layers, LAYER-15). */
export function GoosePartners({ ied, project, open }: { ied: Ied; project: Project; open: (iedId: string) => void }) {
  const running = project.runtime !== 'stopped';
  const map = useGooseLinks(project.revision, running);
  const links = map.links.filter(link => link.publisher === ied.name || link.subscriber === ied.name);
  const notes = map.diagnostics.filter(item => item.ied === ied.name && item.level !== 'info').slice(0, 4);
  if (!links.length) return null;
  const idOf = (name: string) => map.ieds.find(item => item.name === name)?.ied_id;
  return <section className="goose-partners">
    <h3 style={{ margin: '4px 0 8px' }}><InfoTip term="goose" label="GOOSE" /> partners</h3>
    <div className="table-scroll"><table><thead><tr><th>Direction</th><th>Partner</th><th>Block · what</th><th>Status</th></tr></thead>
      <tbody>{links.map(link => {
        const sends = link.publisher === ied.name;
        const partner = sends ? link.subscriber : link.publisher;
        const partnerId = idOf(partner);
        const what = link.inputs.slice(0, 2).map(input => input.desc || input.int_addr).join(', ');
        return <tr key={link.id}>
          <td>{sends ? '↑ Sends' : '↓ Gets'}</td>
          <td>{partnerId ? <button className="partner-chip" onClick={() => open(partnerId)}>{partner}</button>
            : <span className="partner-chip outside" title="Outside project: import its CID or connect the real device">{partner}</span>}</td>
          <td>{link.block || 'vendor input'}{what ? ` · ${what}${link.inputs.length > 2 ? ` +${link.inputs.length - 2}` : ''}` : ''}</td>
          <td><StatusPill status={linkStatus(link, running, project.network.mode === 'lab', !partnerId && !sends)} /></td></tr>;
      })}</tbody></table></div>
    {notes.map(note => <p key={note.message} className="small" style={{ margin: '6px 0 0' }} title={note.hint}>• {note.message}</p>)}
  </section>;
}
