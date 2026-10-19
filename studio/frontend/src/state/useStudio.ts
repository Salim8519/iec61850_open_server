import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { api, connectSession, upload } from '../api/client';
import type { Diagram, Project, StudioEvent } from '../api/types';
import type { ImportPreview } from '../features/ieds/ImportSelection';
import { createHistory, recordDiagram, sameEngineering, undoDiagram, redoDiagram, type DiagramHistory } from './diagramHistory';
import { EquipmentValueStore } from '../equipment/liveValues';
import { EnergizationStore, type Energization } from '../equipment/energization';
import { SignalLayerStore } from '../equipment/signalLayerStore';

export function useStudio() {
  const [project, setProject] = useState<Project | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  // What the running task is doing, in words, so a slow action never looks frozen (change 0105).
  const [working, setWorking] = useState('');
  const [events, setEvents] = useState<StudioEvent[]>([]);
  const [saveState, setSaveState] = useState('All changes saved');
  const [values, setValues] = useState<Record<string, unknown>>({});
  const [importPreview, setImportPreview] = useState<ImportPreview | null>(null);
  const [history, setHistory] = useState<DiagramHistory | null>(null);
  const equipmentValues = useMemo(() => new EquipmentValueStore(), []);
  const energization = useMemo(() => new EnergizationStore(), []);
  const signalLayers = useMemo(() => new SignalLayerStore(), []);
  const historyRef = useRef<DiagramHistory | null>(null);
  const historyIdentity = useRef('');
  const tabId = useRef(crypto.randomUUID());
  const current = useRef<Project | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const saveQueue = useRef<Promise<void>>(Promise.resolve());
  const saveGeneration = useRef(0);
  const apply = useCallback((value: Project) => {
    const identity = `${value.session_id}:${value.id}:${value.ieds.map(ied => ied.id).join(',')}`;
    if (identity !== historyIdentity.current) {
      historyIdentity.current = identity;
      historyRef.current = createHistory(value.diagram); setHistory(historyRef.current);
    }
    current.current = value; setProject(value);
  }, []);
  const refresh = useCallback(async () => { apply(await api<Project>('/project')); }, [apply]);
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
        historyIdentity.current = '';
        await refresh();
        throw reason;
      }
    });
    await saveQueue.current;
    if (generation === saveGeneration.current) setSaveState('All changes saved');
  }, [refresh]);

  useEffect(() => {
    let disposed = false;
    let stream: EventSource | null = null;
    connectSession().then(refresh).then(() => {
      if (disposed) return;
      stream = new EventSource('/api/v1/events');
      stream.onopen = () => {
        Promise.all([refresh(), hydrateEquipment()]).catch(reason => setError(String(reason.message)));
      };
      stream.onmessage = message => {
        const event = JSON.parse(message.data) as StudioEvent;
        // Per-second MMS client summaries feed the SLD layer only, not Activity (spec signal-layers).
        if (event.kind !== 'mms_access') setEvents(items => [...items.slice(-499), event]);
        signalLayers.handle(event, performance.now());
        if (event.kind === 'signal' || event.kind === 'control_applied') {
          const ref = event.status_ref || event.ref;
          setValues(items => ({ ...items, [`${event.ied_id}:${ref}`]: event.value,
            ...Object.fromEntries(Object.entries((event.updates || {}) as Record<string, unknown>)
              .map(([reference, value]) => [`${event.ied_id}:${reference}`, value])) }));
        }
        if (event.kind === 'equipment_state' && event.node_id) {
          equipmentValues.set(event.node_id, event.value);
        }
        if (event.kind === 'energization') {
          energization.set({ nodes: event.nodes as Energization['nodes'], edges: event.edges as Energization['edges'],
            hints: event.hints as Energization['hints'] });
        }
        if (event.kind === 'project_changed') {
          if (event.source === tabId.current && current.current) {
            const updated = { ...current.current, revision: Number(event.revision) };
            current.current = updated; setProject(updated);
          } else {
            saveGeneration.current += 1;
            if (timer.current) clearTimeout(timer.current);
            timer.current = null;
            historyIdentity.current = '';
            setSaveState('All changes saved');
            refresh().catch(reason => setError(String(reason.message)));
          }
        }
        if (event.kind === 'runtime' || event.kind === 'resync' || event.kind === 'worker_failed' || event.kind === 'live_status') {
          if (event.kind === 'runtime' && event.state === 'stopped') {
            setValues({}); equipmentValues.clear(); energization.clear(); signalLayers.stopped();
          }
          refresh().catch(reason => setError(String(reason.message)));
          if (event.kind === 'resync') {
            hydrateEquipment().catch(reason => setError(String(reason.message)));
          }
        }
      };
    }).catch(reason => setError(reason.message));
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
    if (!navigating && historyRef.current) {
      historyRef.current = recordDiagram(historyRef.current, diagram, group);
      setHistory(historyRef.current);
    }
    const unchanged = sameEngineering(diagram, current.current.diagram);
    apply({ ...current.current, diagram });
    if (unchanged) return;
    setSaveState('Saving...');
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => saveDiagram().catch(reason => {
      setSaveState('Not saved'); setError(reason.message);
    }), 400);
  }, [apply, saveDiagram]);

  const navigateHistory = useCallback((direction: 'undo' | 'redo') => {
    if (!historyRef.current || current.current?.runtime !== 'stopped') return;
    historyRef.current = (direction === 'undo' ? undoDiagram : redoDiagram)(historyRef.current);
    setHistory(historyRef.current); updateDiagram(historyRef.current.present, undefined, true);
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
  const importFile = (file: File) => perform(async () => {
    await saveDiagram();
    if (file.name.toLowerCase().endsWith('.subsim')) {
      await upload('/project/open', file); setValues({}); equipmentValues.clear(); energization.clear();
      historyIdentity.current = '';
    } else {
      const preview = await upload('/scl/preview', file) as Omit<ImportPreview, 'file'>;
      setImportPreview({ ...preview, file });
    }
  }, file.name.toLowerCase().endsWith('.subsim') ? 'Opening the project…' : 'Reading the file…');
  const confirmImport = (names: string[]) => perform(async () => {
    if (!importPreview) return;
    const form = new FormData();
    form.append('file', importPreview.file);
    form.append('selected_names', JSON.stringify(names));
    await api('/scl/import', 'POST', form);
    setImportPreview(null); setValues({});
  }, 'Importing the IEDs…');
  return { project, error, setError, busy, working, events, values, equipmentValues, energization, signalLayers, saveState, updateDiagram,
    canUndo: !!history?.past.length, canRedo: !!history?.future.length,
    undo, redo,
    importPreview, confirmImport, cancelImport: () => { setImportPreview(null); setError(''); },
    runtime, addTemplate, importFile, perform, refresh, prepare: saveDiagram,
    save: () => perform(async () => { await saveDiagram(); await api('/project/save', 'POST'); }),
  };
}
