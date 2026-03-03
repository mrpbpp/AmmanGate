import { NextRequest, NextResponse } from "next/server";

const CORE_API_URL = process.env.CORE_API_URL || "http://127.0.0.1:8787";

export async function GET(
  request: NextRequest,
  { params }: { params: { path: string[] } }
) {
  return proxyRequest(request, params.path);
}

export async function POST(
  request: NextRequest,
  { params }: { params: { path: string[] } }
) {
  return proxyRequest(request, params.path);
}

export async function PUT(
  request: NextRequest,
  { params }: { params: { path: string[] } }
) {
  return proxyRequest(request, params.path);
}

export async function DELETE(
  request: NextRequest,
  { params }: { params: { path: string[] } }
) {
  return proxyRequest(request, params.path);
}

async function proxyRequest(request: NextRequest, path: string[]) {
  // path includes "v1" already from the request path, so don't add it again
  const coreUrl = `${CORE_API_URL}/${path.join("/")}`;
  const url = new URL(request.url);
  const queryString = url.search;
  const targetUrl = `${coreUrl}${queryString}`;

  console.log(`[Proxy] ${request.method} ${targetUrl}`);

  try {
    // Get body only for non-GET requests
    let body: string | undefined = undefined;
    if (request.method !== "GET" && request.method !== "HEAD") {
      const text = await request.text();
      body = text || undefined; // Convert empty string to undefined
    }

    // Forward session cookie for authentication
    const sessionId = request.cookies.get("session")?.value;

    const headers: Record<string, string> = {
      "Content-Type": request.headers.get("content-type") || "application/json",
    };

    if (sessionId) {
      headers["Cookie"] = `session=${sessionId}`;
    }

    const response = await fetch(targetUrl, {
      method: request.method,
      headers,
      body,
    });

    // Get response text first to handle potential parsing errors
    const text = await response.text();
    let data;
    try {
      data = JSON.parse(text);
    } catch {
      // If not JSON, return as-is
      data = text;
    }

    return NextResponse.json(data, {
      status: response.status,
      statusText: response.statusText,
    });
  } catch (error: any) {
    console.error("[Proxy] Error:", error.message, "Target:", targetUrl, "Core URL:", CORE_API_URL);
    return NextResponse.json(
      { error: "Failed to connect to bodyguard-core", details: error.message },
      { status: 503 }
    );
  }
}
