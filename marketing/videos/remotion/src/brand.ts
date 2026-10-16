// Brand values shared with scadabench.com (website/src/styles/tokens.css).
import { staticFile } from 'remotion';

export const FPS = 30;
export const WIDTH = 1920;
export const HEIGHT = 1080;
export const FOOTAGE_WIDTH = 1600;
export const TITLE_SECONDS = 3.2;
export const END_SECONDS = 3.6;

export const COLORS = {
  night: '#070d1a', panel: '#111b2e', ink: '#eaf0f8', muted: '#93a2ba', accent: '#2be4b0', signal: '#38bdf8',
};

export const FONTS_CSS = `
@font-face { font-family: Geist; src: url('${staticFile('fonts/geist.woff2')}') format('woff2'); font-weight: 100 900; }
@font-face { font-family: 'Geist Mono'; src: url('${staticFile('fonts/geist-mono.woff2')}') format('woff2'); font-weight: 100 900; }`;

export const SANS = "Geist, 'Segoe UI', sans-serif";
export const MONO = "'Geist Mono', Consolas, monospace";

export type Caption = { start: number; end: number; text: string };
export type Click = { t: number; x: number; y: number };
export type Lesson = { slug: string; number: number; title: string; seconds: number; captions: Caption[]; clicks: Click[] };
