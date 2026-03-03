"use client";

import { useState, useEffect } from 'react';

interface User {
  id: string;
  username: string;
  role: 'admin' | 'user' | 'guest';
  full_name: string;
  email: string;
  profile_picture: string;
  created_at: string;
  last_login: string;
}

export default function UserProfileCard() {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [showEditModal, setShowEditModal] = useState(false);
  const [showAvatarModal, setShowAvatarModal] = useState(false);

  const fetchCurrentUser = async () => {
    try {
      const res = await fetch('/api/v1/me');
      if (res.ok) {
        const data = await res.json();
        setUser(data.user);
      }
    } catch (err) {
      console.error('Failed to fetch user:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCurrentUser();
  }, []);

  const getAvatarUrl = (profilePicture: string, username: string) => {
    if (profilePicture && profilePicture.length > 100) {
      // Base64 image
      return profilePicture;
    }
    if (profilePicture && profilePicture.startsWith('http')) {
      return profilePicture;
    }
    // Default avatar
    return `https://ui-avatars.com/api/?name=${username}&background=3b82f6&color=fff&size=128`;
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
        <h2 className="text-xl font-semibold text-white mb-4">My Profile</h2>
        <div className="text-slate-400">Loading...</div>
      </div>
    );
  }

  if (!user) {
    return (
      <div className="card">
        <h2 className="text-xl font-semibold text-white mb-4">My Profile</h2>
        <div className="text-slate-400">Failed to load profile</div>
      </div>
    );
  }

  return (
    <>
      <div className="card">
        <div className="flex items-start justify-between mb-6">
          <h2 className="text-xl font-semibold text-white">My Profile</h2>
          <button
            onClick={() => setShowEditModal(true)}
            className="btn btn-secondary text-sm"
          >
            Edit Profile
          </button>
        </div>

        <div className="flex flex-col md:flex-row gap-6">
          {/* Avatar Section */}
          <div className="flex flex-col items-center gap-4">
            <div className="relative group">
              <div className="w-32 h-32 rounded-full overflow-hidden border-4 border-slate-700 bg-slate-800">
                <img
                  src={getAvatarUrl(user.profile_picture, user.username)}
                  alt={user.username}
                  className="w-full h-full object-cover"
                />
              </div>
              <button
                onClick={() => setShowAvatarModal(true)}
                className="absolute bottom-0 right-0 w-10 h-10 bg-blue-600 rounded-full flex items-center justify-center text-white opacity-0 group-hover:opacity-100 transition-opacity hover:bg-blue-700"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
              </button>
            </div>
          </div>

          {/* Profile Info */}
          <div className="flex-1 space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <p className="text-sm text-slate-400">Username</p>
                <p className="text-white font-medium">{user.username}</p>
              </div>
              <div>
                <p className="text-sm text-slate-400">Role</p>
                <div className="mt-1">{getRoleBadge(user.role)}</div>
              </div>
              <div>
                <p className="text-sm text-slate-400">Full Name</p>
                <p className="text-white font-medium">{user.full_name || 'Not set'}</p>
              </div>
              <div>
                <p className="text-sm text-slate-400">Email</p>
                <p className="text-white font-medium">{user.email || 'Not set'}</p>
              </div>
              <div>
                <p className="text-sm text-slate-400">Member Since</p>
                <p className="text-white font-medium">
                  {new Date(user.created_at).toLocaleDateString('id-ID', {
                    year: 'numeric',
                    month: 'long',
                    day: 'numeric'
                  })}
                </p>
              </div>
              <div>
                <p className="text-sm text-slate-400">Last Login</p>
                <p className="text-white font-medium">
                  {user.last_login
                    ? new Date(user.last_login).toLocaleString('id-ID')
                    : 'Never'}
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Edit Profile Modal */}
      {showEditModal && (
        <EditProfileModal
          user={user}
          onClose={() => setShowEditModal(false)}
          onSave={() => {
            setShowEditModal(false);
            fetchCurrentUser();
          }}
        />
      )}

      {/* Avatar Modal */}
      {showAvatarModal && (
        <AvatarModal
          user={user}
          onClose={() => setShowAvatarModal(false)}
          onSave={() => {
            setShowAvatarModal(false);
            fetchCurrentUser();
          }}
        />
      )}
    </>
  );
}

// Edit Profile Modal
function EditProfileModal({ user, onClose, onSave }: { user: User; onClose: () => void; onSave: () => void }) {
  const [fullName, setFullName] = useState(user.full_name || '');
  const [email, setEmail] = useState(user.email || '');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const res = await fetch('/api/v1/me/profile', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ full_name: fullName, email })
      });

      const data = await res.json();

      if (res.ok) {
        onSave();
      } else {
        setError(data.error || 'Failed to update profile');
      }
    } catch (err) {
      setError('Failed to connect to server');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-slate-800 rounded-lg p-6 w-full max-w-md">
        <h3 className="text-lg font-semibold text-white mb-4">Edit Profile</h3>

        {error && (
          <div className="mb-4 p-3 bg-red-500/20 border border-red-500/50 rounded-lg text-red-400 text-sm">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-slate-400 text-sm mb-1">Username</label>
            <input
              type="text"
              value={user.username}
              disabled
              className="w-full bg-slate-700 text-slate-500 rounded px-3 py-2 border border-slate-600 cursor-not-allowed"
            />
            <p className="text-xs text-slate-500 mt-1">Username cannot be changed</p>
          </div>

          <div>
            <label className="block text-slate-400 text-sm mb-1">Full Name</label>
            <input
              type="text"
              value={fullName}
              onChange={(e) => setFullName(e.target.value)}
              className="w-full bg-slate-700 text-white rounded px-3 py-2 border border-slate-600 focus:border-blue-500 focus:outline-none"
              placeholder="Enter your full name"
            />
          </div>

          <div>
            <label className="block text-slate-400 text-sm mb-1">Email</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full bg-slate-700 text-white rounded px-3 py-2 border border-slate-600 focus:border-blue-500 focus:outline-none"
              placeholder="Enter your email"
            />
          </div>

          <div className="flex gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2 bg-slate-700 text-white rounded hover:bg-slate-600"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="flex-1 px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
            >
              {loading ? 'Saving...' : 'Save Changes'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// Avatar Modal
function AvatarModal({ user, onClose, onSave }: { user: User; onClose: () => void; onSave: () => void }) {
  const [avatarUrl, setAvatarUrl] = useState(user.profile_picture || '');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const presetAvatars = [
    `https://ui-avatars.com/api/?name=${user.username}&background=3b82f6&color=fff`,
    `https://ui-avatars.com/api/?name=${user.username}&background=ef4444&color=fff`,
    `https://ui-avatars.com/api/?name=${user.username}&background=10b981&color=fff`,
    `https://ui-avatars.com/api/?name=${user.username}&background=f59e0b&color=fff`,
    `https://ui-avatars.com/api/?name=${user.username}&background=8b5cf6&color=fff`,
  ];

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const res = await fetch('/api/v1/me/profile-picture', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ profile_picture: avatarUrl })
      });

      const data = await res.json();

      if (res.ok) {
        onSave();
      } else {
        setError(data.error || 'Failed to update profile picture');
      }
    } catch (err) {
      setError('Failed to connect to server');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-slate-800 rounded-lg p-6 w-full max-w-md">
        <h3 className="text-lg font-semibold text-white mb-4">Change Profile Picture</h3>

        {error && (
          <div className="mb-4 p-3 bg-red-500/20 border border-red-500/50 rounded-lg text-red-400 text-sm">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Preview */}
          <div className="flex justify-center mb-4">
            <div className="w-24 h-24 rounded-full overflow-hidden border-4 border-slate-700 bg-slate-800">
              <img
                src={avatarUrl || presetAvatars[0]}
                alt="Preview"
                className="w-full h-full object-cover"
              />
            </div>
          </div>

          {/* Preset Avatars */}
          <div>
            <label className="block text-slate-400 text-sm mb-2">Choose a preset:</label>
            <div className="flex gap-2 justify-center flex-wrap">
              {presetAvatars.map((url, idx) => (
                <button
                  key={idx}
                  type="button"
                  onClick={() => setAvatarUrl(url)}
                  className={`w-12 h-12 rounded-full overflow-hidden border-2 transition-all ${
                    avatarUrl === url ? 'border-blue-500 scale-110' : 'border-slate-600 hover:border-slate-500'
                  }`}
                >
                  <img src={url} alt={`Preset ${idx + 1}`} className="w-full h-full object-cover" />
                </button>
              ))}
            </div>
          </div>

          {/* Custom URL */}
          <div>
            <label className="block text-slate-400 text-sm mb-1">Or enter image URL:</label>
            <input
              type="url"
              value={avatarUrl}
              onChange={(e) => setAvatarUrl(e.target.value)}
              className="w-full bg-slate-700 text-white rounded px-3 py-2 border border-slate-600 focus:border-blue-500 focus:outline-none"
              placeholder="https://example.com/avatar.jpg"
            />
          </div>

          <div className="flex gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2 bg-slate-700 text-white rounded hover:bg-slate-600"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="flex-1 px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
            >
              {loading ? 'Saving...' : 'Save'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
