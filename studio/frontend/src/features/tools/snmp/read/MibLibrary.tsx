import { RotateCcw, Trash2, Upload } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { snmpApi, type MibModule } from '../../../../api/snmp';
import { MIB_COPY } from '../../../../copy/snmpRead';
import { HelpTip } from '../../../../ui/HelpTip';

/** The MIB library of this PC: built-in standard MIBs (removable) and the files the user added. */
export function MibLibrary({ fail }: { fail: (message: string) => void }) {
  const [modules, setModules] = useState<MibModule[]>([]);
  const [search, setSearch] = useState('');
  const [note, setNote] = useState('');
  const file = useRef<HTMLInputElement>(null);
  useEffect(() => { snmpApi.mibs().then(answer => setModules(answer.modules)).catch(reason => fail(reason.message)); }, [fail]);
  const shown = useMemo(() => modules.filter(module => module.name.toLowerCase().includes(search.trim().toLowerCase())),
    [modules, search]);
  const add = async (files: File[]) => {
    fail('');
    const added: string[] = [];
    for (const picked of files) {
      try {
        const answer = await snmpApi.importMib(picked.name, await picked.text());
        added.push(...answer.imported);
        setModules(answer.modules);
      } catch (reason) {
        fail(`${picked.name}: ${(reason as Error).message}`);
      }
    }
    if (added.length) setNote(MIB_COPY.added(added));
  };
  const remove = (name: string) => snmpApi.removeMib(name).then(answer => setModules(answer.modules))
    .catch(reason => fail(reason.message));
  return <>
    <div className="panel-toolbar">
      <button onClick={() => file.current?.click()}><Upload size={16} />{MIB_COPY.importButton}</button>
      <input ref={file} type="file" multiple hidden accept=".mib,.my,.txt,.smi" aria-label={MIB_COPY.importButton}
        onChange={event => { const picked = [...(event.target.files ?? [])]; if (picked.length) add(picked); event.target.value = ''; }} />
      <button className="ghost" onClick={() => snmpApi.restoreMibs().then(answer => setModules(answer.modules))
        .catch(reason => fail(reason.message))}><RotateCcw size={16} />{MIB_COPY.restore}</button>
      <HelpTip text={MIB_COPY.hint} label="What are MIBs?" />
      <span className="gateway-spacer" />
      <input className="snmp-mib-search" value={search} placeholder={MIB_COPY.search} aria-label={MIB_COPY.search}
        onChange={event => setSearch(event.target.value)} />
    </div>
    {note && <p className="small snmp-live snmp-note" role="status">✓ {note}</p>}
    <div className="table-scroll"><table className="gateway-points snmp-mibs"><thead><tr>
      <th>MIB</th><th>Source</th><th>Names</th><th aria-label="Remove" /></tr></thead>
      <tbody>{shown.map(module => <tr key={module.name}>
        <td><strong>{module.name}</strong>
          {module.missing.length > 0 && <span className="small gateway-warning snmp-block">{MIB_COPY.missing(module.missing)}</span>}
          {module.problems.length > 0 && <span className="small muted snmp-block" title={module.problems.join('\n')}>
            {module.problems.length} {module.problems.length === 1 ? 'definition' : 'definitions'} skipped</span>}</td>
        <td className="small">{module.builtin ? MIB_COPY.builtin : `${MIB_COPY.imported} (${module.file})`}</td>
        <td className="small">{MIB_COPY.names(module.nodes)}</td>
        <td><button className="ghost" onClick={() => remove(module.name)}><Trash2 size={14} />{MIB_COPY.remove}</button></td>
      </tr>)}</tbody></table></div>
  </>;
}
