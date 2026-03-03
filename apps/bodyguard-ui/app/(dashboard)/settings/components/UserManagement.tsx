"use client";

import { useState, useEffect } from 'react';

interface User {
  id: string;
  username: string;
  role: 'admin' | 'user' | 'guest';
  created_at: string;
  created_by: string;
  last_login: string;
  active: boolean;
}

interface NewUser {
  username: string;
  password: string;
  role: 'admin' | 'user' | 'guest';
}

export default function UserManagement() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAddModal, setShowAddModal] = useState(false);
  const [newUser, setNewUser] = useState<NewUser>({
    username: '',
    password: '',
    role: 'user'
  });
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  // Fetch users
  const fetchUsers = async () => {
    try {
      const res = await fetch('/api/v1/users');
      if (res.ok) {
        const data = await res.json();
        setUsers(data.users || []);
      }
    } catch (err) {
      console.error('Failed to fetch users:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUsers();
  }, []);

  // Add user
  const handleAddUser = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    if (!newUser.username || !newUser.password) {
      setError('Username and password are required');
      return;
    }

    if (newUser.password.length < 6) {
      setError('Password must be at least 6 characters');
      return;
    }

    try {
      const res = await fetch('/api/v1/users', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newUser)
      });

      const data = await res.json();

      if (res.ok) {
        setSuccess('User created successfully!');
        setNewUser({ username: '', password: '', role: 'user' });
        setShowAddModal(false);
        fetchUsers();
        setTimeout(() => setSuccess(''), 3000);
      } else {
        setError(data.error || 'Failed to create user');
      }
    } catch (err) {
      setError('Failed to connect to server');
    }
  };

  // Delete user
  const handleDeleteUser = async (userId: string, username: string) => {
    if (!confirm(`Are you sure you want to delete user "${username}"?`)) {
      return;
    }

    try {
      const res = await fetch(`/api/v1/users/${userId}`, {
        method: 'DELETE'
      });

      if (res.ok) {
        setSuccess('User deleted successfully!');
        fetchUsers();
        setTimeout(() => setSuccess(''), 3000);
      } else {
        const data = await res.json();
        setError(data.error || 'Failed to delete user');
      }
    } catch (err) {
      setError('Failed to connect to server');
    }
  };

  // Toggle user active status
  const handleToggleActive = async (userId: string, currentStatus: boolean) => {
    try {
      const res = await fetch(`/api/v1/users/${userId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ active: !currentStatus })
      });

      if (res.ok) {
        setSuccess(`User ${currentStatus ? 'disabled' : 'enabled'} successfully!`);
        fetchUsers();
        setTimeout(() => setSuccess(''), 3000);
      } else {
        const data = await res.json();
        setError(data.error || 'Failed to update user');
      }
    } catch (err) {
      setError('Failed to connect to server');
    }
  };

  // Change user role
  const handleChangeRole = async (userId: string, newRole: string) => {
    try {
      const res = await fetch(`/api/v1/users/${userId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ role: newRole })
      });

      if (res.ok) {
        setSuccess('User role updated successfully!');
        fetchUsers();
        setTimeout(() => setSuccess(''), 3000);
      } else {
        const data = await res.json();
        setError(data.error || 'Failed to update user role');
      }
    } catch (err) {
      setError('Failed to connect to server');
    }
  };

  const getRoleBadge = (role: string) => {
    const styles = {
      admin: 'bg-red-500/20 text-red-400',
      user: 'bg-blue-500/20 text-blue-400',
      guest: 'bg-gray-500/20 text-gray-400'
    };
    return (
      <span className={`px-2 py-1 rounded text-xs font-medium ${styles[role as keyof typeof styles]}`}>
        {role.toUpperCase()}
      </span>
    );
  };

  if (loading) {
    return (
      <div className="card">
        <h2 className="text-xl font-semibold text-white mb-4">User Management</h2>
        <div className="text-slate-400">Loading...</div>
      </div>
    );
  }

  return (
    <div className="card">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold text-white">User Management</h2>
        <button
          onClick={() => setShowAddModal(true)}
          className="btn btn-primary text-sm"
        >
          Add User
        </button>
      </div>

      {/* Success/Error Messages */}
      {success && (
        <div className="mb-4 p-3 bg-green-500/20 border border-green-500/50 rounded-lg text-green-400 text-sm">
          {success}
        </div>
      )}
      {error && (
        <div className="mb-4 p-3 bg-red-500/20 border border-red-500/50 rounded-lg text-red-400 text-sm">
          {error}
        </div>
      )}

      {/* Users Table */}
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="border-b border-slate-700">
              <th className="text-left py-3 px-4 text-slate-400 font-medium text-sm">Username</th>
              <th className="text-left py-3 px-4 text-slate-400 font-medium text-sm">Role</th>
              <th className="text-left py-3 px-4 text-slate-400 font-medium text-sm">Status</th>
              <th className="text-left py-3 px-4 text-slate-400 font-medium text-sm">Created</th>
              <th className="text-left py-3 px-4 text-slate-400 font-medium text-sm">Actions</th>
            </tr>
          </thead>
          <tbody>
            {users.map((user) => (
              <tr key={user.id} className="border-b border-slate-800 hover:bg-slate-800/50">
                <td className="py-3 px-4">
                  <div className="text-white font-medium">{user.username}</div>
                  <div className="text-slate-500 text-xs">{user.id}</div>
                </td>
                <td className="py-3 px-4">
                  <select
                    value={user.role}
                    onChange={(e) => handleChangeRole(user.id, e.target.value)}
                    className="bg-slate-700 text-white text-sm rounded px-2 py-1 border border-slate-600"
                  >
                    <option value="admin">Admin</option>
                    <option value="user">User</option>
                    <option value="guest">Guest</option>
                  </select>
                </td>
                <td className="py-3 px-4">
                  {getRoleBadge(user.role)}
                  <span className={`ml-2 px-2 py-1 rounded text-xs font-medium ${
                    user.active ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'
                  }`}>
                    {user.active ? 'Active' : 'Disabled'}
                  </span>
                </td>
                <td className="py-3 px-4 text-slate-400 text-sm">
                  {new Date(user.created_at).toLocaleDateString()}
                </td>
                <td className="py-3 px-4">
                  <div className="flex gap-2">
                    <button
                      onClick={() => handleToggleActive(user.id, user.active)}
                      className={`text-xs px-2 py-1 rounded ${
                        user.active
                          ? 'bg-orange-500/20 text-orange-400 hover:bg-orange-500/30'
                          : 'bg-green-500/20 text-green-400 hover:bg-green-500/30'
                      }`}
                    >
                      {user.active ? 'Disable' : 'Enable'}
                    </button>
                    <button
                      onClick={() => handleDeleteUser(user.id, user.username)}
                      className="text-xs px-2 py-1 rounded bg-red-500/20 text-red-400 hover:bg-red-500/30"
                    >
                      Delete
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {users.length === 0 && (
              <tr>
                <td colSpan={5} className="py-8 text-center text-slate-500">
                  No users found
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {/* Add User Modal */}
      {showAddModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-slate-800 rounded-lg p-6 w-full max-w-md">
            <h3 className="text-lg font-semibold text-white mb-4">Add New User</h3>

            <form onSubmit={handleAddUser} className="space-y-4">
              <div>
                <label className="block text-slate-400 text-sm mb-1">Username</label>
                <input
                  type="text"
                  value={newUser.username}
                  onChange={(e) => setNewUser({ ...newUser, username: e.target.value })}
                  className="w-full bg-slate-700 text-white rounded px-3 py-2 border border-slate-600 focus:border-blue-500 focus:outline-none"
                  placeholder="Enter username"
                  required
                />
              </div>

              <div>
                <label className="block text-slate-400 text-sm mb-1">Password</label>
                <input
                  type="password"
                  value={newUser.password}
                  onChange={(e) => setNewUser({ ...newUser, password: e.target.value })}
                  className="w-full bg-slate-700 text-white rounded px-3 py-2 border border-slate-600 focus:border-blue-500 focus:outline-none"
                  placeholder="Enter password (min 6 characters)"
                  required
                  minLength={6}
                />
              </div>

              <div>
                <label className="block text-slate-400 text-sm mb-1">Role</label>
                <select
                  value={newUser.role}
                  onChange={(e) => setNewUser({ ...newUser, role: e.target.value as any })}
                  className="w-full bg-slate-700 text-white rounded px-3 py-2 border border-slate-600 focus:border-blue-500 focus:outline-none"
                >
                  <option value="user">User</option>
                  <option value="admin">Admin</option>
                  <option value="guest">Guest</option>
                </select>
              </div>

              <div className="flex gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => {
                    setShowAddModal(false);
                    setError('');
                    setNewUser({ username: '', password: '', role: 'user' });
                  }}
                  className="flex-1 px-4 py-2 bg-slate-700 text-white rounded hover:bg-slate-600"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="flex-1 px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
                >
                  Create User
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
