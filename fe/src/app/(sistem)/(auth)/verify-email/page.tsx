import { Metadata } from "next";
import VerifyEmailForm from "@/components/features/auth/verify-email-form";

export const metadata: Metadata = {
  title: "Verifikasi Email",
};

export default function VerifyEmailPage() {
  return <VerifyEmailForm />;
}
