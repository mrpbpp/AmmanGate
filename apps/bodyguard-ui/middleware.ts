import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

// Force Node.js runtime for crypto module support
export const runtime = 'nodejs';

// Public paths that don't require authentication
const publicPaths = ["/login", "/api/auth/login", "/api/auth/logout"];

export async function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Allow public paths
  if (publicPaths.some((path) => pathname.startsWith(path))) {
    return NextResponse.next();
  }

  // Check for session cookie (fast check, no backend call)
  const sessionCookie = request.cookies.get("session");

  // Also check for old session cookie for migration
  const oldSessionCookie = request.cookies.get("bg_session");

  if (!sessionCookie && !oldSessionCookie) {
    // Redirect to login for protected routes
    const loginUrl = new URL("/login", request.url);
    loginUrl.searchParams.set("redirect", pathname);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    /*
     * Match all request paths except:
     * - api/auth/* (auth endpoints)
     * - api/v1/* (API proxy routes - handle their own auth)
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     * - public folder
     */
    "/((?!api/auth|api/v1|_next/static|_next/image|favicon.ico|.*\\..*|_next).*)",
  ],
};
