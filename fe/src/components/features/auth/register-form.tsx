"use client";

import { useState, useCallback } from "react";
import Link from "next/link";
import Image from "next/image";
import { User, Mail, Lock, Eye, EyeOff, Loader2 } from "lucide-react";
import { useGoogleReCaptcha } from "react-google-recaptcha-v3";
import { toast } from "sonner";
import { authClient } from "@/lib/auth-client";
import { useRouter } from "next/navigation";
import { verifyRecaptchaAction } from "@/app/actions/recaptcha-actions";

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

  // States
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  // Password strength
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
    <div className="min-h-screen w-full grid lg:grid-cols-2 lg:h-screen lg:overflow-hidden">
      {/* Left Side - Branding */}
      <div className="hidden lg:flex flex-col justify-center items-center bg-gradient-to-br from-green-600 via-green-700 to-emerald-800 p-12 relative overflow-hidden lg:rounded-tr-[16px] lg:rounded-br-[16px] shadow-2xl z-20">
        <div className="absolute top-0 left-0 w-full h-full opacity-10">
          <div className="absolute top-20 left-20 w-64 h-64 bg-white rounded-full blur-3xl"></div>
          <div className="absolute bottom-20 right-20 w-96 h-96 bg-white rounded-full blur-3xl"></div>
        </div>
        <div className="relative z-10 text-center space-y-8 max-w-lg">
          <div className="flex justify-center">
            <Image
              src="/images/logo-laci.webp"
              alt="Logo LACI"
              width={200}
              height={172}
              className="drop-shadow-2xl"
              style={{ height: "auto" }}
              priority
            />
          </div>
          <div className="space-y-4">
            <h1 className="text-4xl font-bold text-white">
              Laci Digital IPNU IPPNU
            </h1>
            <p className="text-lg text-green-50 leading-relaxed">
              Bergabunglah dengan sistem manajemen administrasi digital untuk
              PAC IPNU & IPPNU
            </p>
          </div>
          <div className="pt-8 space-y-3 text-green-50">
            <div className="flex items-center gap-3 justify-center">
              <div className="w-2 h-2 bg-green-200 rounded-full"></div>
              <p className="text-sm">Pendaftaran mudah dan cepat</p>
            </div>
            <div className="flex items-center gap-3 justify-center">
              <div className="w-2 h-2 bg-green-200 rounded-full"></div>
              <p className="text-sm">Verifikasi oleh Sekretaris Cabang</p>
            </div>
            <div className="flex items-center gap-3 justify-center">
              <div className="w-2 h-2 bg-green-200 rounded-full"></div>
              <p className="text-sm">Akses penuh setelah aktivasi</p>
            </div>
          </div>
        </div>
      </div>

      {/* Right Side - Form Container */}
      <div className="flex items-start sm:items-center justify-center p-6 bg-white overflow-hidden">
        <div className="w-full max-w-md space-y-4 pt-4 sm:pt-0">
          {/* Mobile Logo */}
          <div className="lg:hidden flex justify-center">
            <Image
              src="/images/logo-laci.webp"
              alt="Logo LACI"
              width={60}
              height={52}
              style={{ height: "auto" }}
              priority
            />
          </div>

          <div className="space-y-1 text-center lg:text-left">
            <h2 className="text-2xl sm:text-3xl font-bold text-slate-900">
              Buat Akun Baru
            </h2>
            <p className="text-xs sm:text-sm text-slate-600">
              {isCabangReady
                ? "Daftar untuk mulai menggunakan Laci Digital"
                : "Pendaftaran Akun PAC Sementara Ditangguhkan"}
            </p>
          </div>

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
                  Pendaftaran akun PAC baru memerlukan **Verifikasi & Periode
                  Aktif** dari Sekretaris Cabang. Saat ini, Admin Cabang belum
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
                  <ServerCrash className="text-slate-400" size={20} />
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
                  className="w-full flex justify-center items-center py-3 px-4 rounded-lg text-white bg-green-700 hover:bg-green-800 transition-all shadow-lg font-bold"
                >
                  Masuk ke Akun Admin
                </Link>

                <p className="text-center text-xs text-slate-400">
                  Jika Anda adalah Admin Cabang, silakan masuk dan aktifkan
                  periode di Dashboard.
                </p>
              </div>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <label className="block text-sm font-semibold text-slate-700">
                  Nama Pimpinan
                </label>
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
                    <User size={18} className="text-slate-400" />
                  </div>
                  <input
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    className="block w-full pl-11 pr-4 py-3 border border-slate-300 rounded-lg focus:ring-2 focus:ring-green-600 outline-none transition-all"
                    placeholder="Nama Pimpinan"
                    required
                    disabled={isLoading}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <label className="block text-sm font-semibold text-slate-700">
                  Alamat Email
                </label>
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
                    <Mail size={18} className="text-slate-400" />
                  </div>
                  <input
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className="block w-full pl-11 pr-4 py-3 border border-slate-300 rounded-lg focus:ring-2 focus:ring-green-600 outline-none transition-all"
                    placeholder="anda@email.com"
                    required
                    disabled={isLoading}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <label className="block text-sm font-semibold text-slate-700">
                  Password
                </label>
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
                    <Lock size={18} className="text-slate-400" />
                  </div>
                  <input
                    type={showPassword ? "text" : "password"}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="block w-full pl-11 pr-12 py-3 border border-slate-300 rounded-lg focus:ring-2 focus:ring-green-600 outline-none transition-all"
                    placeholder="Minimal 6 karakter"
                    required
                    disabled={isLoading}
                  />
                  <button
                    type="button"
                    aria-label={showPassword ? "Sembunyikan password" : "Tampilkan password"}
                    title={showPassword ? "Sembunyikan password" : "Tampilkan password"}
                    className="absolute inset-y-0 right-0 flex w-11 items-center justify-center text-slate-400 transition-colors hover:text-slate-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-green-600"
                    onClick={() => setShowPassword(!showPassword)}
                  >
                    {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
                  </button>
                </div>
              </div>

              {/* Password Strength Indicator */}
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
                <label className="block text-sm font-semibold text-slate-700">
                  Konfirmasi Password
                </label>
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
                    <Lock size={18} className="text-slate-400" />
                  </div>
                  <input
                    type={showConfirmPassword ? "text" : "password"}
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    className="block w-full pl-11 pr-12 py-3 border border-slate-300 rounded-lg focus:ring-2 focus:ring-green-600 outline-none transition-all"
                    placeholder="Ulangi password"
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
                    className="absolute inset-y-0 right-0 flex w-11 items-center justify-center text-slate-400 transition-colors hover:text-slate-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-green-600"
                    onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                  >
                    {showConfirmPassword ? (
                      <EyeOff size={18} />
                    ) : (
                      <Eye size={18} />
                    )}
                  </button>
                </div>
                {/* Match indicator */}
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

              {/* Checkbox Persetujuan Ketentuan & Kebijakan Privasi */}
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
                    className="font-semibold text-green-700 hover:underline"
                  >
                    Ketentuan Penggunaan
                  </Link>{" "}
                  dan{" "}
                  <Link
                    href="/kebijakan-privasi"
                    target="_blank"
                    className="font-semibold text-green-700 hover:underline"
                  >
                    Kebijakan Privasi
                  </Link>{" "}
                  Laci Digital.
                </label>
              </div>

              <button
                type="submit"
                disabled={isLoading || !agreeToTerms}
                className="w-full flex justify-center items-center py-3 px-4 rounded-lg text-white bg-green-700 hover:bg-green-800 transition-all shadow-lg disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer font-bold"
              >
                {isLoading ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  "Daftar Sekarang"
                )}
              </button>
            </form>
          )}
          <div className="text-center text-sm text-slate-600 mt-4">
            Sudah punya akun?{" "}
            <Link
              href="/login"
              className="font-semibold text-green-700 hover:text-green-800"
            >
              Masuk di sini
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
