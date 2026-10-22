import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { api, connectSession, upload } from '../api/client';
import type { Diagram, Project, StudioEvent } from '../api/types';
import type { ImportPreview } from '../features/ieds/ImportSelection';
import { sameEngineering, type DiagramHistory } from './diagramHistory';
import { adoptProject, forgetHistory, NO_HISTORY, recordEdit, stepHistory, type HistorySession } from './historySession';
import { classifyProjectChange, seenInRead } from './projectChange';
import { EquipmentValueStore } from '../equipment/liveValues';
import { EnergizationStore, type Energization } from '../equipment/energization';
import { SignalLayerStore } from '../equipment/signalLayerStore';
import { coalesce } from './coalesce';
import { deliverLive, openEventStream, RELOADS, type StudioStream } from './eventStream';
import { liveData } from './liveData';
import { useWork } from './workSteps';
import { fileKind, OLD_PROJECT_MESSAGE } from './projectFile';

export function useStudio() {
  const [project, setProject] = useState<Project | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const steps = useWork();
  // What the running task is doing, in words, so a slow action never looks frozen (change 0105).
  const [working, setWorking] = useState('');
  const [saveState, setSaveState] = useState('All changes saved');
  const [importPreview, setImportPreview] = useState<ImportPreview | null>(null);
  const [history, setHistory] = useState<DiagramHistory | null>(null);
  const equipmentValues = useMemo(() => new EquipmentValueStore(), []);
  const energization = useMemo(() => new EnergizationStore(), []);
  const signalLayers = useMemo(() => new SignalLayerStore(), []);
  // The Undo history and the project it belongs to (historySession.ts).
  const session = useRef<HistorySession>(NO_HISTORY);
  const tabId = useRef(crypto.randomUUID());
  const current = useRef<Project | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const saveQueue = useRef<Promise<void>>(Promise.resolve());
  const saveGeneration = useRef(0);
  const apply = useCallback((value: Project) => {
    const adopted = adoptProject(session.current, value);
    if (adopted !== session.current) { session.current = adopted; setHistory(adopted.history); }
    current.current = value; setProject(value);
  }, []);
  // One GET /project in flight and one waiting at most, however many events ask (speed lane F).
  // When the last project read started (browser clock = the backend's clock: both run on this PC).
  const readAt = useRef(0);
  const refresh = useMemo(() => coalesce(async () => { readAt.current = Date.now(); apply(await api<Project>('/project')); }), [apply]);
  const hydrateEquipment = useCallback(async () => {
    const result = await api<{ states: { node_id: string; value: unknown }[]; energization: Energization }>('/equipment/states');
    equipmentValues.replace(result.states); energization.set(result.energization);
  }, [equipmentValues, energization]);
  const saveDiagram = useCallback(async () => {
    if (timer.current) clearTimeout(timer.current);
    timer.current = null;
    if (!current.current || current.current.runtime !== 'stopped') return;
    const diagram = current.current.diagram;
    const generation = saveGeneration.current;
    saveQueue.current = saveQueue.current.catch(() => {}).then(async () => {
      if (generation !== saveGeneration.current) return;
      const active = current.current;
      if (!active || active.runtime !== 'stopped') return;
      try {
        const result = await api<{ revision: number }>('/project/diagram', 'PUT', {
          diagram, revision: active.revision, tab_id: tabId.current,
        });
        if (current.current?.session_id === active.session_id) {
          const updated = { ...current.current, revision: result.revision };
          current.current = updated; setProject(updated);
        }
      } catch (reason) {
        saveGeneration.current += 1;
        session.current = forgetHistory(session.current);
        await refresh();
        throw reason;
      }
    });
    await saveQueue.current;
    if (generation === saveGeneration.current) setSaveState('All changes saved');
  }, [refresh]);

  useEffect(() => {
    let disposed = false;
    let stream: StudioStream | null = null;
    const fail = (reason: { message?: string }) => setError(String(reason.message));
    const live = liveData();
    const targets = { values: live.values, log: live.log, layers: signalLayers, equipment: equipmentValues,
      energize: (event: StudioEvent) => energization.set({ nodes: event.nodes as Energization['nodes'],
        edges: event.edges as Energization['edges'], hints: event.hints as Energization['hints'] }) };
    // Startup reads the project once, at once (the stream's first bytes can take a second). The bus
    // replays its events on connect, so changes made meanwhile still ask again (coalesced to one).
    // A reconnect may have missed events, so every later open reads the project again.
    let opened = 0;
    // An event published well before the last project read started is already in that answer
    // (the backend publishes after applying a change), so replayed history asks for nothing.
    const seen = (event: StudioEvent) => seenInRead(event, readAt.current);
    connectSession().then(() => {
      if (disposed) return;
      refresh().catch(fail);
      stream = openEventStream({ open: () => {
        opened += 1;
        Promise.all([opened > 1 ? refresh() : null, hydrateEquipment()]).catch(fail);
      }, event: event => {
        // Values, Activity, layers and equipment go to stores outside React; App state is untouched.
        deliverLive(event, targets, performance.now());
        if (event.kind === 'project_changed') {
          // The stream replays old changes of earlier tabs on every connect; the project read holds
          // them, so only a really new change by someone else drops the pending save and the history.
          const change = classifyProjectChange(event, tabId.current, readAt.current, !!current.current);
          if (change === 'own' && current.current) {
            const updated = { ...current.current, revision: Number(event.revision) };
            current.current = updated; setProject(updated);
          } else if (change === 'other') {
            saveGeneration.current += 1;
            if (timer.current) clearTimeout(timer.current);
            timer.current = null;
            session.current = forgetHistory(session.current);
            setSaveState('All changes saved');
            refresh().catch(fail);
          }
        }
        if (RELOADS.has(event.kind)) {
          if (event.kind === 'runtime' && event.state === 'stopped') {
            live.values.clear(); equipmentValues.clear(); energization.clear(); signalLayers.stopped();
          }
          if (!seen(event)) refresh().catch(fail);
          if (event.kind === 'resync') hydrateEquipment().catch(fail);
        }
      } });
    }).catch(fail);
    return () => { disposed = true; stream?.close(); if (timer.current) clearTimeout(timer.current); };
  }, [equipmentValues, energization, signalLayers, hydrateEquipment, refresh]);

  const perform = useCallback(async (task: () => Promise<unknown>, label = 'Working…') => {
    setBusy(true); setError(''); setWorking(label);
    try { await task(); await refresh(); }
    catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)); }
    finally { setBusy(false); setWorking(''); }
  }, [refresh]);

  const updateDiagram = useCallback((diagram: Diagram, group?: string, navigating = false) => {
    if (!current.current || current.current.runtime !== 'stopped') return;
    if (!navigating && session.current.history) {
      session.current = recordEdit(session.current, diagram, group);
      setHistory(session.current.history);
    }
    const unchanged = sameEngineering(diagram, current.current.diagram);
    // A local edit never goes through apply(): it cannot change which project this is, so it must
    // not be able to start a new history (that lost the first edit after opening the page).
    const edited = { ...current.current, diagram };
    current.current = edited; setProject(edited);
    if (unchanged) return;
    setSaveState('Saving...');
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => saveDiagram().catch(reason => {
      setSaveState('Not saved'); setError(reason.message);
    }), 400);
  }, [saveDiagram]);

  const navigateHistory = useCallback((direction: 'undo' | 'redo') => {
    if (!session.current.history || current.current?.runtime !== 'stopped') return;
    session.current = stepHistory(session.current, direction);
    const stepped = session.current.history;
    setHistory(stepped); if (stepped) updateDiagram(stepped.present, undefined, true);
  }, [updateDiagram]);
  const undo = useCallback(() => navigateHistory('undo'), [navigateHistory]);
  const redo = useCallback(() => navigateHistory('redo'), [navigateHistory]);

  const runtime = (action: string) => perform(async () => {
    if (current.current?.runtime === 'stopped') await saveDiagram();
    await api(`/runtime/${action}`, 'POST');
  }, action === 'start' ? 'Starting the IEDs…' : action === 'stop' ? 'Stopping the IEDs…' : 'Changing automation…');
  const addTemplate = (name: string) => perform(async () => {
    await saveDiagram(); await api('/ieds/template', 'POST', { name });
  }, 'Adding a sample IED…');
  // Opening and importing show their steps in the middle of the window (change 0150).
  const importFile = (file: File) => {
    if (fileKind(file.name) === 'old-project') { setError(OLD_PROJECT_MESSAGE); return Promise.resolve(); }
    const project = fileKind(file.name) === 'project';
    steps.begin(project ? `Opening ${file.name}` : `Reading ${file.name}`, project
      ? ['Save your diagram', 'Open the project file', 'Show the project']
      : ['Save your diagram', 'Read the IEDs in the file']);
    return perform(async () => {
      await saveDiagram(); steps.next();
      if (project) {
        await upload('/project/open', file); liveData().values.clear(); equipmentValues.clear(); energization.clear();
        session.current = forgetHistory(session.current); steps.next();
      } else {
        const preview = await upload('/scl/preview', file) as Omit<ImportPreview, 'file'>;
        setImportPreview({ ...preview, file }); steps.end();
      }
    }, project ? 'Opening the project…' : 'Reading the file…').finally(steps.end);
  };
  const confirmImport = (names: string[]) => {
    if (!importPreview) return Promise.resolve();
    steps.begin(`Importing ${names.length === 1 ? '1 IED' : `${names.length} IEDs`}`,
      ['Read and check the file, then build the IEDs', 'Show the project']);
    return perform(async () => {
      const form = new FormData();
      form.append('file', importPreview.file);
      form.append('selected_names', JSON.stringify(names));
      await api('/scl/import', 'POST', form);
      setImportPreview(null); liveData().values.clear(); steps.next();
    }, 'Importing the IEDs…').finally(steps.end);
  };
  return { project, error, setError, busy, working, work: steps.work, equipmentValues, energization, signalLayers, saveState, updateDiagram,
    canUndo: !!history?.past.length, canRedo: !!history?.future.length,
    undo, redo,
    importPreview, confirmImport, cancelImport: () => { setImportPreview(null); setError(''); },
    runtime, addTemplate, importFile, perform, refresh, prepare: saveDiagram,
    save: () => perform(async () => { await saveDiagram(); await api('/project/save', 'POST'); }),
  };
}
