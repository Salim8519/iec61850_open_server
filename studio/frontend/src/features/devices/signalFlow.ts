export type FlowType = 'goose' | 'report';
export type FlowSignal = { source: string; int_addr: string; desc: string };
export type FlowEdge = { id: string; type: FlowType; publisher: string; block: string; subscriber: string;
  match: 'named' | 'data'; signals: FlowSignal[] };
export type FlowBlock = { id: string; type: FlowType; ied: string; ref: string; name: string; dataset: string;
  members: string[]; app_id?: number; buffered?: boolean };
export type FlowIed = { name: string; desc: string; manufacturer: string; loaded: boolean };
export type FlowDiagnostic = { level: 'problem' | 'info'; ied: string; message: string };
export type FlowGraph = { source: string; filename: string; ieds: FlowIed[]; blocks: FlowBlock[]; edges: FlowEdge[];
  diagnostics: FlowDiagnostic[];
  summary: { ieds: number; goose_links: number; report_links: number; problems: number; infos: number } };

/** Link types are told apart by words and line style, never by colour alone. */
export const LINK_STYLE: Record<FlowType, { label: string; line: 'solid' | 'dashed'; hint: string }> = {
  goose: { label: 'GOOSE', line: 'solid', hint: 'Fast broadcast message, usually trips, blocks and interlocks.' },
  report: { label: 'Report', line: 'dashed', hint: 'Values sent to a client such as a gateway, HMI or another IED.' },
};

/** The SCD only declares a link. Without live evidence it is never shown as received. */
export function linkState(_edge: FlowEdge, live?: { state: string; checked_at: string }): string {
  return live ? `${live.state} (checked ${live.checked_at})` : 'Declared in SCD';
}

/** Latest wire verdict per GOOSE block from the station health result (only what was observed). */
export function wireVerdicts(ieds: { checks: { area: string; item: string; message: string; observed_at: string | null }[] }[]) {
  const result: Record<string, { state: string; checked_at: string }> = {};
  for (const check of ieds.flatMap(ied => ied.checks)) {
    if (check.area === 'goose_wire' && check.observed_at) result[check.item] = { state: check.message, checked_at: check.observed_at.slice(11, 19) };
  }
  return result;
}

export type FlowRow = { block: FlowBlock; edges: FlowEdge[] };

/** One row per control block that has links or matches the filter, in API order. */
export function flowRows(graph: FlowGraph, type: FlowType | 'all', query: string): FlowRow[] {
  const text = query.trim().toLowerCase();
  return graph.blocks.filter(block => type === 'all' || block.type === type).map(block => ({
    block, edges: graph.edges.filter(edge => edge.block === block.id) })).filter(row => {
    if (!text) return true;
    const words = [row.block.ied, row.block.ref, row.block.dataset, ...row.edges.flatMap(edge =>
      [edge.subscriber, ...edge.signals.flatMap(signal => [signal.source, signal.int_addr, signal.desc])])];
    return words.some(word => word.toLowerCase().includes(text));
  });
}

export const MANY_IEDS = 60;
