import { getTranslations, setRequestLocale } from "next-intl/server";
import { redirect } from "next/navigation";
import Image from "next/image";
import { LoginForm } from "@/components/auth/login-form";
import { currentSession } from "@/lib/auth/session";

type LoginPageProps = {
  params: Promise<{
    locale: string;
  }>;
  searchParams: Promise<{
    next?: string;
  }>;
};

export const dynamic = "force-dynamic";

export default async function LoginPage({
  params,
  searchParams,
}: LoginPageProps) {
  const { locale } = await params;
  const { next } = await searchParams;
  setRequestLocale(locale);

  const session = await currentSession();
  if (session) {
    redirect(`/${locale}${session.dashboard_path}`);
  }

  const t = await getTranslations("login");

  return (
    <main
      data-login-page
      className="fixed inset-0 overflow-hidden bg-[#1f1f1f] text-white"
    >
      <div className="grid h-full w-full overflow-hidden bg-[#1d1e23] lg:grid-cols-[44%_56%]">
        <section className="relative flex h-full items-center justify-center bg-[#1d1e23] px-8 py-8 sm:px-12 lg:px-20">
          <LoginForm
            labels={{
              title: t("formTitle"),
              description: t("formDescription"),
              email: t("email"),
              password: t("password"),
              forgotPassword: t("forgotPassword"),
              submit: t("submit"),
              pending: t("pending"),
              signUpPrompt: t("signUpPrompt"),
              signUp: t("signUp"),
              invalidCredentials: t("invalidCredentials"),
              backendUnavailable: t("backendUnavailable"),
              invalidInput: t("invalidInput"),
            }}
            locale={locale}
            nextPath={next}
          />
        </section>

        <section className="relative hidden h-full overflow-hidden bg-[#9662e5] lg:block">
          <div className="absolute -left-32 top-20 size-96 rounded-full bg-white/10" />
          <div className="absolute left-4 top-[66%] size-44 rounded-full bg-white/8" />
          <div className="absolute right-[-72px] top-[-88px] h-72 w-96 rounded-[42%] bg-white/8" />
          <div className="absolute right-[-60px] top-24 h-36 w-80 rounded-[42%] bg-white/8" />

          <div className="relative z-10 mx-auto flex h-full max-w-[560px] flex-col px-16 pb-10 pt-24">
            <div className="space-y-1">
              <h1 className="max-w-[430px] text-[54px] font-black leading-[0.95] text-white">
                {t("welcome")}
              </h1>
              <p className="max-w-[430px] text-[45px] font-light leading-none text-white">
                {t("portal")}
              </p>
              <p className="pt-2 text-[12px] font-semibold text-white/85">
                {t("heroCaption")}
              </p>
            </div>

            <div className="relative mt-auto flex justify-center">
              <Image
                src="/images/login-page-illustration.svg"
                width={520}
                height={426}
                unoptimized
                alt=""
                className="h-auto w-[520px] max-w-none object-contain"
              />
            </div>
          </div>
        </section>
      </div>
    </main>
  );
}
