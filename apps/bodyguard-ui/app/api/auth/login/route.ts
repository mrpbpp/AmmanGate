import { NextRequest, NextResponse } from "next/server";

const BACKEND_API = process.env.BACKEND_API || "http://127.0.0.1:8787/v1";

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const { username, password } = body;

    console.log("[LOGIN] Attempting login for:", username);

    if (!username || !password) {
      console.log("[LOGIN] Missing username or password");
      return NextResponse.json(
        { error: "Username and password are required" },
        { status: 400 }
      );
    }

    // Call backend to authenticate
    console.log("[LOGIN] Calling backend:", `${BACKEND_API}/auth/login`);
    const backendResponse = await fetch(`${BACKEND_API}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });

    console.log("[LOGIN] Backend response status:", backendResponse.status);
    const data = await backendResponse.json();
    console.log("[LOGIN] Backend response data:", { ...data, session_id: data.session_id ? "***" : "missing" });

    if (!backendResponse.ok) {
      console.log("[LOGIN] Backend authentication failed");
      return NextResponse.json(
        { error: data.message || "Invalid credentials" },
        { status: 401 }
      );
    }

    // Create response and clear old cookies
    const response = NextResponse.json({
      success: true,
      user: { username }
    });

    response.cookies.delete("bg_session");

    // Use session_id from JSON response
    if (data.session_id) {
      console.log("[LOGIN] Setting session cookies for user:", username);

      // Set httpOnly cookie for server-side (secure)
      response.cookies.set("session", data.session_id, {
        httpOnly: true,
        sameSite: "lax",
        path: "/",
        secure: false,
        maxAge: 24 * 60 * 60,
      });

      // Also set a client-accessible cookie for client-side auth checks
      // This cookie just indicates the user is logged in (contains username only)
      response.cookies.set("session_client", username, {
        httpOnly: false,  // Allow client-side access
        sameSite: "lax",
        path: "/",
        secure: false,
        maxAge: 24 * 60 * 60,
      });

      console.log("[LOGIN] Login successful for:", username);
    } else {
      console.error("[LOGIN] No session_id in backend response:", data);
      return NextResponse.json(
        { error: "Authentication failed - no session returned" },
        { status: 500 }
      );
    }

    return response;
  } catch (error) {
    console.error("[LOGIN] Error:", error);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    );
  }
}
