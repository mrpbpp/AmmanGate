import { NextRequest, NextResponse } from "next/server";

const BACKEND_API = process.env.NEXT_PUBLIC_CORE_API || "http://127.0.0.1:8787/v1";

// Helper to get session cookie value (session ID for backend auth)
function getSessionId(request: NextRequest): string | null {
  return request.cookies.get("session")?.value || null;
}

// Helper to get username from session_client cookie
function getUsername(request: NextRequest): string | null {
  return request.cookies.get("session_client")?.value || null;
}

export async function GET(request: NextRequest) {
  try {
    const sessionId = getSessionId(request);
    const username = getUsername(request);

    if (!sessionId) {
      return NextResponse.json(
        { error: "Not authenticated" },
        { status: 401 }
      );
    }

    // Get users using session ID for authentication
    const response = await fetch(`${BACKEND_API}/users`, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        "Cookie": `session=${sessionId}`,
      },
    });

    const data = await response.json();

    if (!response.ok) {
      return NextResponse.json(data, { status: response.status });
    }

    // Find the current user from the users list
    const users = data.users || [];

    // Use username from session_client if available, otherwise first user
    let currentUser;
    if (username) {
      currentUser = users.find((u: any) => u.username === username);
    }

    // Fallback to first user if username not found
    if (!currentUser && users.length > 0) {
      currentUser = users[0];
    }

    if (!currentUser) {
      return NextResponse.json(
        { error: "User not found" },
        { status: 404 }
      );
    }

    return NextResponse.json({ user: currentUser });
  } catch (error) {
    console.error("Error fetching user profile:", error);
    return NextResponse.json(
      { error: "Failed to fetch user profile" },
      { status: 500 }
    );
  }
}
