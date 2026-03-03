"use client";

const SESSION_CLIENT_COOKIE = "session_client";  // Client-accessible cookie
const OLD_COOKIE_NAME = "bg_session";

export interface SessionUser {
  username: string;
  loginTime?: number;
  expiresAt?: number;
}

// Client-side cookie helpers
function getCookie(name: string): string | null {
  if (typeof document === "undefined") return null;
  const value = `; ${document.cookie}`;
  const parts = value.split(`; ${name}=`);
  if (parts.length === 2) return parts.pop()?.split(";").shift() || null;
  return null;
}

export async function createSession(username: string): Promise<void> {
  // Session creation is now handled by the backend
  // This function is kept for compatibility but doesn't do anything
  console.log("createSession called - sessions are now managed by the backend");
}

export async function clearSession(): Promise<void> {
  if (typeof document === "undefined") return;
  document.cookie = `session=; path=/; max-age=0`;
  document.cookie = `session_client=; path=/; max-age=0`;
  document.cookie = `${OLD_COOKIE_NAME}=; path=/; max-age=0`;
}

export async function getSession(): Promise<SessionUser | null> {
  if (typeof document === "undefined") return null;

  // Check for client-accessible session cookie
  const sessionClientCookie = getCookie(SESSION_CLIENT_COOKIE);
  if (sessionClientCookie) {
    return { username: sessionClientCookie };
  }

  // Fallback to old session format for migration
  const oldSession = getCookie(OLD_COOKIE_NAME);
  if (oldSession) {
    const parts = oldSession.split(":");
    if (parts.length >= 1) {
      return { username: parts[0] };
    }
  }

  return null;
}

export async function isAuthenticated(): Promise<boolean> {
  return (await getSession()) !== null;
}

// Default admin credentials (should be changed in production)
export const DEFAULT_ADMIN = {
  username: "admin",
  password: "admin123",
};
