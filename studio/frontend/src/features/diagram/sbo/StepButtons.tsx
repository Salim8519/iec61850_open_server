import { SBO_COPY } from '../../../copy/selectOperate';
import './sbo.css';

export type StepDirection = 'lower' | 'higher';

/** ▼ Lower / ▲ Raise for a step command (BSC or ISC, a tap changer). The arrow and the word both say
 *  the direction; the caller decides whether a press selects first or sends at once (ADR-0108). */
export function StepButtons({ disabled, title, label, press }: {
  disabled: boolean; title?: string; label: string; press: (direction: StepDirection) => void;
}) {
  return <span className="sbo-steps" role="group" aria-label={label}>
    <button type="button" disabled={disabled} title={title || SBO_COPY.lowerHint} onClick={() => press('lower')}>
      <span aria-hidden="true">▼</span>{SBO_COPY.lower}</button>
    <button type="button" disabled={disabled} title={title || SBO_COPY.raiseHint} onClick={() => press('higher')}>
      <span aria-hidden="true">▲</span>{SBO_COPY.raise}</button>
  </span>;
}

/** The words of a step value: "Raise" for higher, "Lower" for lower. */
export const stepWord = (direction: StepDirection): string => direction === 'higher' ? SBO_COPY.raise : SBO_COPY.lower;
