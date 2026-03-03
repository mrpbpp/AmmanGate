import { NextRequest, NextResponse } from "next/server";

const BACKEND_API = "http://127.0.0.1:8787/v1";

// GET /api/v1/users - List all users
export async function GET(request: NextRequest) {
  try {
    const sessionCookie = request.cookies.get("session");

    if (!sessionCookie?.value) {
      return NextResponse.json(
        { error: "Not authenticated" },
        { status: 401 }
      );
    }

    const response = await fetch(`${BACKEND_API}/users`, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        "Cookie": `session=${sessionCookie.value}`,
      },
    });

    const data = await response.json();

    if (!response.ok) {
      return NextResponse.json(data, { status: response.status });
    }

    return NextResponse.json(data);
  } catch (error) {
    console.error("Error fetching users:", error);
    return NextResponse.json(
      { error: "Failed to fetch users" },
      { status: 500 }
    );
  }
}

// POST /api/v1/users - Create a new user
export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const sessionCookie = request.cookies.get("session");

    if (!sessionCookie?.value) {
      return NextResponse.json(
        { error: "Not authenticated" },
        { status: 401 }
      );
    }

    const response = await fetch(`${BACKEND_API}/users`, {
      method: "POST",
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
    console.error("Error creating user:", error);
    return NextResponse.json(
      { error: "Failed to create user" },
      { status: 500 }
    );
  }
}
