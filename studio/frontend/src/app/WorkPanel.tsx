import { useShownWork, workSteps, type Work } from '../state/workSteps';
import { StepList } from '../ui/StepList';

/** The panel in the middle of the window while a project opens or an SCL file is read (change
 *  0150): each step ticked as it finishes, so nobody thinks the app hangs. */
export function WorkPanel({ work }: { work: Work | null }) {
  const shown = useShownWork(work);
  if (!shown) return null;
  return <div className="work-panel-backdrop"><div className="work-panel">
    <StepList title={shown.title} steps={workSteps(shown)} />
  </div></div>;
}
