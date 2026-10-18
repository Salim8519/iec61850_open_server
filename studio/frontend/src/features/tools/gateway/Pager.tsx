/** Previous/Next under a long list; shown only when there is more than one page. */
export function Pager({ count, page, size, setPage, noun }: {
  count: number; page: number; size: number; setPage: (page: number) => void; noun: string;
}) {
  const pages = Math.max(1, Math.ceil(count / size));
  const current = Math.min(page, pages - 1);
  if (pages <= 1) return null;
  return <div className="pager"><span>{count.toLocaleString()} {noun} · page {current + 1} of {pages}</span>
    <span className="row"><button disabled={current === 0} onClick={() => setPage(current - 1)}>Previous</button>
      <button disabled={current >= pages - 1} onClick={() => setPage(current + 1)}>Next</button></span></div>;
}
