/** The words under Open and Close after the engine answered. A test command (Test=TRUE) moves
 *  nothing, and the engine says so in `message` (change 0176); it is shown as it is. */
export function operateWords(result: { action: string; wire_value: boolean | null; message?: string }): string {
  if (result.message) return result.message;
  const sent = result.wire_value === null ? '' : ` · sent ${result.wire_value ? 1 : 0}`;
  return `${result.action[0].toUpperCase() + result.action.slice(1)} done${sent}`;
}
