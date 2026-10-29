import { auth } from "@/auth";
import { redirect } from "next/navigation";
import ProfilePageClient from "@/components/features/profile/profile-page-client";

export const metadata = {
  title: "Profile | Laci Digital",
};

export default async function ProfilePage() {
  const session = await auth();

  if (!session?.user?.id) {
    redirect("/login");
  }

  const userForProfile = {
    ...session.user,
  };

  return <ProfilePageClient user={userForProfile} />;
}
