export default function MemberPicker({ users, selected, onChange, exclude }) {
  const toggle = (id) =>
    onChange(selected.includes(id) ? selected.filter((x) => x !== id) : [...selected, id]);

  return (
    <div className="member-picker">
      {users
        .filter((u) => u.id !== exclude)
        .map((u) => (
          <label key={u.id} className="chip">
            <input type="checkbox" checked={selected.includes(u.id)} onChange={() => toggle(u.id)} />
            {u.name} <span className="muted small">({u.role})</span>
          </label>
        ))}
    </div>
  );
}
