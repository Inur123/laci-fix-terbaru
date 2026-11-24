"use client";

import { useState, useRef, useEffect, Suspense } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { authClient } from "@/lib/auth-client";
import { useRouter, useSearchParams } from "next/navigation";
import { checkEmailVerificationStatus } from "@/app/actions/auth-actions";
import { AuthSplitLayout } from "@/components/features/public-v2/auth-split-layout";
import { Button } from "@/components/ui/button";

function VerifyEmailContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const email = searchParams.get("email") || "";

  const [isChecking, setIsChecking] = useState(true);
  const [otp, setOtp] = useState(["", "", "", "", "", ""]);
  const [otpPending, setOtpPending] = useState(false);
  const [resendTimer, setResendTimer] = useState(0);
  const otpInputs = useRef<(HTMLInputElement | null)[]>([]);

  useEffect(() => {
    if (!email) {
      toast.error("Email tidak ditemukan. Silakan daftar ulang.");
      router.push("/register");
      return;
    }

    // Verified accounts do not need the OTP form.
    const checkStatus = async () => {
      const result = await checkEmailVerificationStatus(email);
      if (result.verified) {
        toast.info("Email Anda sudah terverifikasi sebelumnya.");
        router.push("/login?verified=true");
      } else {
        setIsChecking(false);
      }
    };

    checkStatus();
  }, [email, router]);

  useEffect(() => {
    if (resendTimer > 0) {
      const timer = setTimeout(() => setResendTimer(resendTimer - 1), 1000);
      return () => clearTimeout(timer);
    }
  }, [resendTimer]);

  const handleOtpChange = (index: number, value: string) => {
    if (!/^\d*$/.test(value)) return;

    const newOtp = [...otp];
    newOtp[index] = value.slice(-1);
    setOtp(newOtp);

    if (value && index < 5) {
      otpInputs.current[index + 1]?.focus();
    }
  };

  const handleKeyDown = (index: number, e: React.KeyboardEvent) => {
    if (e.key === "Backspace" && !otp[index] && index > 0) {
      otpInputs.current[index - 1]?.focus();
    }
  };

  const handlePaste = (e: React.ClipboardEvent) => {
    e.preventDefault();
    const data = e.clipboardData.getData("text").trim();
    if (!/^\d+$/.test(data)) return;

    const pastedCode = data.slice(0, 6).split("");
    const newOtp = [...otp];

    pastedCode.forEach((char, i) => {
      if (i < 6) newOtp[i] = char;
    });

    setOtp(newOtp);

    const nextIndex = Math.min(pastedCode.length, 5);
    otpInputs.current[nextIndex]?.focus();
  };

  const handleVerifyOtp = async (e: React.FormEvent) => {
    e.preventDefault();
    const otpCode = otp.join("");
    if (otpCode.length < 6) {
      toast.error("Silakan masukkan kode 6-digit lengkap.");
      return;
    }

    setOtpPending(true);
    try {
      const { error } = await authClient.emailOtp.verifyEmail({
        email,
        otp: otpCode,
      });

      if (error) {
        let errorMessage = error.message || "Kode verifikasi salah.";
        if (
          error.status === 429 ||
          error.code === "TOO_MANY_REQUESTS" ||
          error.message?.toLowerCase().includes("too many")
        ) {
          errorMessage =
            "Terlalu banyak percobaaan verifikasi. Mohon tunggu beberapa saat sebelum mencoba kembali.";
        } else if (error.message?.includes("Invalid OTP")) {
          errorMessage = "Kode OTP tidak valid atau sudah kedaluwarsa.";
        }
        toast.error(errorMessage);
      } else {
        router.push("/login?verified=true");
      }
    } catch (err) {
      toast.error("Gagal memverifikasi kode.");
    } finally {
      setOtpPending(false);
    }
  };

  const handeResendOtp = async () => {
    if (resendTimer > 0) return;

    try {
      const { error } = await authClient.emailOtp.sendVerificationOtp({
        email,
        type: "email-verification",
      });

      if (error) {
        let errorMessage = error.message || "Gagal mengirim ulang kode.";
        if (
          error.status === 429 ||
          error.code === "TOO_MANY_REQUESTS" ||
          error.message?.toLowerCase().includes("too many")
        ) {
          errorMessage =
            "Terlalu banyak permintaan pengiriman OTP. Mohon tunggu 1-2 menit sebelum mencoba lagi.";
        }
        toast.error(errorMessage);
      } else {
        toast.success("Kode baru telah dikirim.");
        setResendTimer(60);
      }
    } catch (err) {
      toast.error("Gagal mengirim ulang kode.");
    }
  };

  if (isChecking) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center bg-white">
        <Loader2 className="animate-spin text-green-600 h-10 w-10 mb-4" />
        <p className="text-slate-500 text-sm font-medium animate-pulse">
          Menghubungkan ke sistem...
        </p>
      </div>
    );
  }

  return (
    <AuthSplitLayout
      page="verify"
      cardTitle="Verifikasi email Anda"
      cardDescription="Masukkan kode 6 digit yang telah dikirim ke alamat email Anda."
      brandDescription="Satu langkah lagi untuk mengaktifkan akun Laci Digital dan menjaga akses organisasi tetap aman."
    >
          <p className="mb-6 break-all rounded-xl border border-emerald-100 bg-emerald-50/70 px-4 py-3 text-center text-sm font-semibold text-[#075d32]">
            {email}
          </p>

          <form onSubmit={handleVerifyOtp} className="space-y-6">
            <fieldset>
              <legend className="sr-only">Kode verifikasi 6 digit</legend>
              <div className="flex justify-center gap-2 sm:gap-3">
              {otp.map((digit, index) => (
                <input
                  key={index}
                  ref={(el) => {
                    otpInputs.current[index] = el;
                  }}
                  type="text"
                  inputMode="numeric"
                  autoComplete={index === 0 ? "one-time-code" : "off"}
                  aria-label={`Digit kode verifikasi ${index + 1}`}
                  maxLength={1}
                  value={digit}
                  onChange={(e) => handleOtpChange(index, e.target.value)}
                  onKeyDown={(e) => handleKeyDown(index, e)}
                  onPaste={handlePaste}
                  className="h-12 w-10 rounded-xl border border-solid border-slate-200/80 bg-white text-center text-xl font-bold text-[#091526] shadow-none outline-none transition-colors hover:border-slate-300 focus:border-[#0a8744] focus:ring-0 sm:h-14 sm:w-12"
                />
              ))}
              </div>
            </fieldset>

            <div className="space-y-4">
              <Button
                type="submit"
                disabled={otpPending}
                className="h-12 w-full rounded-xl bg-[#0a8744] font-bold text-white shadow-sm shadow-emerald-900/10 hover:bg-[#086d37] focus-visible:ring-[#0a8744]/30"
              >
                {otpPending ? (
                  <Loader2 className="animate-spin" />
                ) : (
                  "Verifikasi Akun"
                )}
              </Button>

              <div className="flex flex-col items-center gap-4">
                <button
                  type="button"
                  onClick={handeResendOtp}
                  disabled={resendTimer > 0}
                  className="min-h-11 rounded-lg px-3 text-sm font-semibold text-[#0a8744] transition-colors hover:bg-emerald-50 hover:text-[#075d32] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#0a8744] disabled:cursor-not-allowed disabled:text-slate-400"
                >
                  {resendTimer > 0
                    ? `Kirim ulang kode dalam ${resendTimer} detik`
                    : "Kirim ulang kode verifikasi"}
                </button>
              </div>
            </div>
          </form>
    </AuthSplitLayout>
  );
}

export default function VerifyEmailForm() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen flex items-center justify-center">
          <Loader2 className="animate-spin text-green-600 h-12 w-12" />
        </div>
      }
    >
      <VerifyEmailContent />
    </Suspense>
  );
}
