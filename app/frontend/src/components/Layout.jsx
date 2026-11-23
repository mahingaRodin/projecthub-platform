import { NavLink, Outlet } from 'react-router-dom';
import { can, useAuth } from '../auth.jsx';

export default function Layout() {
  const { user, logout } = useAuth();
  return (
    <>
      <header className="topbar">
        <div className="brand">▦ ProjectHub</div>
        <nav>
          <NavLink to="/" end>Dashboard</NavLink>
          <NavLink to="/projects">Projects</NavLink>
          {can.manageUsers(user) && <NavLink to="/users">Users</NavLink>}
        </nav>
        <div className="who">
          <span>{user.name}</span>
          <span className={`badge role-${user.role}`}>{user.role}</span>
          <button className="ghost" onClick={logout}>Log out</button>
        </div>
      </header>
      <main className="container">
        <Outlet />
      </main>
    </>
  );
}
