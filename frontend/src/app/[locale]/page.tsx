import { setRequestLocale } from "next-intl/server";
import { redirect } from "next/navigation";
import { currentSession } from "@/lib/auth/session";

type HomePageProps = {
  params: Promise<{
    locale: string;
  }>;
};

export const dynamic = "force-dynamic";

export default async function HomePage({ params }: HomePageProps) {
  const { locale } = await params;
  setRequestLocale(locale);
  const session = await currentSession();

  redirect(session ? `/${locale}${session.dashboard_path}` : `/${locale}/login`);
}
