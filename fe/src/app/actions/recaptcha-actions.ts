"use server";

import { backendRequest } from "@/lib/backend";

export async function verifyRecaptchaAction(token: string): Promise<boolean> {
  try {
    return Boolean((await backendRequest<any>("/api/recaptcha/verify", { method: "POST", body: { token } })).valid);
  } catch {
    return false;
  }
}
