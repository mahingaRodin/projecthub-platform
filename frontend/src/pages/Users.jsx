import { useEffect, useState } from 'react';
import { api } from '../api.js';
import { useAuth } from '../auth.jsx';

export default function Users() {
  const { user } = useAuth();
  const [users, setUsers] = useState([]);
  const [error, setError] = useState('');

  const load = () => api('/users').then(setUsers).catch((e) => setError(e.message));
  useEffect(() => { load(); }, []);

  const run = async (fn) => {
    setError('');
    try {
      await fn();
      load();
    } catch (err) {
      setError(err.message);
    }
  };

  return (
    <>
      <h1>Users</h1>
      <p className="muted">Admins can change roles and remove accounts. Projects owned by a removed user are transferred to you.</p>
      {error && <div className="alert">{error}</div>}
      <div className="card">
        <table className="list">
          <thead><tr><th>Name</th><th>Email</th><th>Role</th><th></th></tr></thead>
          <tbody>
            {users.map((u) => (
              <tr key={u.id}>
                <td>{u.name}</td>
                <td>{u.email}</td>
                <td>
                  <select
                    value={u.role}
                    disabled={u.id === user.id}
                    onChange={(e) => run(() => api(`/users/${u.id}/role`, { method: 'PATCH', body: { role: e.target.value } }))}
                  >
                    <option value="admin">admin</option><option value="manager">manager</option><option value="member">member</option>
                  </select>
                </td>
                <td>
                  {u.id !== user.id && (
                    <button className="link danger" onClick={() => confirm(`Delete ${u.email}?`) && run(() => api(`/users/${u.id}`, { method: 'DELETE' }))}>
                      Delete
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
