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
  /** A trial that could not confirm the date online for longer than its offline grace. */
  online_check_required?: boolean;
  app_version: string;
};

export class LicenseError extends Error {
  code: string;
  constructor(message: string, code: string) { super(message); this.code = code; }
}

export const LICENSE_REQUIRED_EVENT = 'studio-license-required';

async function call<T = LicenseStatus>(path: string, method = 'GET', body?: unknown): Promise<T> {
  const response = await fetch(`/api/v1/license${path}`, {
    method, credentials: 'same-origin',
    headers: { 'X-Studio-Frontend': window.location.origin, ...(body ? { 'Content-Type': 'application/json' } : {}) },
    body: body ? JSON.stringify(body) : undefined,
  });
  const data = await response.json().catch(() => ({ error: `Request failed (${response.status})` }));
  if (!response.ok) throw new LicenseError(data.error || `Request failed (${response.status})`, data.code || 'error');
  return data as T;
}

export const getLicenseStatus = () => call('/status');
/** email/code only when the server asked for a confirmed email (a shared office, a reused PC). */
export const startTrial = (email?: string, code?: string) => call('/trial', 'POST', { email, code });
/** "sent": a 6-digit code is on its way; "no_code_needed": mail is unavailable, just continue. */
export const requestTrialCode = (email: string) =>
  call<{ status: 'sent' | 'no_code_needed' }>('/trial/email-code', 'POST', { email });
export const checkOnline = () => call('/check-online', 'POST', {});
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
