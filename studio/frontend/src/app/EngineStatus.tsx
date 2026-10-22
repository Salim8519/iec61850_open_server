import { useSyncExternalStore } from 'react';
import { CONNECTION_COPY, connectionHealth, type ConnectionState } from '../api/connectionHealth';

/** The connection the page really has (change 0177): it follows every API answer and failure. */
export function useConnection(loaded: boolean): ConnectionState {
  const state = useSyncExternalStore(connectionHealth.subscribe, connectionHealth.state);
  return state === 'connected' && !loaded ? 'connecting' : state;
}

const COLOR: Record<ConnectionState, string> = { connecting: 'var(--warning)', connected: 'var(--success)', lost: 'var(--error)' };

/** The sidebar line: a dot and words (the words stay in the title when the sidebar is narrow). */
export function EngineStatus({ state }: { state: ConnectionState }) {
  const words = CONNECTION_COPY[state];
  return <span className="small muted engine-status" style={{ padding: '4px 12px' }}
    title={state === 'lost' ? CONNECTION_COPY.lostHint : words} role="status">
    <span className="dot" style={{ display: 'inline-block', marginRight: 8, color: COLOR[state] }} />
    <span>{words}</span></span>;
}

/** One line over the page while the engine cannot be reached, so old values are never taken as live. */
export function ConnectionNotice({ state }: { state: ConnectionState }) {
  if (state !== 'lost') return null;
  return <div className="error-banner" role="alert"><span><strong>{CONNECTION_COPY.lost}.</strong> {CONNECTION_COPY.lostHint}</span></div>;
}
