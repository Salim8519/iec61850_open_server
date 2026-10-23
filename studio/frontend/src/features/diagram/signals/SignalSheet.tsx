import { Radio, Share2, X } from 'lucide-react';
import { useSyncExternalStore } from 'react';
import type { GooseLink } from '../../../api/signalLayers';
import { clientLine, externalClients, linkLine, linkStatus, outsideNames } from '../../../copy/signalLayers';
import type { SignalLayerStore } from '../../../equipment/signalLayerStore';
import { StatusPill } from '../../../ui/StatusPill';
import type { SignalSelection } from './SignalOverlay';

/** Side sheet for a clicked GOOSE link, chip or IED component flash: the declared signal-flow details plus the live verdict. */
export function SignalSheet({ store, selection, select, showFlow }: {
  store: SignalLayerStore; selection: NonNullable<SignalSelection>;
  select: (selection: SignalSelection) => void; showFlow: (iedId?: string) => void;
}) {
  const state = useSyncExternalStore(store.subscribe, store.get, store.get);
  const running = state.map.running;
  const link = selection.kind === 'link' ? state.map.links.find(item => item.id === selection.id) : undefined;
  const name = selection.kind === 'ied' ? selection.name : link?.subscriber || '';
  const ied = state.map.ieds.find(item => item.name === (link ? link.subscriber : name));
  const flowTarget = ied?.ied_id || state.map.ieds.find(item => item.name === link?.publisher)?.ied_id || undefined;
  const outside = outsideNames(state.map.ieds);
  const title = link ? `${link.publisher} → ${link.subscriber}` : name;
  return <aside className="sheet signal-sheet" aria-label={`${title} signals`}>
    <div className="sheet-head">
      <span className="sheet-head-symbol"><Radio size={26} aria-hidden /></span>
      <div className="sheet-title"><h2 title={title}>{title}</h2><p>{link ? `GOOSE ${link.block || 'from a vendor input'}` : 'IED signals'}</p></div>
      <button className="icon ghost" aria-label="Close signal details" onClick={() => select(null)}><X size={18} /></button>
    </div>
    <div className="sheet-body">
      {link ? <LinkDetails link={link} running={running} outside={outside.has(link.publisher)} block={state.map.blocks.find(item => item.gocb_ref === link.gocb_ref)} />
        : <IedLinks name={name} links={state.map.links.filter(item => item.publisher === name || item.subscriber === name)}
          running={running} outside={outside} select={select} />}
      {selection.kind === 'ied' && ied?.ied_id && <section className="sheet-section"><h3>Read by clients</h3>
        {externalClients(state.clients[ied.ied_id]).length ? externalClients(state.clients[ied.ied_id]).map(client =>
          <p key={client.peer} className="small" style={{ margin: 0 }}>{clientLine(client)}</p>)
          : <p className="small muted" style={{ margin: 0 }}>{running ? 'No SCADA or test program is connected over MMS.'
            : 'Press Run to see which programs connect.'}</p>}</section>}
      {flowTarget && <section className="sheet-section"><button className="ghost" onClick={() => showFlow(flowTarget)}>
        <Share2 size={16} />Open in Signal flow</button></section>}
    </div>
  </aside>;
}

function LinkDetails({ link, running, outside, block }: { link: GooseLink; running: boolean; outside: boolean;
  block?: { app_id: number | null; mac: string; dataset: string; conf_rev: number | null } }) {
  const rows: [string, string][] = [['Control block', link.gocb_ref || 'Not named in the file'],
    ['Dataset', block?.dataset || '—'], ['APPID', block?.app_id === null || block?.app_id === undefined ? '—'
      : block.app_id.toString(16).padStart(4, '0').toUpperCase()], ['MAC', block?.mac || link.mac || '—'],
    ['Configuration revision', block?.conf_rev === null || block?.conf_rev === undefined ? '—' : String(block.conf_rev)],
    ['Matched by', link.match === 'named' ? 'block name (srcCBName)' : link.match === 'vendor_hint'
      ? 'vendor input address' : 'data (FCDA)']];
  return <>
    <section className="sheet-section"><StatusPill status={linkStatus(link, running, outside)} showHint />
      <dl>{rows.map(([label, value]) => <div key={label} style={{ display: 'contents' }}><dt>{label}</dt><dd>{value}</dd></div>)}</dl>
    </section>
    <section className="sheet-section"><h3>{link.inputs.length} signal{link.inputs.length === 1 ? '' : 's'} received</h3>
      <table><thead><tr><th>Input</th><th>From</th></tr></thead><tbody>{link.inputs.map(input =>
        <tr key={`${input.int_addr}|${input.source}`}><td>{input.desc || input.int_addr}</td><td>{input.source}</td></tr>)}</tbody></table>
    </section>
  </>;
}

/** One line per link, "← C11_Y120 · GoCB01 · Receiving", the reason on hover (change 0097). */
function IedLinks({ name, links, running, outside, select }: { name: string; links: GooseLink[]; running: boolean;
  outside: Set<string>; select: (selection: SignalSelection) => void }) {
  return <section className="sheet-section"><h3>GOOSE</h3>
    {links.length ? links.map(link => {
      const status = linkStatus(link, running, outside.has(link.publisher));
      return <button key={link.id} className="ghost link-row" title={status.hint}
        onClick={() => select({ kind: 'link', id: link.id })}>
        <span>{linkLine(link, name)} · {status.tone === 'problem' ? '⚠ ' : ''}{status.label}</span></button>;
    }) : <p className="small muted" style={{ margin: 0 }}>This IED neither sends nor expects GOOSE in its files.</p>}
  </section>;
}
