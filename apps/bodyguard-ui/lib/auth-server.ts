import { cookies } from "next/headers";

// New session cookie name (matches backend)
const SESSION_COOKIE = "session";
const BACKEND_API = "http://127.0.0.1:8787/v1";

export interface SessionUser {
  username: string;
  id: string;
  role: string;
}

// Server-side session functions for API routes
export async function createServerSession(username: string): Promise<void> {
  // This is now handled by the backend - kept for compatibility
  const now = Date.now();
  const cookieStore = await cookies();
  cookieStore.set("bg_session", `${username}:${now}:${now + (24 * 60 * 60 * 1000)}`, {
    httpOnly: false,
    sameSite: "lax",
    path: "/",
    maxAge: 24 * 60 * 60,
  });
}

export async function clearServerSession(): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.delete("bg_session");
  cookieStore.delete("session");
}

export async function getServerSession(): Promise<SessionUser | null> {
  const cookieStore = await cookies();
  const sessionCookie = cookieStore.get(SESSION_COOKIE)?.value;

  if (!sessionCookie) {
    // Fallback to old session format for migration
    const oldSession = cookieStore.get("bg_session")?.value;
    if (oldSession) {
      const parts = oldSession.split(":");
      if (parts.length >= 1) {
        return { username: parts[0], id: "unknown", role: "admin" };
      }
    }
    return null;
  }

  // Validate session with backend
  try {
    const response = await fetch(`${BACKEND_API}/me`, {
      headers: {
        "Cookie": `${SESSION_COOKIE}=${sessionCookie}`,
      },
    });

    if (response.ok) {
      const data = await response.json();
      return {
        username: data.user.username,
        id: data.user.id,
        role: data.user.role,
      };
    }
  } catch (error) {
    console.error("Session validation error:", error);
  }

  return null;
}

export async function isServerAuthenticated(): Promise<boolean> {
  try {
    const session = await getServerSession();
    return session !== null;
  } catch (error) {
    console.error("Auth check error:", error);
    return false;
  }
}

// Default admin credentials (should be changed in production)
export const DEFAULT_ADMIN = {
  username: "admin",
  password: "admin123",
};

export function verifyPassword(password: string, hash: string): boolean {
  return password === DEFAULT_ADMIN.password;
}

export function hashPassword(password: string): string {
  return password;
}
