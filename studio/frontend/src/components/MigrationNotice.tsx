import type { Project } from '../api/types';
import { CompactNotice } from './CompactNotice';

/** "Project upgraded to format …" as one compact line; the details and the recovery file stay one click away. */
export function MigrationNotice({ notice }: { notice: Project['migration_notice'] }) {
  if (!notice) return null;
  return <CompactNotice message={notice.message} detailsLabel={notice.backup ? 'Details and recovery file' : 'Details'}
    dismissLabel="Dismiss migration notice">
    <ul>{notice.details.map(item => <li key={item}>{item}</li>)}</ul>
    {notice.backup && <><p className="small" style={{ margin: 0 }}>Recovery file (the project before the upgrade):</p>
      <code>{notice.backup}</code></>}
  </CompactNotice>;
}
