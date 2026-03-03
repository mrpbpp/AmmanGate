import { NextRequest } from "next/server";

const CORE_API_URL = process.env.CORE_API_URL || "http://127.0.0.1:8787";

export async function GET(request: NextRequest) {
  const wsUrl = CORE_API_URL.replace("http://", "ws://").replace("https://", "wss://");
  const targetUrl = `${wsUrl}/v1/ws`;

  // For WebSocket, we need to upgrade the connection
  // Next.js API routes don't directly support WebSocket, so this is a placeholder
  // The client should connect directly to the WebSocket URL
  return new Response(
    JSON.stringify({
      message: "WebSocket connection should be made directly from client",
      url: targetUrl,
    }),
    {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }
  );
}
