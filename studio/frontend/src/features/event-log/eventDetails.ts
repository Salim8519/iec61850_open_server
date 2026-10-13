import type { StudioEvent } from '../../api/types';

function rendered(value: unknown) {
  return typeof value === 'boolean' ? String(value) : String(value ?? 'unknown');
}

function feedbackDetails(event: StudioEvent) {
  const targets = Number(event.target_count || 0);
  const changed = Number(event.changed_count || 0);
  const feedback = targets === 0 ? 'no mapped feedback' :
    changed > 0 ? `feedback changed (${changed}/${targets})` : 'feedback unchanged';
  const raw = event.raw_wire_value !== undefined && event.raw_wire_value !== event.wire_value ?
    `requested ctlVal=${rendered(event.raw_wire_value)} / executed ctlVal=${rendered(event.wire_value)}` :
    `ctlVal=${rendered(event.wire_value)}`;
  return `${rendered(event.control_ref)} / ${raw} / ${rendered(event.action)} / ${feedback}`;
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
    const adapted = event.auto_applied === true;
    if (adapted && event.state === 'selected') {
      return `${reference} / selected; CONCEN compatibility operation applied immediately`;
    }
    if (adapted && event.state === 'timed_out') {
      return `${reference} / selection timed out after compatibility operation was already applied`;
    }
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
  if (event.kind === 'control_compatibility_applied') {
    return `${rendered(event.ref)} / CONCEN ${rendered(event.requested_action)} adapted to ` +
      `${rendered(event.executed_action)} and executed from SBOw`;
  }
  if (event.kind === 'control_compatibility_oper_confirmed') {
    return `${rendered(event.ref)} / matching Oper acknowledged; duplicate movement suppressed`;
  }
  if (event.kind === 'equipment_feedback_observed'
      || event.kind === 'equipment_feedback_applied') {
    return feedbackDetails(event);
  }
  if (event.kind === 'control_applied') {
    const raw = event.raw_ctl_value !== undefined && event.raw_ctl_value !== event.ctl_value ?
      `requested ctlVal=${rendered(event.raw_ctl_value)} / executed ctlVal=${rendered(event.ctl_value)}` :
      `ctlVal=${rendered(event.ctl_value)}`;
    return `${rendered(event.ref)} / ${raw} / ${rendered(event.action)}`;
  }
  const subject = event.message || event.ref || event.status_ref || event.state || '';
  return `${subject}${event.value !== undefined ? ` -> ${rendered(event.value)}` : ''}`;
}
