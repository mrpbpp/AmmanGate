import { NextRequest, NextResponse } from "next/server";

const BACKEND_API = process.env.BACKEND_API || "http://127.0.0.1:8787/v1";

export async function POST(request: NextRequest) {
  const response = NextResponse.json({ success: true, message: "Logged out successfully" });

  // Get the session cookie
  const sessionCookie = request.cookies.get("session");

  // Call backend to invalidate the session
  if (sessionCookie?.value) {
    try {
      await fetch(`${BACKEND_API}/auth/logout`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Cookie': `session=${sessionCookie.value}`
        }
      });
    } catch (error) {
      console.error("Backend logout error:", error);
    }
  }

  // Clear all session cookies
  response.cookies.delete("session");
  response.cookies.delete("session_client");
  response.cookies.delete("bg_session");

  return response;
}
