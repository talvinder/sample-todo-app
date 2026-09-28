import { useEffect, useState } from 'react'

export default function App() {
  const [todos, setTodos] = useState([])
  const [title, setTitle] = useState('')
  const [source, setSource] = useState('')

  const load = async () => {
    const res = await fetch('/api/todos')
    setSource(res.headers.get('X-Cache') === 'HIT' ? 'Redis cache' : 'MySQL')
    setTodos(await res.json())
  }

  useEffect(() => { load() }, [])

  const add = async (e) => {
    e.preventDefault()
    if (!title.trim()) return
    await fetch('/api/todos', { method: 'POST', body: JSON.stringify({ title: title.trim() }) })
    setTitle('')
    load()
  }

  const toggle = async (id) => { await fetch(`/api/todos/${id}`, { method: 'PATCH' }); load() }
  const remove = async (id) => { await fetch(`/api/todos/${id}`, { method: 'DELETE' }); load() }

  return (
    <main>
      <h1>Todos</h1>
      <form onSubmit={add}>
        <input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="What needs doing?" autoFocus />
        <button>Add</button>
      </form>
      <ul>
        {todos.map((t) => (
          <li key={t.id} className={t.done ? 'done' : ''}>
            <label>
              <input type="checkbox" checked={t.done} onChange={() => toggle(t.id)} />
              <span>{t.title}</span>
            </label>
            <button className="delete" onClick={() => remove(t.id)} aria-label={`Delete ${t.title}`}>×</button>
          </li>
        ))}
      </ul>
      {todos.length === 0 && <p className="empty">Nothing to do yet.</p>}
      {source && <p className="source">Last list loaded from {source}</p>}
      <button className="refresh" onClick={load}>Reload list</button>
    </main>
  )
}
