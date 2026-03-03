import { NextRequest, NextResponse } from "next/server";

const BACKEND_API = "http://127.0.0.1:8787/v1";

// PUT /api/v1/me/profile-picture - Update user profile picture
export async function PUT(request: NextRequest) {
  try {
    const body = await request.json();
    const sessionCookie = request.cookies.get("session");

    if (!sessionCookie?.value) {
      return NextResponse.json(
        { error: "Not authenticated" },
        { status: 401 }
      );
    }

    const response = await fetch(`${BACKEND_API}/me/profile-picture`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
        "Cookie": `session=${sessionCookie.value}`,
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
