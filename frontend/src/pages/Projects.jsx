import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api.js';
import { can, useAuth } from '../auth.jsx';
import MemberPicker from '../components/MemberPicker.jsx';

const empty = { name: '', description: '', status: 'planned', memberIds: [] };

export default function Projects() {
  const { user } = useAuth();
  const [projects, setProjects] = useState([]);
  const [users, setUsers] = useState([]);
  const [form, setForm] = useState(empty);
  const [showForm, setShowForm] = useState(false);
  const [error, setError] = useState('');

  const load = () => api('/projects').then(setProjects).catch((e) => setError(e.message));

  useEffect(() => {
    load();
    if (can.createProject(user)) api('/users').then(setUsers).catch(() => {});
  }, [user]);

  const create = async (e) => {
    e.preventDefault();
    setError('');
    try {
      await api('/projects', { method: 'POST', body: form });
      setForm(empty);
      setShowForm(false);
      load();
    } catch (err) {
      setError(err.message);
    }
  };

  return (
    <>
      <div className="page-head">
        <h1>Projects</h1>
        {can.createProject(user) && (
          <button onClick={() => setShowForm(!showForm)}>{showForm ? 'Cancel' : '+ New project'}</button>
        )}
      </div>
      {error && <div className="alert">{error}</div>}

      {showForm && (
        <form className="card form" onSubmit={create}>
          <label>Name<input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required minLength={2} /></label>
          <label>Description<textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} /></label>
          <label>Status
            <select value={form.status} onChange={(e) => setForm({ ...form, status: e.target.value })}>
              <option value="planned">planned</option><option value="active">active</option><option value="completed">completed</option>
            </select>
          </label>
          <div className="field-label">Members</div>
          <MemberPicker users={users} selected={form.memberIds} exclude={user.id} onChange={(memberIds) => setForm({ ...form, memberIds })} />
          <button>Create project</button>
        </form>
      )}

      {projects.length === 0 && <p className="muted">No projects visible to you yet.</p>}
      <div className="grid">
        {projects.map((p) => {
          const pct = p.taskCount ? Math.round((p.doneCount / p.taskCount) * 100) : 0;
          return (
            <Link key={p.id} to={`/projects/${p.id}`} className="card project-card">
              <div className="row">
                <h3>{p.name}</h3>
                <span className={`badge ps-${p.status}`}>{p.status}</span>
              </div>
              <p className="muted clamp">{p.description || 'No description'}</p>
              <div className="progress"><div style={{ width: `${pct}%` }} /></div>
              <div className="row small muted">
                <span>{p.doneCount}/{p.taskCount} tasks done</span>
                <span>Owner: {p.owner?.name ?? '—'}</span>
              </div>
            </Link>
          );
        })}
      </div>
    </>
  );
}
