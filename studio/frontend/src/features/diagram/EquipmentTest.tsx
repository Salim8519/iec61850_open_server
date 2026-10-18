import { useState } from 'react';
import type { EquipmentNode, Project } from '../../api/types';
import { kindInfo } from '../../equipment/kindInfo';
import { TEST_COPY, testAvailability, type TestButton } from '../../equipment/testButtons';

export type RunTest = (nodeId: string, action: string, extra?: { value?: number; row?: string }) => void;

/** Test buttons in the equipment side sheet (ADR-0067): only while running, only for simulated IEDs.
 * Each button asks the backend to write the bound point as if the plant changed by itself. */
export function EquipmentTest({ node, project, running, busy, test, message }: {
  node: EquipmentNode; project: Project; running: Set<string>; busy: boolean; test: RunTest; message?: string;
}) {
  const [value, setValue] = useState('');
  const data = node.data;
  const { buttons, reason } = testAvailability(data, project.ieds, running);
  if (reason === 'unbound') return null;
  const switching = ['CB', 'DS', 'ES'].includes(data.kind);
  const press = (button: TestButton) => test(node.id, button.action,
    button.action === 'set' ? { value: Number(value) } : button.row ? { row: button.row } : undefined);
  return <section className="sheet-section"><h3>{TEST_COPY.title}</h3>
    {reason === 'real' && <p className="small muted" style={{ margin: 0 }}>{TEST_COPY.real}</p>}
    {reason === 'not_running' && <p className="small muted" style={{ margin: 0 }}>{TEST_COPY.notRunning}</p>}
    {buttons.some(button => button.action === 'set') && <label className="small">{TEST_COPY.valueLabel}
      <input type="number" step="any" value={value} onChange={event => setValue(event.target.value)} /></label>}
    {buttons.length > 0 && <div className="sheet-actions test-actions">{buttons.map(button =>
      <button key={`${button.row || ''}:${button.action}`} className={button.tone === 'danger' ? 'danger-outline' : undefined}
        disabled={busy || (button.action === 'set' && (value.trim() === '' || !Number.isFinite(Number(value))))}
        onClick={() => press(button)}>{button.label}</button>)}</div>}
    {buttons.length > 0 && <p className="small muted" style={{ margin: 0 }}>
      {switching ? TEST_COPY.explain(kindInfo(data.kind).name) : TEST_COPY.explainOther}</p>}
    {message && <p className="small" role="status" style={{ margin: 0 }}>{message}</p>}
  </section>;
}
