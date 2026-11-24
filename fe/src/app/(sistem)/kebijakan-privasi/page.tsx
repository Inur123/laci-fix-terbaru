import type { Metadata } from "next";
import KebijakanPrivasiClient from "@/components/features/kebijakan-privasi/kebijakan-privasi-client";
import { auth } from "@/auth";
import { publicFont } from "@/lib/public-font";

export const metadata: Metadata = {
  title: "Kebijakan Privasi",
  description:
    "Kebijakan privasi platform Laci Digital. Pelajari bagaimana data Anda dikumpulkan, digunakan, dan dilindungi oleh PC IPNU IPPNU Kabupaten Magetan.",
};

export default async function KebijakanPrivasiPage() {
  const session = await auth();
  return (
    <div className={publicFont.className}>
      <KebijakanPrivasiClient isAuthenticated={Boolean(session?.user)} />
    </div>
  );
}
