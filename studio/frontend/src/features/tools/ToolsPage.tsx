import { ArrowLeft, Radar, Radio, Router, Telescope } from 'lucide-react';
import { useState } from 'react';
import type { Activity } from '../../api/activity';
import type { Project } from '../../api/types';
import { TOOLS, TOOLS_COPY, type ToolId } from '../../copy/tools';
import { ExplorerTool } from './explorer/ExplorerTool';
import { FindTool } from './find/FindTool';
import { ScadaTool } from './gateway/ScadaTool';
import { TrafficTool } from './TrafficTool';
import './tools.css';

const ICONS: Record<ToolId, typeof Radio> = { explorer: Telescope, find: Radar, gateway: Router, traffic: Radio };

/** The Tools place: a list of tool cards, or the one open tool with a way back. */
export function ToolsPage({ project, locked, perform, tool, openTool, running, activity, runIeds }: {
  project: Project; locked: boolean; perform: (task: () => Promise<unknown>) => Promise<void>;
  tool: ToolId | null; openTool: (tool: ToolId | null) => void; running: Partial<Record<ToolId, string>>;
  activity: Activity | null; runIeds: () => void;
}) {
  const [target, setTarget] = useState<{ host: string; port: number } | undefined>();
  if (tool) {
    const card = TOOLS.find(entry => entry.id === tool)!;
    const Icon = ICONS[tool];
    return <div className={`content${tool === 'gateway' ? ' tool-fill' : ''}`}>
      <div className="row tool-header">
        <button className="ghost" onClick={() => openTool(null)}><ArrowLeft size={16} />{TOOLS_COPY.back}</button>
        <h2><Icon size={20} aria-hidden /> {card.name}</h2>
      </div>
      {tool === 'explorer' && <ExplorerTool target={target} />}
      {tool === 'find' && <FindTool openExplorer={(host, port) => { setTarget({ host, port }); openTool('explorer'); }} />}
      {tool === 'gateway' && <ScadaTool runState={project.runtime} runIeds={runIeds} activity={activity} />}
      {tool === 'traffic' && <TrafficTool project={project} locked={locked} perform={perform} />}
    </div>;
  }
  return <div className="content">
    <p className="muted" style={{ margin: 0 }}>{TOOLS_COPY.intro}</p>
    <div className="choice-grid tool-grid">{TOOLS.map(card => {
      const Icon = ICONS[card.id];
      return <button key={card.id} type="button" className="choice tool-card" data-tool={card.id}
        onClick={() => { setTarget(undefined); openTool(card.id); }}>
        <span className="row"><span className="guide-empty-icon tool-icon"><Icon size={20} /></span>
          <strong>{card.name}</strong></span>
        {running[card.id] && <span className="tool-live" data-live={card.id}><span className="live-dot" aria-hidden />
          {running[card.id]}</span>}
        <span>{card.line}</span>
        <span className="small muted"><strong className="small">{TOOLS_COPY.whatIsThis}</strong> {card.hint}</span>
      </button>;
    })}</div>
  </div>;
}
