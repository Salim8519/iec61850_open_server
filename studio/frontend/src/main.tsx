import React from 'react';
import ReactDOM from 'react-dom/client';
import { App } from './app/App';
import { LicenseBoundary } from './features/license/LicenseBoundary';
import { initPerf } from './app/perf';
import '@xyflow/react/dist/style.css';
import './styles/tokens.css';
import './styles/base.css';
import './styles/layout.css';
import './styles/workspace.css';
import './styles/canvas-interaction.css';
import './styles/fill.css';

initPerf();
ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode><LicenseBoundary><App /></LicenseBoundary></React.StrictMode>,
);
