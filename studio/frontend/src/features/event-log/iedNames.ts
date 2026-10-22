/** IED names for Activity details (change 0177): events carry only the IED id, so the page tells this
 *  small map the names each time it reads the project. Unknown ids show as they are. */
const names = new Map<string, string>();

export function rememberIedNames(ieds: { id: string; name: string }[]) {
  names.clear();
  for (const ied of ieds) names.set(ied.id, ied.name);
}

export const iedName = (id: unknown) => names.get(String(id ?? '')) ?? String(id ?? '');
