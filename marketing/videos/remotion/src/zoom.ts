// Camera for the screen recording: ease in toward each click, hold, ease back out.
// Overlapping clicks blend their focus points, so the camera glides instead of jumping.
import type { Click } from './brand';

const RISE = [-0.9, -0.15];
const FALL = [1.1, 2.0];
export const MAX_ZOOM = 0.3;

const smooth = (x: number) => x * x * (3 - 2 * x);

function weight(dt: number): number {
  if (dt < RISE[0] || dt > FALL[1]) return 0;
  if (dt < RISE[1]) return smooth((dt - RISE[0]) / (RISE[1] - RISE[0]));
  if (dt <= FALL[0]) return 1;
  return smooth(1 - (dt - FALL[0]) / (FALL[1] - FALL[0]));
}

export function camera(clicks: Click[], seconds: number, width: number, height: number) {
  let total = 0;
  let strongest = 0;
  let x = 0;
  let y = 0;
  for (const click of clicks) {
    const w = weight(seconds - click.t);
    if (w === 0) continue;
    total += w;
    strongest = Math.max(strongest, w);
    x += click.x * w;
    y += click.y * w;
  }
  if (total === 0) return { scale: 1, originX: width / 2, originY: height / 2 };
  return { scale: 1 + MAX_ZOOM * strongest, originX: x / total, originY: y / total };
}
