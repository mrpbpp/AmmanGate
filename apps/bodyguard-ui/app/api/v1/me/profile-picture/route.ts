import { NextRequest, NextResponse } from "next/server";

const BACKEND_API = process.env.NEXT_PUBLIC_CORE_API || "http://127.0.0.1:8787/v1";

// Helper to get session ID for backend auth
function getSessionId(request: NextRequest): string | null {
  return request.cookies.get("session")?.value || null;
}

// Helper to get username from session_client cookie
function getUsername(request: NextRequest): string | null {
  return request.cookies.get("session_client")?.value || null;
}

// PUT /api/v1/me/profile-picture - Update user profile picture
export async function PUT(request: NextRequest) {
  try {
    const body = await request.json();
    const sessionId = getSessionId(request);

    if (!sessionId) {
      return NextResponse.json(
        { error: "Not authenticated" },
        { status: 401 }
      );
    }

    // First get current user to find their ID
    const usersResponse = await fetch(`${BACKEND_API}/users`, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        "Cookie": `session=${sessionId}`,
      },
    });

    const usersData = await usersResponse.json();
    const username = getUsername(request);
    const users = usersData.users || [];

    // Find current user
    let currentUser;
    if (username) {
      currentUser = users.find((u: any) => u.username === username);
    }
    if (!currentUser && users.length > 0) {
      currentUser = users[0];
    }

    if (!currentUser) {
      return NextResponse.json(
        { error: "User not found" },
        { status: 404 }
      );
    }

    // Update user profile picture
    const response = await fetch(`${BACKEND_API}/users/${currentUser.id}`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
        "Cookie": `session=${sessionId}`,
      },
      body: JSON.stringify(body),
    });

    const data = await response.json();

    if (!response.ok) {
      return NextResponse.json(data, { status: response.status });
    }

    return NextResponse.json(data);
  } catch (error) {
    console.error("Error updating profile picture:", error);
    return NextResponse.json(
      { error: "Failed to update profile picture" },
      { status: 500 }
    );
  }
}
