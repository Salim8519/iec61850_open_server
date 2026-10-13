import { Check } from 'lucide-react';
import type { ReactNode } from 'react';
import './ui.css';

/** Numbered steps with a visible "Step n of m" label and the current step's content. */
export function StepWizard({ steps, current, children, footer }: {
  steps: string[]; current: number; children: ReactNode; footer: ReactNode;
}) {
  return <div className="wizard">
    <ol className="wizard-steps" aria-label={`Step ${current + 1} of ${steps.length}`}>
      {steps.map((title, index) => <li key={title} className={index === current ? 'current' : index < current ? 'done' : ''}
        aria-current={index === current ? 'step' : undefined}>
        <span className="wizard-number">{index < current ? <Check size={12} /> : index + 1}</span>
        <span>{title}{index < current && <span className="sr-only"> (done)</span>}</span>
      </li>)}
    </ol>
    <p className="small muted wizard-progress">Step {current + 1} of {steps.length}: {steps[current]}</p>
    <div className="wizard-body">{children}</div>
    <div className="wizard-footer">{footer}</div>
  </div>;
}
