import type { Metadata } from "next";
import FAQClient from "@/components/features/faq/faq-client";
import { auth } from "@/auth";
import { publicFont } from "@/lib/public-font";

export const metadata: Metadata = {
  title: "FAQ - Pertanyaan Umum",
  description:
    "Temukan jawaban dari pertanyaan yang sering ditanyakan seputar Laci Digital, platform manajemen organisasi PC IPNU IPPNU Kabupaten Magetan.",
};

export default async function FAQPage() {
  const session = await auth();
  return (
    <div className={publicFont.className}>
      <FAQClient isAuthenticated={Boolean(session?.user)} />
    </div>
  );
}
