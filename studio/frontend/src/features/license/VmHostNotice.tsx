import { useEffect } from 'react';
import { getLicenseStatus, type LicenseStatus } from '../../api/license';
import { VM_HOST_NOTICE_SECONDS, VM_HOST_COPY as COPY } from '../../copy/vmHost';

/**
 * A line on the activation screen for people who run ScadaBench inside a virtual machine: the Go backend
 * looks for the host PC by itself in the background, so the screen only has to ask again now and then and
 * carry on as soon as the host has answered (ADR-0099).
 */
export function VmHostNotice({ changed }: { changed: (status: LicenseStatus) => void }) {
  useEffect(() => {
    const timer = window.setInterval(() => {
      getLicenseStatus().then(status => { if (status.licensed) changed(status); }).catch(() => {});
    }, VM_HOST_NOTICE_SECONDS * 1000);
    return () => window.clearInterval(timer);
  }, [changed]);
  return <p className="small muted license-hint" style={{ margin: 0 }}>{COPY.gateNotice}</p>;
}
