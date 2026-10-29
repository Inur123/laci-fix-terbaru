import { Metadata } from "next";
import RegisterForm from "@/components/features/auth/register-form";
import { backendRequest } from "@/lib/backend";

export const dynamic = "force-dynamic";

export const metadata: Metadata = {
  title: "Register",
};

export default async function RegisterPage() {
  // Cek apakah ada Sekretaris Cabang yang aktif DAN memiliki periode aktif
  const readiness = await backendRequest<{ ready: boolean }>("/api/public/readiness").catch(() => ({ ready: false }));
  const isCabangReady = readiness.ready;

  return <RegisterForm isCabangReady={isCabangReady} />;
}
