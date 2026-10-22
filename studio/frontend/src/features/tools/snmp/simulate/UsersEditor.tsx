import { Eye, EyeOff, Plus, Trash2 } from 'lucide-react';
import { useState } from 'react';
import type { Level, SnmpUser } from '../../../../api/snmp';
import { AUTH_NAMES, LEVEL_HINT, LEVELS, PRIV_HINT, PRIV_NAMES, SETTINGS_COPY } from '../../../../copy/snmp';
import { HelpTip } from '../../../../ui/HelpTip';
import { SaveProblem } from '../../../../ui/SaveStatus';

const NEW_USER: SnmpUser = { name: '', level: 'authPriv', auth: 'sha256', auth_password: '', priv: 'aes128',
  priv_password: '' };

/** The v3 users of a device (or of the trap receiver): name, security and the two passwords. `typing` tells
 *  the caller that the change is typed text (a save may wait for the field to be left) and not a click.
 *  `problem` is the red line under the users when they cannot be saved. */
export function UsersEditor({ users, change, title = SETTINGS_COPY.users, problem = '' }: {
  users: SnmpUser[]; change: (users: SnmpUser[], typing?: boolean) => void; title?: string; problem?: string;
}) {
  const update = (index: number, next: SnmpUser, typing: boolean) => change(users.map((user, at) => at === index ? next : user), typing);
  return <fieldset className="snmp-fieldset"><legend>{title}</legend>
    {users.map((user, index) => <UserFields key={index} user={user} change={(next, typing) => update(index, next, typing)}
      remove={() => change(users.filter((_, at) => at !== index))} />)}
    <button type="button" onClick={() => change([...users, { ...NEW_USER, name: `user${users.length + 1}` }])}>
      <Plus size={16} />{SETTINGS_COPY.addUser}</button>
    <SaveProblem text={problem} />
  </fieldset>;
}

/** One v3 user. Passwords are hidden until Show is pressed. */
export function UserFields({ user, change, remove }: {
  user: SnmpUser; change: (user: SnmpUser, typing: boolean) => void; remove?: () => void;
}) {
  const [visible, setVisible] = useState(false);
  const set = (part: Partial<SnmpUser>, typing = false) => change({ ...user, ...part }, typing);
  const secret = (value: string | undefined, key: 'auth_password' | 'priv_password', label: string) =>
    <label>{label}<span className="row snmp-secret">
      <input type={visible ? 'text' : 'password'} value={value ?? ''} autoComplete="off" minLength={8}
        onChange={event => set({ [key]: event.target.value }, true)} />
      <button type="button" className="ghost icon" aria-label={visible ? SETTINGS_COPY.hide : SETTINGS_COPY.show}
        onClick={() => setVisible(!visible)}>{visible ? <EyeOff size={16} /> : <Eye size={16} />}</button>
    </span></label>;
  return <div className="snmp-user form-grid">
    <label>{SETTINGS_COPY.userName}<input value={user.name} onChange={event => set({ name: event.target.value }, true)}
      autoComplete="off" spellCheck={false} /></label>
    <label><span className="label-row">{SETTINGS_COPY.level}<HelpTip text={LEVEL_HINT} /></span>
      <select aria-label={SETTINGS_COPY.level} value={user.level} onChange={event => set({ level: event.target.value as Level })}>
      {(Object.keys(LEVELS) as Level[]).map(level => <option key={level} value={level}>{LEVELS[level]}</option>)}</select></label>
    {user.level !== 'noAuthNoPriv' && <>
      <label>{SETTINGS_COPY.auth}<select value={user.auth} onChange={event => set({ auth: event.target.value })}>
        {Object.entries(AUTH_NAMES).map(([key, name]) => <option key={key} value={key}>{name}</option>)}</select></label>
      {secret(user.auth_password, 'auth_password', SETTINGS_COPY.authPassword)}
    </>}
    {user.level === 'authPriv' && <>
      <label><span className="label-row">{SETTINGS_COPY.priv}<HelpTip text={PRIV_HINT} /></span>
        <select aria-label={SETTINGS_COPY.priv} value={user.priv} onChange={event => set({ priv: event.target.value })}>
          {Object.entries(PRIV_NAMES).map(([key, name]) => <option key={key} value={key}>{name}</option>)}</select></label>
      {secret(user.priv_password, 'priv_password', SETTINGS_COPY.privPassword)}
    </>}
    <p className="small muted snmp-user-hint">{user.level === 'noAuthNoPriv' ? '' : SETTINGS_COPY.passwordHint}
      {remove && <button type="button" className="ghost" onClick={remove}><Trash2 size={14} />{SETTINGS_COPY.removeUser}</button>}</p>
  </div>;
}
