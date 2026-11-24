import type { Metadata } from "next";
import KetentuanClient from "@/components/features/ketentuan/ketentuan-client";
import { auth } from "@/auth";
import { publicFont } from "@/lib/public-font";

export const metadata: Metadata = {
  title: "Ketentuan Penggunaan",
  description:
    "Ketentuan dan syarat penggunaan platform Laci Digital untuk PC IPNU IPPNU Kabupaten Magetan. Baca sebelum menggunakan layanan kami.",
};

export default async function KetentuanPage() {
  const session = await auth();
  return (
    <div className={publicFont.className}>
      <KetentuanClient isAuthenticated={Boolean(session?.user)} />
    </div>
  );
}
