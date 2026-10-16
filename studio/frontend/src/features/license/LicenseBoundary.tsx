import { useCallback, useEffect, useState, type ReactNode } from 'react';
import { connectSession } from '../../api/client';
import { getLicenseStatus, LICENSE_REQUIRED_EVENT, type LicenseStatus } from '../../api/license';
import { LicenseGate } from './LicenseGate';

/**
 * Shows the app only when the backend says this computer is licensed. Development
 * backends always answer "licensed", so this renders the app straight away there.
 * A 402 from any later request (for example a trial ending mid-session) brings the
 * activation screen back without losing saved work.
 */
export function LicenseBoundary({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<LicenseStatus | null>(null);
  const [failure, setFailure] = useState('');
  const load = useCallback(() => getLicenseStatus().then(setStatus).catch(error => setFailure(String(error.message || error))), []);
  useEffect(() => { connectSession().catch(() => {}).then(load); }, [load]);
  useEffect(() => {
    const recheck = () => { load(); };
    window.addEventListener(LICENSE_REQUIRED_EVENT, recheck);
    return () => window.removeEventListener(LICENSE_REQUIRED_EVENT, recheck);
  }, [load]);
  if (status?.licensed) return <>{children}</>;
  if (status) return <LicenseGate status={status} changed={setStatus} />;
  // Unknown state (old backend, session missing): show the app; it reports its own errors.
  if (failure) return <>{children}</>;
  return null;
}
