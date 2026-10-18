import { Plus, Save, X } from 'lucide-react';
import { useEffect, useState } from 'react';
import { masterApi, type CommandRoute } from '../../../../api/master';
import { CONVERTER_COPY, DEVICE_KINDS, UPSTREAM_KINDS } from '../../../../copy/converter';
import { HelpTip } from '../../../../ui/HelpTip';

/** Upstream command address -> device command; needed where the device cannot be discovered (IEC 104). */
export function CommandRoutes({ locked, fail }: { locked: boolean; fail: (message: string) => void }) {
  const [routes, setRoutes] = useState<CommandRoute[] | null>(null);
  const [dirty, setDirty] = useState(false);
  useEffect(() => { masterApi.config().then(config => setRoutes(config.routes)).catch(error => fail(error.message)); }, []);
  const edit = (index: number, change: Partial<CommandRoute>) => {
    setRoutes((routes ?? []).map((route, at) => at === index ? { ...route, ...change } : route));
    setDirty(true);
  };
  const add = () => {
    const last = routes?.[routes.length - 1];
    setRoutes([...(routes ?? []), { upstream: (last?.upstream ?? 7000) + 1, upstream_kind: 'command_double', kind: 'double',
      address: (last?.address ?? 6000) + 1, select: true }]);
    setDirty(true);
  };
  const save = () => routes && masterApi.saveRoutes(routes).then(answer => { setRoutes(answer.routes); setDirty(false); })
    .catch(error => fail(error.message));
  const number = (value: string) => Number(value.replace(/\D/g, '') || 0);
  return <>
    <div className="panel-toolbar">
      <button onClick={add} disabled={locked}><Plus size={16} />{CONVERTER_COPY.addRoute}</button>
      <HelpTip text={CONVERTER_COPY.routesHint} label="What are command routes?" />
      <span className="gateway-spacer" />
      {locked && <span className="small muted">Stop serving to change the routes.</span>}
      {dirty && <button className="primary" onClick={save}><Save size={16} />{CONVERTER_COPY.saveRoutes}</button>}
    </div>
    {routes && routes.length === 0 ? <p className="muted watch-empty">{CONVERTER_COPY.routesEmpty}</p> :
      <div className="table-scroll"><table className="gateway-points"><thead><tr><th>{CONVERTER_COPY.upstream}</th>
        <th>{CONVERTER_COPY.upstreamKind}</th><th>{CONVERTER_COPY.deviceKind}</th><th>{CONVERTER_COPY.deviceAddress}</th>
        <th>{CONVERTER_COPY.select}</th><th aria-label="Remove" /></tr></thead>
        <tbody>{(routes ?? []).map((route, index) => <tr key={index}>
          <td><input className="gateway-number" value={route.upstream} disabled={locked} aria-label={CONVERTER_COPY.upstream}
            onChange={event => edit(index, { upstream: number(event.target.value) })} /></td>
          <td><select value={route.upstream_kind} disabled={locked} aria-label={CONVERTER_COPY.upstreamKind}
            onChange={event => edit(index, { upstream_kind: event.target.value })}>
            {Object.entries(UPSTREAM_KINDS).map(([key, label]) => <option key={key} value={key}>{label}</option>)}</select></td>
          <td><select value={route.kind} disabled={locked} aria-label={CONVERTER_COPY.deviceKind}
            onChange={event => edit(index, { kind: event.target.value })}>
            {Object.entries(DEVICE_KINDS).map(([key, label]) => <option key={key} value={key}>{label}</option>)}</select></td>
          <td><input className="gateway-number" value={route.address} disabled={locked} aria-label={CONVERTER_COPY.deviceAddress}
            onChange={event => edit(index, { address: number(event.target.value) })} /></td>
          <td><input type="checkbox" checked={route.select} disabled={locked} aria-label={CONVERTER_COPY.select}
            onChange={event => edit(index, { select: event.target.checked })} /></td>
          <td><button className="ghost icon-button" disabled={locked} aria-label="Remove route" title="Remove"
            onClick={() => { setRoutes(routes!.filter((_, at) => at !== index)); setDirty(true); }}><X size={15} /></button></td>
        </tr>)}</tbody></table></div>}
  </>;
}
