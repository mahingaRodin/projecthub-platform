import { useCallback, useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { api } from '../api.js';
import { can, useAuth } from '../auth.jsx';
import MemberPicker from '../components/MemberPicker.jsx';

const COLUMNS = [
  { key: 'todo', label: 'To do' },
  { key: 'in_progress', label: 'In progress' },
  { key: 'done', label: 'Done' },
];
const emptyTask = { title: '', description: '', priority: 'medium', assigneeId: '', dueDate: '' };

export default function ProjectDetail() {
  const { id } = useParams();
  const { user } = useAuth();
  const navigate = useNavigate();
  const [project, setProject] = useState(null);
  const [tasks, setTasks] = useState([]);
  const [users, setUsers] = useState([]);
  const [taskForm, setTaskForm] = useState(emptyTask);
  const [editing, setEditing] = useState(false);
  const [edit, setEdit] = useState(null);
  const [error, setError] = useState('');
  const [newMember, setNewMember] = useState('');
  const [newOwner, setNewOwner] = useState('');

  const load = useCallback(async () => {
    try {
      const [p, t] = await Promise.all([api(`/projects/${id}`), api(`/projects/${id}/tasks`)]);
      setProject(p);
      setTasks(t);
    } catch (err) {
      setError(err.message);
    }
  }, [id]);

  useEffect(() => {
    load();
    if (can.createProject(user)) api('/users').then(setUsers).catch(() => {});
  }, [load, user]);

  if (error && !project) return <div className="alert">{error} <Link to="/projects">Back to projects</Link></div>;
  if (!project) return <div className="muted">Loading…</div>;

  const people = [project.owner, ...project.members].filter(Boolean);
  const nameOf = (uid) => people.find((p) => p.id === uid)?.name ?? 'Unassigned';
  const run = async (fn) => {
    setError('');
    try {
      await fn();
      await load();
    } catch (err) {
      setError(err.message);
    }
  };

  const createTask = (e) => {
    e.preventDefault();
    const body = { ...taskForm };
    if (!body.assigneeId) delete body.assigneeId;
    if (!body.dueDate) delete body.dueDate;
    run(async () => {
      await api(`/projects/${id}/tasks`, { method: 'POST', body });
      setTaskForm(emptyTask);
    });
  };

  const saveProject = (e) => {
    e.preventDefault();
    run(async () => {
      await api(`/projects/${id}`, { method: 'PUT', body: edit });
      setEditing(false);
    });
  };

  const deleteProject = () => {
    if (!confirm(`Delete "${project.name}" and all its tasks?`)) return;
    run(async () => {
      await api(`/projects/${id}`, { method: 'DELETE' });
      navigate('/projects');
    });
  };

  const saveMembers = (memberIds) =>
    run(() => api(`/projects/${id}`, { method: 'PUT', body: { memberIds } }));

  const addMember = (e) => {
    e.preventDefault();
    if (!newMember) return;
    saveMembers([...project.memberIds, newMember]);
    setNewMember('');
  };

  const removeMember = (member) => {
    const assigned = tasks.filter((t) => t.assigneeId === member.id && t.status !== 'done').length;
    const warning = assigned ? `\n\n${member.name} still has ${assigned} open task(s) on this project.` : '';
    if (!confirm(`Remove ${member.name} from "${project.name}"?${warning}`)) return;
    saveMembers(project.memberIds.filter((m) => m !== member.id));
  };

  const candidates = users.filter((u) => u.id !== project.ownerId && !project.memberIds.includes(u.id));
  // Only managers and admins can own a project.
  const ownerCandidates = users.filter((u) => u.id !== project.ownerId && ['manager', 'admin'].includes(u.role));

  const transferOwnership = (e) => {
    e.preventDefault();
    const target = users.find((u) => u.id === newOwner);
    if (!target) return;
    const losingControl = user.id === project.ownerId && user.role !== 'admin';
    const note = losingControl
      ? '\n\nYou will stay on the team as a member but will no longer be able to manage this project.'
      : `\n\n${project.owner?.name ?? 'The current owner'} will stay on the team as a member.`;
    if (!confirm(`Make ${target.name} the owner of "${project.name}"?${note}`)) return;
    run(() => api(`/projects/${id}`, { method: 'PUT', body: { ownerId: newOwner } }));
    setNewOwner('');
  };

  return (
    <>
      <Link to="/projects" className="small">← All projects</Link>
      <div className="page-head">
        <div>
          <h1>{project.name} <span className={`badge ps-${project.status}`}>{project.status}</span></h1>
          <p className="muted">{project.description}</p>
        </div>
        {project.canManage && (
          <div className="actions">
            <button className="ghost" onClick={() => {
              setEdit({ name: project.name, description: project.description, status: project.status, memberIds: project.memberIds });
              setEditing(!editing);
            }}>{editing ? 'Cancel' : 'Edit'}</button>
            <button className="danger" onClick={deleteProject}>Delete</button>
          </div>
        )}
      </div>
      {error && <div className="alert">{error}</div>}

      {editing && (
        <form className="card form" onSubmit={saveProject}>
          <label>Name<input value={edit.name} onChange={(e) => setEdit({ ...edit, name: e.target.value })} required /></label>
          <label>Description<textarea value={edit.description} onChange={(e) => setEdit({ ...edit, description: e.target.value })} /></label>
          <label>Status
            <select value={edit.status} onChange={(e) => setEdit({ ...edit, status: e.target.value })}>
              <option value="planned">planned</option><option value="active">active</option><option value="completed">completed</option>
            </select>
          </label>
          <div className="field-label">Members</div>
          <MemberPicker users={users} selected={edit.memberIds} exclude={project.ownerId} onChange={(memberIds) => setEdit({ ...edit, memberIds })} />
          <button>Save</button>
        </form>
      )}

      <div className="card team">
        <h2>Team</h2>
        <div className="team-list">
          {project.owner && (
            <span className="chip person">
              {project.owner.name} <span className="badge role-owner">owner</span>
            </span>
          )}
          {project.members.map((m) => (
            <span key={m.id} className="chip person">
              {m.name} <span className={`badge role-${m.role}`}>{m.role}</span>
              {project.canManage && (
                <button className="link danger" title={`Remove ${m.name}`} onClick={() => removeMember(m)}>✕</button>
              )}
            </span>
          ))}
          {project.members.length === 0 && <span className="muted small">No members yet.</span>}
        </div>
        {project.canManage && (
          <form className="add-member" onSubmit={addMember}>
            <select value={newMember} onChange={(e) => setNewMember(e.target.value)}>
              <option value="">{candidates.length ? 'Choose a user to add…' : 'Everyone is already on this project'}</option>
              {candidates.map((u) => <option key={u.id} value={u.id}>{u.name} ({u.role})</option>)}
            </select>
            <button disabled={!newMember}>Add to project</button>
          </form>
        )}
        {project.canManage && (
          <form className="add-member" onSubmit={transferOwnership}>
            <select value={newOwner} onChange={(e) => setNewOwner(e.target.value)}>
              <option value="">{ownerCandidates.length ? 'Transfer ownership to…' : 'No other manager or admin to transfer to'}</option>
              {ownerCandidates.map((u) => <option key={u.id} value={u.id}>{u.name} ({u.role})</option>)}
            </select>
            <button className="ghost" disabled={!newOwner}>Transfer ownership</button>
          </form>
        )}
        {project.canManage && (
          <p className="muted small">Only people on the team can be assigned tasks. Only managers and admins can own a project.</p>
        )}
      </div>

      {project.canManage && (
        <form className="card task-form" onSubmit={createTask}>
          <label>New task<input placeholder="Task title" value={taskForm.title} onChange={(e) => setTaskForm({ ...taskForm, title: e.target.value })} required minLength={2} /></label>
          <label>Priority
            <select value={taskForm.priority} onChange={(e) => setTaskForm({ ...taskForm, priority: e.target.value })}>
              <option value="low">low</option><option value="medium">medium</option><option value="high">high</option>
            </select>
          </label>
          <label>Assign task to
            <select value={taskForm.assigneeId} onChange={(e) => setTaskForm({ ...taskForm, assigneeId: e.target.value })}>
              <option value="">Unassigned</option>
              {people.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
            </select>
          </label>
          <label>Due date<input type="date" value={taskForm.dueDate} onChange={(e) => setTaskForm({ ...taskForm, dueDate: e.target.value })} /></label>
          <button>Add task</button>
        </form>
      )}

      <div className="board">
        {COLUMNS.map((col) => (
          <div key={col.key} className="column">
            <h3>{col.label} <span className="muted">({tasks.filter((t) => t.status === col.key).length})</span></h3>
            {tasks.filter((t) => t.status === col.key).map((t) => {
              const mayChangeStatus = project.canManage || t.assigneeId === user.id;
              return (
                <div key={t.id} className="card task">
                  <div className="row">
                    <b>{t.title}</b>
                    <span className={`badge pr-${t.priority}`}>{t.priority}</span>
                  </div>
                  <div className="small muted">
                    {nameOf(t.assigneeId)}{t.dueDate && ` · due ${t.dueDate.slice(0, 10)}`}
                  </div>
                  <div className="row">
                    <select
                      value={t.status}
                      disabled={!mayChangeStatus}
                      title={mayChangeStatus ? '' : 'Only the assignee or project owner can change this'}
                      onChange={(e) => run(() => api(`/tasks/${t.id}`, { method: 'PATCH', body: { status: e.target.value } }))}
                    >
                      {COLUMNS.map((c) => <option key={c.key} value={c.key}>{c.label}</option>)}
                    </select>
                    {project.canManage && (
                      <button className="link danger" onClick={() => run(() => api(`/tasks/${t.id}`, { method: 'DELETE' }))}>Delete</button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        ))}
      </div>
    </>
  );
}
