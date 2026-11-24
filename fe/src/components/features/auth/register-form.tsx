"use client";

import { useState, useCallback } from "react";
import Link from "next/link";
import { User, Mail, Lock, Eye, EyeOff, Loader2 } from "lucide-react";
import { useGoogleReCaptcha } from "react-google-recaptcha-v3";
import { toast } from "sonner";
import { authClient } from "@/lib/auth-client";
import { useRouter } from "next/navigation";
import { verifyRecaptchaAction } from "@/app/actions/recaptcha-actions";
import { AuthSplitLayout } from "@/components/features/public-v2/auth-split-layout";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

import { AlertTriangle, ServerCrash, Clock } from "lucide-react";

export default function RegisterForm({
  isCabangReady = true,
}: {
  isCabangReady?: boolean;
}) {
  const { executeRecaptcha } = useGoogleReCaptcha();
  const router = useRouter();
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [agreeToTerms, setAgreeToTerms] = useState(false);

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  const getPasswordStrength = (pwd: string) => {
    if (!pwd) return { score: 0, label: "", color: "" };
    let score = 0;
    if (pwd.length >= 6) score++;
    if (pwd.length >= 10) score++;
    if (/[A-Z]/.test(pwd)) score++;
    if (/[0-9]/.test(pwd)) score++;
    if (/[^A-Za-z0-9]/.test(pwd)) score++;

    if (score <= 1) return { score: 1, label: "Lemah", color: "#ef4444" };
    if (score === 2) return { score: 2, label: "Cukup", color: "#f97316" };
    if (score === 3) return { score: 3, label: "Sedang", color: "#eab308" };
    if (score === 4) return { score: 4, label: "Kuat", color: "#22c55e" };
    return { score: 5, label: "Sangat Kuat", color: "#16a34a" };
  };

  const strength = getPasswordStrength(password);

  const handleSubmit = useCallback(
    async (e: React.FormEvent<HTMLFormElement>) => {
      e.preventDefault();

      if (password !== confirmPassword) {
        toast.error("Password tidak cocok!");
        return;
      }

      if (password.length < 6) {
        toast.error("Password minimal 6 karakter!");
        return;
      }

      if (!agreeToTerms) {
        toast.error(
          "Anda harus menyetujui Ketentuan Penggunaan dan Kebijakan Privasi.",
        );
        return;
      }

      const localCaptchaBypass = process.env.NODE_ENV !== "production";
      if (!executeRecaptcha && !localCaptchaBypass) {
        toast.error("Verifikasi keamanan belum siap. Silakan refresh halaman.");
        return;
      }

      setIsLoading(true);

      try {
        const isHuman = localCaptchaBypass
          ? true
          : await verifyRecaptchaAction(await executeRecaptcha!("register"));

        if (!isHuman) {
          toast.error("Terdeteksi sebagai aktivitas mencurigakan (Bot).");
          setIsLoading(false);
          return;
        }

        const { error: signUpError } = await authClient.signUp.email({
          name,
          email,
          password,
        });

        if (signUpError) {
          let errorMessage = "Gagal mendaftar. Silakan coba lagi.";
          if (
            signUpError.status === 429 ||
            signUpError.code === "TOO_MANY_REQUESTS" ||
            signUpError.message?.toLowerCase().includes("too many")
          ) {
            errorMessage =
              "Terlalu banyak permintaan pendaftaran. Mohon tunggu beberapa saat sebelum mencoba kembali.";
          } else if (
            signUpError.code === "USER_ALREADY_EXISTS" ||
            signUpError.message?.includes("User already exists")
          ) {
            errorMessage =
              "Email sudah terdaftar. Gunakan email lain atau silakan Masuk.";
          } else if (signUpError.code === "INVALID_PASSWORD") {
            errorMessage = "Password tidak memenuhi syarat keamanan.";
          } else if (signUpError.code === "INVALID_EMAIL") {
            errorMessage = "Format email tidak valid.";
          } else {
            errorMessage =
              signUpError.message || "Terjadi kesalahan saat pendaftaran.";
          }
          toast.error(errorMessage);
          setIsLoading(false);
          return;
        }

        const { error: otpError } =
          await authClient.emailOtp.sendVerificationOtp({
            email,
            type: "email-verification",
          });
        if (otpError) {
          toast.warning(
            "Akun berhasil dibuat, tetapi kode OTP belum terkirim. Gunakan Kirim Ulang di halaman verifikasi.",
          );
        } else {
          toast.success(
            "Akun berhasil dibuat! Kode OTP telah dikirim ke email Anda.",
          );
        }
        router.push(`/verify-email?email=${encodeURIComponent(email)}`);
      } catch (error) {
        console.error("Register error:", error);
        toast.error("Terjadi kesalahan. Silakan coba lagi.");
        setIsLoading(false);
      }
    },
    [
      name,
      email,
      password,
      confirmPassword,
      agreeToTerms,
      executeRecaptcha,
      router,
    ],
  );

  return (
    <AuthSplitLayout
      page="register"
      cardTitle="Buat akun baru"
      cardDescription={
        isCabangReady
          ? "Daftar untuk mulai menggunakan Laci Digital"
          : "Pendaftaran akun PAC sementara ditangguhkan"
      }
      alternateAction={
        <p>
          Sudah punya akun?{" "}
          <Link
            href="/login"
            className="font-semibold text-[#0a8744] underline-offset-4 hover:underline focus-visible:rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#0a8744]"
          >
            Masuk di sini
          </Link>
        </p>
      }
    >
          {!isCabangReady ? (
            <div className="space-y-6">
              <div className="bg-amber-50 border border-amber-200 rounded-xl p-6 space-y-4">
                <div className="flex items-center gap-3 text-amber-700">
                  <div className="p-2 bg-amber-100 rounded-lg">
                    <AlertTriangle size={24} />
                  </div>
                  <h3 className="font-bold">Sistem Belum Siap</h3>
                </div>
                <p className="text-sm text-amber-800 leading-relaxed">
                  Pendaftaran akun PAC baru memerlukan verifikasi dan periode
                  aktif dari Sekretaris Cabang. Saat ini, Admin Cabang belum
                  mengatur periode kepengurusan yang aktif.
                </p>
                <div className="pt-2 border-t border-amber-100">
                  <div className="flex items-center gap-2 text-xs text-amber-600 font-medium">
                    <Clock size={14} />
                    <span>
                      Silakan hubungi Sekretaris Cabang untuk aktivasi sistem.
                    </span>
                  </div>
                </div>
              </div>

              <div className="space-y-3">
                <div className="flex items-center gap-3 p-4 rounded-lg bg-slate-50 border border-slate-100">
                  <ServerCrash className="text-slate-500" size={20} />
                  <div className="text-xs text-slate-500">
                    <span className="font-bold block text-slate-700 mb-0.5">
                      Hanya 1 Admin Utama
                    </span>
                    Sistem ini dipusatkan pada satu akun Sekretaris Cabang
                    sebagai otoritas tertinggi.
                  </div>
                </div>

                <Link
                  href="/login"
                  className="flex h-12 w-full items-center justify-center rounded-xl bg-[#0a8744] px-4 text-sm font-bold text-white shadow-sm hover:bg-[#086d37] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#0a8744]/30 focus-visible:ring-offset-2"
                >
                  Masuk ke Akun Admin
                </Link>

                <p className="text-center text-xs text-slate-500">
                  Jika Anda adalah Admin Cabang, silakan masuk dan aktifkan
                  periode di Dashboard.
                </p>
              </div>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4" name="register">
              <div className="space-y-2">
                <Label htmlFor="register-name" className="font-semibold text-slate-700">
                  Nama Pimpinan
                </Label>
                <div className="relative">
                  <User aria-hidden="true" className="pointer-events-none absolute left-3.5 top-1/2 size-[18px] -translate-y-1/2 text-slate-500" />
                  <Input
                    id="register-name"
                    name="name"
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    className="h-12 rounded-xl border border-solid border-slate-200/80 bg-white pl-11 shadow-none placeholder:text-slate-500 hover:border-slate-300 focus-visible:border-[#0a8744] focus-visible:ring-0"
                    placeholder="Nama Pimpinan"
                    autoComplete="name"
                    required
                    disabled={isLoading}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="register-email" className="font-semibold text-slate-700">
                  Alamat Email
                </Label>
                <div className="relative">
                  <Mail aria-hidden="true" className="pointer-events-none absolute left-3.5 top-1/2 size-[18px] -translate-y-1/2 text-slate-500" />
                  <Input
                    id="register-email"
                    name="email"
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className="h-12 rounded-xl border border-solid border-slate-200/80 bg-white pl-11 shadow-none placeholder:text-slate-500 hover:border-slate-300 focus-visible:border-[#0a8744] focus-visible:ring-0"
                    placeholder="anda@email.com"
                    autoComplete="email"
                    required
                    disabled={isLoading}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="register-password" className="font-semibold text-slate-700">
                  Password
                </Label>
                <div className="relative">
                  <Lock aria-hidden="true" className="pointer-events-none absolute left-3.5 top-1/2 size-[18px] -translate-y-1/2 text-slate-500" />
                  <Input
                    id="register-password"
                    name="password"
                    type={showPassword ? "text" : "password"}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="h-12 rounded-xl border border-solid border-slate-200/80 bg-white pl-11 pr-12 shadow-none placeholder:text-slate-500 hover:border-slate-300 focus-visible:border-[#0a8744] focus-visible:ring-0"
                    placeholder="Minimal 6 karakter"
                    autoComplete="new-password"
                    required
                    disabled={isLoading}
                  />
                  <button
                    type="button"
                    aria-label={showPassword ? "Sembunyikan password" : "Tampilkan password"}
                    title={showPassword ? "Sembunyikan password" : "Tampilkan password"}
                    className="absolute inset-y-0 right-0 flex w-12 items-center justify-center rounded-r-xl text-slate-500 transition-colors hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-[#0a8744]"
                    onClick={() => setShowPassword(!showPassword)}
                  >
                    {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
                  </button>
                </div>
              </div>

              {password && (
                <div className="space-y-1.5 -mt-2">
                  <div className="flex gap-1">
                    {[1, 2, 3, 4, 5].map((level) => (
                      <div
                        key={level}
                        className="h-1.5 flex-1 rounded-full transition-all duration-300"
                        style={{
                          backgroundColor:
                            level <= strength.score
                              ? strength.color
                              : "#e2e8f0",
                        }}
                      />
                    ))}
                  </div>
                  <p
                    className="text-xs font-medium"
                    style={{ color: strength.color }}
                  >
                    Kekuatan password: {strength.label}
                  </p>
                </div>
              )}

              <div className="space-y-2">
                <Label htmlFor="register-password-confirm" className="font-semibold text-slate-700">
                  Konfirmasi Password
                </Label>
                <div className="relative">
                  <Lock aria-hidden="true" className="pointer-events-none absolute left-3.5 top-1/2 size-[18px] -translate-y-1/2 text-slate-500" />
                  <Input
                    id="register-password-confirm"
                    name="passwordConfirm"
                    type={showConfirmPassword ? "text" : "password"}
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    className="h-12 rounded-xl border border-solid border-slate-200/80 bg-white pl-11 pr-12 shadow-none placeholder:text-slate-500 hover:border-slate-300 focus-visible:border-[#0a8744] focus-visible:ring-0"
                    placeholder="Ulangi password"
                    autoComplete="new-password"
                    required
                    disabled={isLoading}
                  />
                  <button
                    type="button"
                    aria-label={
                      showConfirmPassword
                        ? "Sembunyikan konfirmasi password"
                        : "Tampilkan konfirmasi password"
                    }
                    title={
                      showConfirmPassword
                        ? "Sembunyikan konfirmasi password"
                        : "Tampilkan konfirmasi password"
                    }
                    className="absolute inset-y-0 right-0 flex w-12 items-center justify-center rounded-r-xl text-slate-500 transition-colors hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-[#0a8744]"
                    onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                  >
                    {showConfirmPassword ? (
                      <EyeOff size={18} />
                    ) : (
                      <Eye size={18} />
                    )}
                  </button>
                </div>
                {confirmPassword && (
                  <p
                    className="text-xs font-medium"
                    style={{
                      color:
                        confirmPassword === password ? "#16a34a" : "#ef4444",
                    }}
                  >
                    {confirmPassword === password
                      ? "✓ Password cocok"
                      : "✗ Password tidak cocok"}
                  </p>
                )}
              </div>

              <div className="flex items-start gap-2">
                <input
                  id="agreeToTerms"
                  type="checkbox"
                  checked={agreeToTerms}
                  onChange={(e) => setAgreeToTerms(e.target.checked)}
                  disabled={isLoading}
                  className="mt-0.5 h-4 w-4 shrink-0 accent-green-700 cursor-pointer"
                />
                <label
                  htmlFor="agreeToTerms"
                  className="text-xs text-slate-600 leading-relaxed cursor-pointer select-none"
                >
                  Saya telah membaca dan menyetujui{" "}
                  <Link
                    href="/ketentuan-penggunaan"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="rounded-sm font-semibold text-green-700 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#0a8744]"
                  >
                    Ketentuan Penggunaan
                  </Link>{" "}
                  dan{" "}
                  <Link
                    href="/kebijakan-privasi"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="rounded-sm font-semibold text-green-700 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#0a8744]"
                  >
                    Kebijakan Privasi
                  </Link>{" "}
                  Laci Digital.
                </label>
              </div>

              <Button
                type="submit"
                disabled={isLoading || !agreeToTerms}
                className="h-12 w-full rounded-xl bg-[#0a8744] font-bold text-white shadow-sm shadow-emerald-900/10 hover:bg-[#086d37] focus-visible:ring-[#0a8744]/30"
              >
                {isLoading ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  "Daftar Sekarang"
                )}
              </Button>
            </form>
          )}
    </AuthSplitLayout>
  );
}
