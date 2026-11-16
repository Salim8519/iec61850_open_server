import { X } from 'lucide-react';
import { useEffect, useRef, type ReactNode } from 'react';
import './popup-dialog.css';

/** A command screen as a pop-up over the page (owner, 2026-11-11): the list under it stays as it was
 *  and nothing is pushed aside. Escape and the X close it, except while a command is on its way. */
export function PopupDialog({ title, label, busy = false, close, children, wide = false }: {
  title: ReactNode; label: string; busy?: boolean; close: () => void; children: ReactNode; wide?: boolean;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  useEffect(() => { dialog.current?.showModal(); }, []);
  return <dialog ref={dialog} className={`popup-dialog${wide ? ' wide' : ''}`} aria-label={label}
    onCancel={event => { event.preventDefault(); if (!busy) close(); }}>
    <div className="popup-head"><h2>{title}</h2>
      <button className="ghost icon-button" disabled={busy} onClick={close} aria-label="Close"><X size={16} /></button></div>
    <div className="popup-body">{children}</div>
  </dialog>;
}
