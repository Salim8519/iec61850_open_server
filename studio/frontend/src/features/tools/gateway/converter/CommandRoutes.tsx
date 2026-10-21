import { Plus, Save, X } from 'lucide-react';
import { useEffect, useState } from 'react';
import { masterApi, type CommandRoute } from '../../../../api/master';
import { CONVERTER_COPY, DEVICE_KINDS, UPSTREAM_KINDS } from '../../../../copy/converter';
import { kindOfGatewayPoint, kindOfMasterCommand, type Kind } from '../../../../points/pointKind';
import { EditableText } from '../../../../ui/EditableText';
import { HelpTip } from '../../../../ui/HelpTip';
import { KindBadge, kindRow } from '../../../../ui/KindBadge';
import { ChoiceEdit } from '../ChoiceEdit';

const choicesOf = (names: Record<string, string>) => Object.entries(names).map(([value, label]) => ({ value, label }));
const UPSTREAM_CHOICES = choicesOf(UPSTREAM_KINDS);
const DEVICE_CHOICES = choicesOf(DEVICE_KINDS);
/** A command type as it is shown in a route: badge + the name the choice list uses. */
const shownKind = (kind: Kind, name: string, raw?: string) => <span className="type-cell"><KindBadge kind={kind} raw={raw} />
  <span className="small">{name}</span></span>;

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
      <div className="table-scroll"><table className="gateway-points compact-rows"><thead><tr><th>{CONVERTER_COPY.upstream}</th>
        <th>{CONVERTER_COPY.upstreamKind}</th><th>{CONVERTER_COPY.deviceKind}</th><th>{CONVERTER_COPY.deviceAddress}</th>
        <th>{CONVERTER_COPY.select}</th><th aria-label="Remove" /></tr></thead>
        <tbody>{(routes ?? []).map((route, index) => <tr key={index} {...kindRow(kindOfGatewayPoint(route.upstream_kind))}>
          <td className="address-cell"><EditableText mono inputMode="numeric" value={String(route.upstream)} disabled={locked}
            label={CONVERTER_COPY.upstream} onSave={text => edit(index, { upstream: number(text) })} /></td>
          <td><ChoiceEdit label={CONVERTER_COPY.upstreamKind} value={route.upstream_kind} choices={UPSTREAM_CHOICES} disabled={locked}
            shown={shownKind(kindOfGatewayPoint(route.upstream_kind), UPSTREAM_KINDS[route.upstream_kind] ?? route.upstream_kind)}
            onChange={upstream_kind => edit(index, { upstream_kind })} /></td>
          <td><ChoiceEdit label={CONVERTER_COPY.deviceKind} value={route.kind} choices={DEVICE_CHOICES} disabled={locked}
            shown={shownKind(kindOfMasterCommand(route.kind), DEVICE_KINDS[route.kind] ?? route.kind, DEVICE_KINDS[route.kind])}
            onChange={kind => edit(index, { kind })} /></td>
          <td className="address-cell"><EditableText mono inputMode="numeric" value={String(route.address)} disabled={locked}
            label={CONVERTER_COPY.deviceAddress} onSave={text => edit(index, { address: number(text) })} /></td>
          <td><input type="checkbox" checked={route.select} disabled={locked} aria-label={CONVERTER_COPY.select}
            onChange={event => edit(index, { select: event.target.checked })} /></td>
          <td><button className="ghost icon-button" disabled={locked} aria-label="Remove route" title="Remove"
            onClick={() => { setRoutes(routes!.filter((_, at) => at !== index)); setDirty(true); }}><X size={15} /></button></td>
        </tr>)}</tbody></table></div>}
  </>;
}
