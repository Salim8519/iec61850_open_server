import { Component, createElement, type ErrorInfo, type ReactNode } from 'react';
import { CRASH_COPY } from '../copy/crash.ts';

/** Catches any error while drawing the window, so it never goes blank (fix plan A15).
 *  Written without JSX so the frontend tests can render it under plain Node. */
export class ErrorBoundary extends Component<{ children: ReactNode }, { failed: boolean }> {
  state = { failed: false };

  static getDerivedStateFromError(): { failed: boolean } {
    return { failed: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('The window failed to draw', error, info.componentStack);
  }

  render(): ReactNode {
    if (!this.state.failed) return this.props.children;
    return createElement('div', { className: 'boot-wait', role: 'alert' },
      createElement('div', { className: 'stack', style: { maxWidth: 420, textAlign: 'center' } },
        createElement('strong', null, CRASH_COPY.title),
        createElement('span', { className: 'muted' }, CRASH_COPY.detail),
        createElement('button', { className: 'primary', type: 'button', onClick: () => window.location.reload() },
          CRASH_COPY.reload)));
  }
}
