import { useState } from 'react';
import { GATEWAY_NETWORK_COPY as COPY } from '../../../copy/gateway';

/** The value of the "Type an address…" item in a Network list (ADR-0125). */
export const TYPE_ADDRESS = '__type';

/** The box "Type an address…" opens in place: the Go engine checks the address and puts it on the
 *  ScadaBench Lab card; this only sends it and shows why it was refused. */
export function TypeAddress({ labReady, placeholder, add }: {
  labReady: boolean; placeholder: string; add: (address: string) => Promise<unknown>;
}) {
  const [typed, setTyped] = useState('');
  const [error, setError] = useState('');
  const send = () => {
    setError('');
    add(typed.trim()).catch((problem: Error) => setError(problem.message));
  };
  return <>
    <div className="label-row" style={{ gap: 8 }}>
      <input autoFocus value={typed} placeholder={placeholder} aria-label={COPY.type}
        onChange={event => setTyped(event.target.value)} onKeyDown={event => { if (event.key === 'Enter') send(); }} />
      <button type="button" onClick={send} disabled={!typed.trim()}>{COPY.add}</button></div>
    {!labReady && <p className="side-line">{COPY.noLab}</p>}
    {error && <p className="side-error" role="alert">{error}</p>}
  </>;
}
