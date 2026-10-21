import { ChevronDown, Download, ListChecks, Plus } from 'lucide-react';
import { snmpApi } from '../../../../api/snmp';
import { POINTS_COPY } from '../../../../copy/snmp';
import { HelpTip } from '../../../../ui/HelpTip';
import { Menu } from '../../../../ui/Menu';
import { startDownload } from '../../startDownload';

/** The buttons in front of the Values filters: Add value, Select, and More ▾ (Export CSV). */
export function PointsActions({ device, adding, toggleAdding, picking, togglePicking }: {
  device: string; adding: boolean; toggleAdding: () => void; picking: boolean; togglePicking: () => void;
}) {
  return <div className="snmp-actions">
    <button className={adding ? 'active-toggle' : ''} aria-pressed={adding} onClick={toggleAdding}>
      <Plus size={16} />{POINTS_COPY.add}</button>
    <button className={picking ? 'active-toggle' : ''} aria-pressed={picking} title={POINTS_COPY.selectHint}
      onClick={togglePicking}><ListChecks size={16} />{POINTS_COPY.select}</button>
    <Menu label={POINTS_COPY.moreLabel} trigger={<>{POINTS_COPY.more}<ChevronDown size={14} aria-hidden /></>} align="left"
      items={[{ label: POINTS_COPY.exportCsv, icon: <Download size={16} />, onSelect: () => startDownload(snmpApi.csvUrl(device)) }]} />
    <HelpTip text={POINTS_COPY.helpText} label="How do I change values?" />
  </div>;
}
