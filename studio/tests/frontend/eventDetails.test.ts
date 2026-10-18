import assert from 'node:assert/strict';
import test from 'node:test';
import { eventDetails } from '../../frontend/src/features/event-log/eventDetails.ts';

test('external control detail shows reference, raw value, action and feedback result', () => {
  const base = { id: 1, timestamp: 1, kind: 'equipment_feedback_observed',
    control_ref: 'DemoCTRL/GGIO1$CO$SPCSO1$Oper', wire_value: true,
    action: 'reset', target_count: 1, changed_count: 1 };
  assert.equal(eventDetails(base),
    'DemoCTRL/GGIO1$CO$SPCSO1$Oper / ctlVal=true / reset / feedback changed (1/1)');
  assert.match(eventDetails({ ...base, changed_count: 0 }), /feedback unchanged$/);
  assert.match(eventDetails({ ...base, target_count: 0, changed_count: 0 }), /no mapped feedback$/);
  assert.match(eventDetails({ ...base, kind: 'equipment_feedback_applied' }),
    /ctlVal=true \/ reset \/ feedback changed \(1\/1\)$/);
});

test('enhanced SBO selection explains that the switch moves only on Oper', () => {
  const selected = { id: 2, timestamp: 1, kind: 'control_selection',
    ref: 'BayCTRL/XCBR1.Pos', state: 'selected', sbo_timeout_ms: 30000 };
  assert.equal(eventDetails(selected),
    'BayCTRL/XCBR1.Pos / selected; waiting for Oper (30-second timeout)');
  assert.match(eventDetails({ ...selected, state: 'timed_out' }), /selection timed out$/);
  assert.match(eventDetails({ ...selected, state: 'operated' }), /selection completed$/);
});

test('an applied control shows the value it was given, never an adapted one (ADR-0069)', () => {
  const applied = { id: 3, timestamp: 1, kind: 'control_applied',
    ref: 'BayCTRL/XCBR1.Pos', ctl_value: true, action: 'close' };
  assert.equal(eventDetails(applied), 'BayCTRL/XCBR1.Pos / ctlVal=true / close');
});
