import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api.js';
import { useAuth } from '../auth.jsx';

const permissions = {
  admin: 'You can manage every project, every task and all user accounts.',
  manager: 'You can create projects and fully manage the projects you own.',
  member: 'You can view projects you belong to and update the status of tasks assigned to you.',
};

export default function Dashboard() {
  const { user } = useAuth();
  const [stats, setStats] = useState(null);
  const [error, setError] = useState('');

  useEffect(() => {
    api('/stats').then(setStats).catch((e) => setError(e.message));
  }, []);

  if (error) return <div className="alert">{error}</div>;
  if (!stats) return <div className="muted">Loading…</div>;

  const t = stats.tasksByStatus;
  return (
    <>
      <h1>Welcome, {user.name.split(' ')[0]}</h1>
      <p className="muted">{permissions[user.role]}</p>

      <div className="stats">
        <Stat label="Projects" value={stats.projects} />
        <Stat label="To do" value={t.todo} />
        <Stat label="In progress" value={t.in_progress} />
        <Stat label="Done" value={t.done} />
        <Stat label="Overdue" value={stats.overdueTasks} warn={stats.overdueTasks > 0} />
      </div>

      <div className="card">
        <h2>My open tasks</h2>
        {stats.myOpenTasks.length === 0 ? (
          <p className="muted">Nothing assigned to you. 🎉</p>
        ) : (
          <table className="list">
            <thead><tr><th>Task</th><th>Project</th><th>Status</th><th>Priority</th><th>Due</th></tr></thead>
            <tbody>
              {stats.myOpenTasks.map((task) => (
                <tr key={task.id}>
                  <td>{task.title}</td>
                  <td><Link to={`/projects/${task.projectId}`}>{task.projectName}</Link></td>
                  <td><span className={`badge st-${task.status}`}>{task.status.replace('_', ' ')}</span></td>
                  <td><span className={`badge pr-${task.priority}`}>{task.priority}</span></td>
                  <td>{task.dueDate ? task.dueDate.slice(0, 10) : '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </>
  );
}

function Stat({ label, value, warn }) {
  return (
    <div className={`card stat ${warn ? 'warn' : ''}`}>
      <div className="stat-value">{value}</div>
      <div className="muted">{label}</div>
    </div>
  );
}
