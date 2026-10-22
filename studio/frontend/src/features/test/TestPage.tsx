import { useState } from 'react';
import type { Project } from '../../api/types';
import { SegmentedControl } from '../../ui/SegmentedControl';
import { Generators } from './Generators';
import { SequenceEditor } from './SequenceEditor';
import { TestPoints } from './TestPoints';

type View = 'points' | 'generators' | 'sequences';

export function TestPage({ project, locked, perform, prepare, refresh }: {
  project: Project; locked: boolean;
  perform: (task: () => Promise<unknown>) => Promise<void>; prepare: () => Promise<void>; refresh: () => Promise<unknown>;
}) {
  const [view, setView] = useState<View>('points');
  return <div className="content">
    <SegmentedControl label="Test tools" value={view} onChange={setView} options={[
      { value: 'points', label: 'Test points' }, { value: 'generators', label: 'Generators' },
      { value: 'sequences', label: 'Sequences' }]} />
    {view === 'points' && <TestPoints project={project} perform={perform} prepare={prepare} />}
    {view === 'generators' && <Generators project={project} perform={perform} />}
    {view === 'sequences' && <SequenceEditor ieds={project.ieds} scenarios={project.scenarios} running={locked} perform={perform}
      refresh={refresh} />}
  </div>;
}
