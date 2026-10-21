import { useEffect } from 'react';
import { getLicenseStatus, type LicenseStatus } from '../../api/license';
import { VM_HOST_NOTICE_SECONDS, VM_HOST_COPY as COPY } from '../../copy/vmHost';
import { Disclosure } from '../../ui/Disclosure';
import { Spinner } from '../../ui/Spinner';

/**
 * "Using a virtual machine?" on the activation screen: closed like the offline options beside it. The Go
 * backend looks for the host PC by itself in the background, so the screen asks again now and then, even
 * while this is closed, and carries on as soon as the host has answered (ADR-0099).
 */
export function VmHostNotice({ changed }: { changed: (status: LicenseStatus) => void }) {
  useEffect(() => {
    const timer = window.setInterval(() => {
      getLicenseStatus().then(status => { if (status.licensed) changed(status); }).catch(() => {});
    }, VM_HOST_NOTICE_SECONDS * 1000);
    return () => window.clearInterval(timer);
  }, [changed]);
  return <Disclosure title={COPY.gateTitle}>
    <div className="license-options">
      <p className="small muted license-hint">{COPY.gateIntro}</p>
      <ol className="small license-steps">{COPY.gateSteps.map(step => <li key={step}>{step}</li>)}</ol>
      <Spinner label={COPY.gateLooking} />
      <p className="small muted" style={{ margin: 0 }}>{COPY.gateNote}</p>
    </div>
  </Disclosure>;
}
