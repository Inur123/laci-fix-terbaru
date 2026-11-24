import { Metadata } from "next";

export const metadata: Metadata = {
  title: "Auth",
};

import RecaptchaProvider from "@/components/providers/recaptcha-provider";
import { publicFont } from "@/lib/public-font";

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <RecaptchaProvider>
      <div
        className={`${publicFont.className} public-auth-shell min-h-screen bg-white antialiased`}
      >
        {children}
      </div>
    </RecaptchaProvider>
  );
}
