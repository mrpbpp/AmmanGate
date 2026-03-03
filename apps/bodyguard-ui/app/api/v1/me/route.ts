import { NextRequest, NextResponse } from "next/server";

const BACKEND_API = "http://127.0.0.1:8787/v1";

export async function GET(request: NextRequest) {
  try {
    // Get the session cookie from the request
    const sessionCookie = request.cookies.get("session");

    if (!sessionCookie?.value) {
      return NextResponse.json(
        { error: "Not authenticated" },
        { status: 401 }
      );
    }

    // Forward the request to backend with session cookie
    const response = await fetch(`${BACKEND_API}/me`, {
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
    console.error("Error fetching user profile:", error);
    return NextResponse.json(
      { error: "Failed to fetch user profile" },
      { status: 500 }
    );
  }
}
