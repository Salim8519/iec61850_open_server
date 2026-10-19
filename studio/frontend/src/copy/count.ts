/** "1 note", "2 notes": a number with its noun in the right form, for counts shown to the user. */
export function countOf(count: number, one: string, many = `${one}s`) {
  return `${count.toLocaleString()} ${count === 1 ? one : many}`;
}
