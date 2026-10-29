import { Metadata } from "next";

export const metadata: Metadata = {
  title: "Auth",
};

import RecaptchaProvider from "@/components/providers/recaptcha-provider";

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <RecaptchaProvider>
      <div className="min-h-screen bg-white antialiased">{children}</div>
    </RecaptchaProvider>
  );
}
