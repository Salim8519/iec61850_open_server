import { releaseOf, releasesUpTo, type Release } from '../../copy/releaseNotes';
import { CHANGELOG_URL, WHATS_NEW_COPY } from '../../copy/whatsNew';
import { Disclosure } from '../../ui/Disclosure';
import { PopupDialog } from '../../ui/PopupDialog';
import { noteParts } from './whatsNewChoice';
import './whats-new.css';

function Notes({ release }: { release: Release }) {
  return <ul className="whats-new-notes">{release.notes.map(note => {
    const { kind, text } = noteParts(note);
    return <li key={note}><span className="whats-new-kind">{kind}</span><span>{text}</span></li>;
  })}</ul>;
}

/** This version's notes first, earlier versions folded below them (change 0238). */
export function WhatsNewDialog({ current, close }: { current: string; close: () => void }) {
  const release = releaseOf(current) ?? releasesUpTo(current)[0];
  if (!release) return null;
  const earlier = releasesUpTo(current).filter(item => item.version !== release.version);
  return <PopupDialog title={WHATS_NEW_COPY.title(release.version)} label={WHATS_NEW_COPY.label} close={close}>
    <p className="whats-new-title">{release.title}</p>
    <Notes release={release} />
    {earlier.length > 0 && <Disclosure title={WHATS_NEW_COPY.earlier}>
      {earlier.map(item => <section key={item.version} className="whats-new-earlier">
        <h3>{item.version} · {item.title}</h3><Notes release={item} /></section>)}
    </Disclosure>}
    <div className="popup-actions">
      <a className="whats-new-link" href={CHANGELOG_URL} target="_blank" rel="noreferrer">{WHATS_NEW_COPY.allOnline}</a>
      <button className="primary" onClick={close} autoFocus>{WHATS_NEW_COPY.done}</button>
    </div>
  </PopupDialog>;
}
