import { NextRequest, NextResponse } from "next/server";

const BACKEND_API = "http://127.0.0.1:8787/v1";

// PUT /api/v1/users/[id] - Update a user
export async function PUT(
  request: NextRequest,
  { params }: { params: { id: string } }
) {
  try {
    const body = await request.json();
    const sessionCookie = request.cookies.get("session");

    if (!sessionCookie?.value) {
      return NextResponse.json(
        { error: "Not authenticated" },
        { status: 401 }
      );
    }

    const response = await fetch(`${BACKEND_API}/users/${params.id}`, {
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
    console.error("Error updating user:", error);
    return NextResponse.json(
      { error: "Failed to update user" },
      { status: 500 }
    );
  }
}

// DELETE /api/v1/users/[id] - Delete a user
export async function DELETE(
  request: NextRequest,
  { params }: { params: { id: string } }
) {
  try {
    const sessionCookie = request.cookies.get("session");

    if (!sessionCookie?.value) {
      return NextResponse.json(
        { error: "Not authenticated" },
        { status: 401 }
      );
    }

    const response = await fetch(`${BACKEND_API}/users/${params.id}`, {
      method: "DELETE",
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
    console.error("Error deleting user:", error);
    return NextResponse.json(
      { error: "Failed to delete user" },
      { status: 500 }
    );
  }
}
