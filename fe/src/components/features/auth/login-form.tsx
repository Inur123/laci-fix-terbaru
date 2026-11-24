"use client";

import { useState, useCallback, useEffect } from "react";
import Link from "next/link";
import { Mail, Lock, Eye, EyeOff, Loader2 } from "lucide-react";
import LoginClient from "@/components/features/auth/login-client";
import { AuthSplitLayout } from "@/components/features/public-v2/auth-split-layout";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { useGoogleReCaptcha } from "react-google-recaptcha-v3";
import { toast } from "sonner";
import { authClient } from "@/lib/auth-client";
import { useRouter } from "next/navigation";
import { verifyRecaptchaAction } from "@/app/actions/recaptcha-actions";

const requestLocation = (onSuccess?: () => void, onError?: () => void) => {
  if (typeof window === "undefined") return;

  if (!navigator.geolocation) {
    toast.error(
      "Browser Anda tidak mendukung deteksi lokasi (Geolocation). Silakan gunakan browser lain.",
    );
    if (onError) onError();
    return;
  }

  toast.info(
    "Sistem mendeteksi lokasi Anda untuk keamanan. Mohon izinkan akses lokasi jika diminta.",
  );

  navigator.geolocation.getCurrentPosition(
    async (position) => {
      const { latitude, longitude } = position.coords;

      document.cookie = `user_lat=${latitude}; path=/; max-age=86400; SameSite=Lax`;
      document.cookie = `user_lng=${longitude}; path=/; max-age=86400; SameSite=Lax`;

      try {
        let cleanAddress = "";
        // BigDataCloud is the primary reverse-geocoding provider.
        try {
          const res = await fetch(
            `https://api.bigdatacloud.net/data/reverse-geocode-client?latitude=${latitude}&longitude=${longitude}&localityLanguage=id`,
            { signal: AbortSignal.timeout(3000) },
          );
          if (res.ok) {
            const data = await res.json();
            const parts = [];
            if (data.locality) parts.push(data.locality);
            if (data.city) parts.push(data.city);
            if (data.principalSubdivision)
              parts.push(data.principalSubdivision);
            if (parts.length > 0) {
              cleanAddress = parts.join(", ");
            }
          }
        } catch (e) {
          console.warn("BigDataCloud failed, falling back to Nominatim...", e);
        }

        // Fall back to Nominatim when the primary lookup fails.
        if (!cleanAddress) {
          const res = await fetch(
            `https://nominatim.openstreetmap.org/reverse?format=json&lat=${latitude}&lon=${longitude}`,
            {
              headers: {
                "Accept-Language": "id-ID,id;q=0.9,en;q=0.8",
                "User-Agent": "LaciDigital/1.0",
              },
              signal: AbortSignal.timeout(3000),
            },
          );
          if (res.ok) {
            const data = await res.json();
            if (data && data.address) {
              const addr = data.address;
              const addressParts = [];
              const village =
                addr.village || addr.suburb || addr.neighbourhood || addr.road;
              const district =
                addr.county ||
                addr.city_district ||
                addr.municipality ||
                addr.subdistrict;
              const city = addr.city || addr.regency || addr.town;
              const state = addr.state;

              if (village) addressParts.push(village);
              if (district) addressParts.push(district);
              if (city) addressParts.push(city);
              if (state) addressParts.push(state);

              cleanAddress =
                addressParts.length > 0
                  ? addressParts.join(", ")
                  : data.display_name.split(",").slice(0, 3).join(",").trim();
            }
          }
        }

        if (cleanAddress) {
          document.cookie = `user_address=${encodeURIComponent(cleanAddress)}; path=/; max-age=86400; SameSite=Lax`;
        }
      } catch (e) {
        console.warn("Reverse geocoding warning:", e);
      }

      if (onSuccess) onSuccess();
    },
    (error) => {
      console.warn(
        "Geolocation warning:",
        error.message || `Code: ${error.code}`,
      );
      toast.error(
        "Akses Lokasi Ditolak/Gagal. Anda wajib mengizinkan akses lokasi di browser untuk login.",
      );
      if (onError) onError();
    },
    {
      enableHighAccuracy: true,
      timeout: 10000,
      maximumAge: 0,
    },
  );
};

export default function LoginForm() {
  useEffect(() => {
    if (navigator.permissions && navigator.permissions.query) {
      navigator.permissions
        .query({ name: "geolocation" as PermissionName })
        .then((result) => {
          if (result.state !== "granted") {
            requestLocation();
          } else {
            // Refresh location cookies when permission was granted earlier.
            navigator.geolocation.getCurrentPosition((pos) => {
              const { latitude, longitude } = pos.coords;
              document.cookie = `user_lat=${latitude}; path=/; max-age=86400; SameSite=Lax`;
              document.cookie = `user_lng=${longitude}; path=/; max-age=86400; SameSite=Lax`;
            });
          }
        })
        .catch(() => {
          requestLocation();
        });
    } else {
      requestLocation();
    }
  }, []);
  return (
    <AuthSplitLayout
      page="login"
      cardTitle="Selamat datang kembali"
      cardDescription="Masuk untuk mengelola administrasi organisasi Anda"
      alternateAction={
        <p>
          Belum punya akun?{" "}
          <Link
            href="/register"
            className="font-semibold text-[#0a8744] underline-offset-4 hover:underline focus-visible:rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#0a8744]"
          >
            Daftar sekarang
          </Link>
        </p>
      }
    >
      <LoginClient />
      <LoginWithRecaptcha />
    </AuthSplitLayout>
  );
}

function LoginWithRecaptcha() {
  const { executeRecaptcha } = useGoogleReCaptcha();
  const router = useRouter();
  const [showPassword, setShowPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [isGoogleLoading, setIsGoogleLoading] = useState(false);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const handleSubmit = useCallback(
    async (e: React.FormEvent<HTMLFormElement>) => {
      e.preventDefault();

      const localCaptchaBypass = process.env.NODE_ENV !== "production";
      if (!executeRecaptcha && !localCaptchaBypass) {
        toast.error("Verifikasi keamanan belum siap. Silakan refresh halaman.");
        return;
      }

      const cookiesArr = document.cookie.split(";");
      const hasLat = cookiesArr.some((c) => c.trim().startsWith("user_lat="));
      const hasLng = cookiesArr.some((c) => c.trim().startsWith("user_lng="));

      if (!hasLat || !hasLng) {
        toast.error(
          "Akses Masuk Ditolak: Anda wajib mengaktifkan dan mengizinkan akses lokasi pada browser/perangkat Anda untuk masuk.",
        );
        requestLocation();
        setIsLoading(false);
        return;
      }

      setIsLoading(true);

      try {
        const isHuman = localCaptchaBypass
          ? true
          : await verifyRecaptchaAction(await executeRecaptcha!("login"));
        if (!isHuman) {
          toast.error(
            "Terdeteksi sebagai aktivitas mencurigakan (Bot). Silakan coba lagi.",
          );
          setIsLoading(false);
          return;
        }

        const { error } = await authClient.signIn.email({
          email,
          password,
        });

        if (error) {
          let errorMessage = "Email atau password salah.";
          if (
            error.status === 429 ||
            error.code === "TOO_MANY_REQUESTS" ||
            error.message?.toLowerCase().includes("too many")
          ) {
            errorMessage =
              "Terlalu banyak percobaan login. Mohon tunggu beberapa saat sebelum mencoba kembali.";
          } else if (
            error.status === 401 ||
            error.code === "INVALID_EMAIL_OR_PASSWORD"
          ) {
            errorMessage =
              "Email atau password salah. Silakan periksa kembali.";
          } else {
            errorMessage = error.message || "Terjadi kesalahan saat login.";
          }
          toast.error(errorMessage);
          setIsLoading(false);
          return;
        }

        const session = await authClient.getSession();
        if (
          session.data?.user &&
          (session.data.user as any).isActive === false
        ) {
          toast.error("Akun Anda belum diaktifkan oleh Sekretaris Cabang.");
          await authClient.signOut();
          setIsLoading(false);
          return;
        }

        // Dashboard shows the success toast after redirect.
        router.push("/dashboard?login=success");
        router.refresh();
      } catch (error) {
        console.error("Login error:", error);
        toast.error("Terjadi kesalahan. Silakan coba lagi.");
        setIsLoading(false);
      }
    },
    [email, password, executeRecaptcha, router],
  );

  return (
    <form
      onSubmit={handleSubmit}
      className="space-y-5"
      name="login"
      data-form-type="login"
    >
      <div className="space-y-2">
        <Label htmlFor="email" className="font-semibold text-slate-700">
          Alamat Email
        </Label>
        <div className="relative">
          <Mail
            aria-hidden="true"
            className="pointer-events-none absolute left-3.5 top-1/2 size-[18px] -translate-y-1/2 text-slate-500"
          />
          <Input
            type="email"
            id="email"
            name="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="h-12 rounded-xl border border-solid border-slate-200/80 bg-white pl-11 text-slate-900 shadow-none placeholder:text-slate-500 hover:border-slate-300 focus-visible:border-[#0a8744] focus-visible:ring-0"
            placeholder="anda@email.com"
            autoComplete="email"
            data-lpignore="false"
            data-form-type="email"
            required
            disabled={isLoading}
          />
        </div>
      </div>

      <div className="space-y-2">
        <div className="flex items-center justify-between gap-4">
          <Label htmlFor="password" className="font-semibold text-slate-700">
          Password
          </Label>
          <Link
            href="/faq"
            className="text-xs font-semibold text-[#0a8744] underline-offset-4 hover:underline focus-visible:rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#0a8744]"
          >
            Butuh bantuan?
          </Link>
        </div>
        <div className="relative">
          <Lock
            aria-hidden="true"
            className="pointer-events-none absolute left-3.5 top-1/2 size-[18px] -translate-y-1/2 text-slate-500"
          />
          <Input
            type={showPassword ? "text" : "password"}
            id="password"
            name="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="h-12 rounded-xl border border-solid border-slate-200/80 bg-white pl-11 pr-12 text-slate-900 shadow-none placeholder:text-slate-500 hover:border-slate-300 focus-visible:border-[#0a8744] focus-visible:ring-0"
            placeholder="Masukkan password"
            autoComplete="current-password"
            data-lpignore="false"
            data-form-type="password"
            required
            disabled={isLoading}
          />

          <button
            type="button"
            aria-label={showPassword ? "Sembunyikan password" : "Tampilkan password"}
            title={showPassword ? "Sembunyikan password" : "Tampilkan password"}
            className="absolute inset-y-0 right-0 flex w-12 items-center justify-center rounded-r-xl text-slate-500 transition-colors hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-[#0a8744]"
            onClick={() => setShowPassword(!showPassword)}
            disabled={isLoading}
          >
            {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
          </button>
        </div>
      </div>

      <Button
        type="submit"
        disabled={isLoading}
        className="h-12 w-full rounded-xl bg-[#0a8744] font-bold text-white shadow-sm shadow-emerald-900/10 hover:bg-[#086d37] focus-visible:ring-[#0a8744]/30"
      >
        {isLoading ? (
          <>
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            Memproses...
          </>
        ) : (
          "Masuk"
        )}
      </Button>

      <div className="relative py-1">
        <Separator />
        <span className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 bg-white px-3 text-[11px] font-medium uppercase tracking-[0.14em] text-slate-500">
          atau
        </span>
      </div>

      <Button
        type="button"
        onClick={async () => {
          const cookiesArr = document.cookie.split(";");
          const hasLat = cookiesArr.some((c) =>
            c.trim().startsWith("user_lat="),
          );
          const hasLng = cookiesArr.some((c) =>
            c.trim().startsWith("user_lng="),
          );

          if (!hasLat || !hasLng) {
            toast.error(
              "Akses Masuk Ditolak: Anda wajib mengaktifkan dan mengizinkan akses lokasi pada browser/perangkat Anda untuk masuk.",
            );
            requestLocation();
            return;
          }

          setIsGoogleLoading(true);
          try {
            await authClient.signIn.social({
              provider: "google",
              callbackURL: `${window.location.origin}/dashboard?login=success`,
              errorCallbackURL: `${window.location.origin}/login?error=unregistered`,
            });
          } catch (err) {
            console.error("Google login error:", err);
            toast.error("Terjadi kesalahan saat login Google.");
            setIsGoogleLoading(false);
          }
        }}
        disabled={isLoading || isGoogleLoading}
        variant="outline"
        className="h-12 w-full rounded-xl border-slate-200/80 bg-white font-semibold text-slate-700 hover:border-slate-300 hover:bg-slate-50"
      >
        {isGoogleLoading ? (
          <Loader2 className="w-5 h-5 animate-spin text-green-600" />
        ) : (
          <svg
            viewBox="0 0 24 24"
            className="w-5 h-5"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
              fill="#4285F4"
            />
            <path
              d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
              fill="#34A853"
            />
            <path
              d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l3.66-2.84z"
              fill="#FBBC05"
            />
            <path
              d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
              fill="#EA4335"
            />
          </svg>
        )}
        {isGoogleLoading ? "Menyambungkan..." : "Masuk dengan Google"}
      </Button>
    </form>
  );
}
