// License status and activation calls. Errors keep the server's code so the
// screen can offer the right next step (for example "move the key here").
export type LicenseStatus = {
  licensed: boolean;
  mode: 'development' | 'trial' | 'lifetime' | 'unlicensed';
  plan: string | null;
  days_remaining: number | null;
  expires_at: number | null;
  reason: string | null;
  license_id: string | null;
  email: string | null;
  enforced: boolean;
  server_url: string;
  masked_key: string | null;
  hwid: string | null;
  transfer_available: boolean;
};

export class LicenseError extends Error {
  code: string;
  constructor(message: string, code: string) { super(message); this.code = code; }
}

export const LICENSE_REQUIRED_EVENT = 'studio-license-required';

async function call(path: string, method = 'GET', body?: unknown): Promise<LicenseStatus> {
  const response = await fetch(`/api/v1/license${path}`, {
    method, credentials: 'same-origin',
    headers: { 'X-Studio-Frontend': window.location.origin, ...(body ? { 'Content-Type': 'application/json' } : {}) },
    body: body ? JSON.stringify(body) : undefined,
  });
  const data = await response.json().catch(() => ({ error: `Request failed (${response.status})` }));
  if (!response.ok) throw new LicenseError(data.error || `Request failed (${response.status})`, data.code || 'error');
  return data as LicenseStatus;
}

export const getLicenseStatus = () => call('/status');
export const startTrial = () => call('/trial', 'POST', {});
export const activateLicense = (licenseKey: string, move = false) =>
  call('/activate', 'POST', { license_key: licenseKey, move });
export const activateOffline = (certificate: string) => call('/activate-offline', 'POST', { certificate });
export const updateLicenseServerUrl = (serverUrl: string) => call('/config', 'PUT', { server_url: serverUrl });

export function buyUrl(status: LicenseStatus): string | null {
  if (!status.server_url) return null;
  return `${status.server_url}/buy${status.hwid ? `?hwid=${encodeURIComponent(status.hwid)}` : ''}`;
}

export function offlineUrl(status: LicenseStatus): string | null {
  return status.server_url ? `${status.server_url}/activate` : null;
}
