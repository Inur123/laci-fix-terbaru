"use server";

import { revalidatePath } from "next/cache";
import { backendDataOrNull, backendRequest, errorMessage, queryString } from "@/lib/backend";

export async function verifyOTP(email: string, otp: string) {
  try {
    return await backendRequest<any>("/api/auth/otp/verify", { method: "POST", body: { email, otp } });
  } catch (error) {
    return { error: errorMessage(error, "Kode OTP tidak valid.") };
  }
}

export async function resendVerificationAction(email: string) {
  try {
    return await backendRequest<any>("/api/auth/otp/send", { method: "POST", body: { email } });
  } catch (error) {
    return { error: errorMessage(error, "Gagal mengirim email verifikasi.") };
  }
}

export async function checkEmailVerificationStatus(email: string) {
  try {
    return await backendRequest<any>(`/api/auth/verification-status${queryString({ email })}`);
  } catch (error) {
    return { error: errorMessage(error, "User tidak ditemukan") };
  }
}

export async function getPACUsers(query?: string, page = 1, limit = 10, status?: string, emailStatus?: string, sortKey?: string | null, sortDir: "asc" | "desc" = "desc") {
  return backendRequest<any>(`/api/users${queryString({ q: query, page, limit, status, emailStatus, sortKey, sortDir })}`);
}

export async function getUserStats() {
  return backendRequest<any>("/api/users/stats");
}

export async function getUserDetail(userId: string) {
  return backendDataOrNull(`/api/users/${encodeURIComponent(userId)}`);
}

export async function toggleUserStatus(userId: string) {
  try {
    const result = await backendRequest<any>(`/api/users/${encodeURIComponent(userId)}/status`, { method: "PATCH" });
    revalidatePath("/dashboard/manajemen-user");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal mengubah status akun.") };
  }
}

export async function deleteUser(userId: string) {
  try {
    const result = await backendRequest<any>(`/api/users/${encodeURIComponent(userId)}`, { method: "DELETE" });
    revalidatePath("/dashboard/manajemen-user");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal menghapus user.") };
  }
}

export async function resetUserPassword(userId: string) {
  try {
    return await backendRequest<any>(`/api/users/${encodeURIComponent(userId)}/reset-password`, { method: "POST" });
  } catch (error) {
    return { error: errorMessage(error, "Gagal mereset password.") };
  }
}

export async function updateProfile(formData: FormData) {
  try {
    const result = await backendRequest<any>("/api/me", { method: "PATCH", body: formData });
    revalidatePath("/dashboard", "layout");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal memperbarui profil.") };
  }
}
