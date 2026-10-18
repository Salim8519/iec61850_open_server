import type { StudioEvent } from '../../api/types';

function rendered(value: unknown) {
  return typeof value === 'boolean' ? String(value) : String(value ?? 'unknown');
}

function feedbackDetails(event: StudioEvent) {
  const targets = Number(event.target_count || 0);
  const changed = Number(event.changed_count || 0);
  const feedback = targets === 0 ? 'no mapped feedback' :
    changed > 0 ? `feedback changed (${changed}/${targets})` : 'feedback unchanged';
  return `${rendered(event.control_ref)} / ctlVal=${rendered(event.wire_value)} / ${rendered(event.action)} / ${feedback}`;
}

export function eventDetails(event: StudioEvent) {
  if (event.kind === 'project_changed') {
    return `authoritative project revision ${rendered(event.revision)}`;
  }
  if (event.kind === 'control_selection') {
    const reference = rendered(event.ref);
    const timeout = Number(event.sbo_timeout_ms || 0);
    const duration = timeout && timeout % 1000 === 0 ?
      `${timeout / 1000}-second timeout` : `${timeout} ms timeout`;
    const states: Record<string, string> = {
      selected: `selected; waiting for Oper (${duration})`,
      operated: 'Oper accepted; selection completed',
      canceled: 'selection canceled',
      timed_out: 'selection timed out',
      failed: 'Oper failed; selection released',
      disconnected: 'selecting client disconnected; selection released',
    };
    return `${reference} / ${states[String(event.state)] || rendered(event.state)}`;
  }
  if (event.kind === 'equipment_feedback_observed'
      || event.kind === 'equipment_feedback_applied') {
    return feedbackDetails(event);
  }
  if (event.kind === 'control_applied') {
    return `${rendered(event.ref)} / ctlVal=${rendered(event.ctl_value)} / ${rendered(event.action)}`;
  }
  if (event.kind === 'energization') {
    const counts = new Map<string, number>();
    Object.values((event.edges || {}) as Record<string, string>).forEach(state => counts.set(state, (counts.get(state) || 0) + 1));
    const parts = [...counts].map(([state, count]) => `${count} ${state.replace('_', ' ')}`);
    return parts.length ? `wires now: ${parts.join(', ')}` : 'no power wires on the diagram';
  }
  const subject = event.message || event.ref || event.status_ref || event.state || '';
  return `${subject}${event.value !== undefined ? ` -> ${rendered(event.value)}` : ''}`;
}
