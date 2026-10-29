import { getSession } from "@/lib/auth-session";
export type Role = "SEKRETARIS_CABANG" | "SEKRETARIS_PAC";

export async function auth() {
  const session = await getSession();
  if (!session?.user?.isActive) return null;

  return { user: session.user, expires: session.session.expiresAt };
}
