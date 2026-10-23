import { api } from './client';

/** GOOSE messages of IED components and GOOSE links (ADR-0112). Go decides everything here: which
 * values a message holds (functional constraint aware), which IEDs "Add all GOOSE IEDs" places and
 * where, and whether a send is allowed. The browser only shows the answers and sends commands. */
export type GooseValueType = 'onoff' | 'position' | 'number';
export type GooseLeaf = { ref: string; kind: string; fc: string; value_type: GooseValueType | null; writable: boolean;
  sendable: boolean; description: string; value: unknown };
export type GooseMember = { position: number; text: string; fc: string; refs: GooseLeaf[] };
export type GooseMessage = { gocb_ref: string; name: string; dataset: string; go_id: string; app_id: number | null;
  conf_rev: number | null; subscribers: string[]; complete: boolean; members: GooseMember[] };
export type GooseReceive = { gocb_ref: string | null; publisher: string; block: string; status: string };
/** GET /ieds/{id}/goose/messages. `why_not` says in words why nothing can be sent now ("" when it can). */
export type IedMessages = { ied_id: string; name: string; role: string; running: boolean; can_send: boolean;
  why_not: string; messages: GooseMessage[]; receives: GooseReceive[] };
/** GET /goose/members: every IED's messages with their exact leaf references. */
export type GooseMembers = { ieds: Record<string, Record<string, string[]>> };
export type GooseProposal = { nodes: { ied_id: string; name: string; position: { x: number; y: number } }[] };
export type GooseSendBody = { action: 'open' | 'closed' | 'on' | 'off' | 'value'; value?: number }
  | { gocb_ref: string; ref: string; value: boolean | number };
export type GooseSendAnswer = { node_id: string; ied_id: string; gocb_ref: string; action: string;
  applied: Record<string, unknown>; correlation_id: string };

export const GOOSE_MEMBERS = '/goose/members';
export const messagesPath = (iedId: string) => `/ieds/${encodeURIComponent(iedId)}/goose/messages`;
export const getIedMessages = (iedId: string) => api<IedMessages>(messagesPath(iedId));
export const getGooseMembers = () => api<GooseMembers>(GOOSE_MEMBERS);
export const getGooseProposal = () => api<GooseProposal>('/diagram/goose-ieds');
export const sendGoose = (nodeId: string, body: GooseSendBody) =>
  api<GooseSendAnswer>(`/equipment/${encodeURIComponent(nodeId)}/goose`, 'POST', body);
