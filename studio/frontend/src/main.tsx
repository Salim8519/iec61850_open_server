import React from 'react';
import ReactDOM from 'react-dom/client';
import { App } from './app/App';
import { LicenseBoundary } from './features/license/LicenseBoundary';
import { initPerf } from './app/perf';
import { installFit } from './app/installFit';
import { api } from './api/client';
import { liveFeed } from './api/polls';
import { installPushClient } from './api/pushClient';
import '@xyflow/react/dist/style.css';
import './styles/tokens.css';
import './styles/base.css';
import './styles/layout.css';
import './styles/workspace.css';
import './styles/canvas-interaction.css';
import './styles/fill.css';

initPerf();
// Fit to window before the first paint: small windows show the big-window layout evenly smaller.
installFit();
// Go engine: one push stream per tab replaces polling; Python keeps polling (capabilities 404).
installPushClient({ feed: liveFeed, put: body => api('/push/topics', 'PUT', body) });
ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode><LicenseBoundary><App /></LicenseBoundary></React.StrictMode>,
);
