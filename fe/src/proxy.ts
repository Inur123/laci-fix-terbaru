import { NextResponse, type NextRequest } from "next/server";

export default async function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const isDashboard = pathname.startsWith("/dashboard");
  const isAuthPage = pathname === "/login" || pathname === "/register";
  const hasSession = Boolean(request.cookies.get("laci_session"));

  if (isDashboard && !hasSession) return NextResponse.redirect(new URL("/login", request.url));
  if (isAuthPage && !hasSession) return NextResponse.next();

  let authData: any = null;
  if (hasSession) {
    try {
      const backend = (process.env.BACKEND_URL || "http://127.0.0.1:8080").replace(/\/$/, "");
      const response = await fetch(`${backend}/api/auth/session`, {
        headers: { cookie: request.headers.get("cookie") || "" },
        cache: "no-store",
      });
      if (response.ok) authData = await response.json();
    } catch {
      // Server components perform the definitive authentication check.
    }
  }

  if (isDashboard) {
    if (!authData?.session || !authData?.user) return NextResponse.redirect(new URL("/login", request.url));
    if (!authData.user.isActive) return NextResponse.redirect(new URL("/login?error=account_inactive", request.url));
    const verificationAllowed = pathname === "/dashboard" || pathname === "/dashboard/profile";
    if (!authData.user.emailVerified && !verificationAllowed) {
      return NextResponse.redirect(new URL("/dashboard", request.url));
    }
  } else if (isAuthPage && authData?.session && authData?.user?.isActive) {
    return NextResponse.redirect(new URL("/dashboard", request.url));
  }

  const response = NextResponse.next();
  response.headers.set("x-pathname", pathname);
  return response;
}

export const config = { matcher: ["/dashboard/:path*", "/login", "/register"] };
