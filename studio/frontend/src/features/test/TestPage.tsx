import { useState } from 'react';
import type { Project, StudioEvent } from '../../api/types';
import { SegmentedControl } from '../../ui/SegmentedControl';
import { Generators } from './Generators';
import { SequenceEditor } from './SequenceEditor';
import { TestPoints } from './TestPoints';

type View = 'points' | 'generators' | 'sequences';

export function TestPage({ project, locked, values, perform, prepare }: {
  project: Project; locked: boolean; values: Record<string, unknown>; events: StudioEvent[];
  perform: (task: () => Promise<unknown>) => Promise<void>; prepare: () => Promise<void>;
}) {
  const [view, setView] = useState<View>('points');
  return <div className="content">
    <SegmentedControl label="Test tools" value={view} onChange={setView} options={[
      { value: 'points', label: 'Test points' }, { value: 'generators', label: 'Generators' },
      { value: 'sequences', label: 'Sequences' }]} />
    {view === 'points' && <TestPoints project={project} values={values} perform={perform} prepare={prepare} />}
    {view === 'generators' && <Generators project={project} values={values} perform={perform} />}
    {view === 'sequences' && <SequenceEditor ieds={project.ieds} scenarios={project.scenarios} running={locked} perform={perform} />}
  </div>;
}
