import { useEffect, useState } from 'react';
import { snmpApi } from '../../api/snmp';

/** How many simulated SNMP devices follow the IEDs onto the network (ADR-0092); 0 on the Python
 *  engine, which has no SNMP tool. Read again when the project revision changes. */
export function useSnmpCount(revision: number) {
  const [count, setCount] = useState(0);
  useEffect(() => {
    let live = true;
    snmpApi.config().then(config => { if (live) setCount(config.devices.length); }).catch(() => { if (live) setCount(0); });
    return () => { live = false; };
  }, [revision]);
  return count;
}
