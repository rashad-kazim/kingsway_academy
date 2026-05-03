import { getTranslations, setRequestLocale } from "next-intl/server";
import { redirect } from "next/navigation";
import Image from "next/image";
import type { CSSProperties } from "react";
import { LoginForm } from "@/components/auth/login-form";
import { currentSession } from "@/lib/auth/session";
import { appAsset } from "@/lib/assets";

type LoginPageProps = {
  params: Promise<{
    locale: string;
  }>;
  searchParams: Promise<{
    next?: string;
  }>;
};

export const dynamic = "force-dynamic";

const bubbles = [
  { x: "2%", size: "210px", duration: "44s", delay: "-8s", drift: "26px" },
  { x: "12%", size: "86px", duration: "31s", delay: "-21s", drift: "-18px" },
  { x: "29%", size: "156px", duration: "49s", delay: "-13s", drift: "30px" },
  { x: "29%", size: "260px", duration: "58s", delay: "-5s", drift: "42px" },
  { x: "48%", size: "190px", duration: "54s", delay: "-30s", drift: "38px" },
  { x: "58%", size: "245px", duration: "48s", delay: "-18s", drift: "34px" },
  { x: "71%", size: "106px", duration: "36s", delay: "-7s", drift: "-24px" },
  { x: "83%", size: "260px", duration: "60s", delay: "-24s", drift: "-46px" },
  { x: "94%", size: "128px", duration: "39s", delay: "-4s", drift: "18px" },
];

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
      className="fixed inset-0 overflow-hidden bg-[#fdfdfd] text-[#0a284b]"
    >
      <div className="absolute inset-0 z-0 bg-[#fdfdfd]" />
      <div className="absolute inset-y-0 right-0 z-0 hidden w-[57%] bg-[linear-gradient(135deg,rgb(10,40,75),rgb(15,55,100))] lg:block" />
      <LoginBubbles />
      <div className="relative z-20 grid h-full w-full overflow-hidden lg:grid-cols-[43%_57%]">
        <section className="relative isolate flex h-full items-center justify-center px-6 sm:px-10 lg:px-12">
          <LoginForm
            labels={{
              title: t("formTitle"),
              description: t("formDescription"),
              email: t("email"),
              password: t("password"),
              forgotPassword: t("forgotPassword"),
              submit: t("submit"),
              pending: t("pending"),
              invalidCredentials: t("invalidCredentials"),
              backendUnavailable: t("backendUnavailable"),
              invalidInput: t("invalidInput"),
            }}
            locale={locale}
            nextPath={next}
          />
        </section>

        <section className="relative isolate hidden h-full overflow-hidden lg:block">
          <div className="relative z-20 flex h-full flex-col px-[9%] pt-[10vh]">
            <h1 className="max-w-[520px] text-[clamp(34px,3.1vw,58px)] font-black leading-[0.98] text-white">
              {t("welcome")}
            </h1>
            <p className="mt-2 max-w-[600px] text-[clamp(34px,3.2vw,60px)] font-black leading-none text-[#ef2334]">
              {t("portal")}
            </p>
            <p className="mt-5 text-[15px] font-semibold text-white">
              {t("heroCaption")}
            </p>
          </div>
          <Image
            src={appAsset("/images/login-right-side.png")}
            priority
            alt=""
            width={860}
            height={573}
            unoptimized={process.env.NODE_ENV === "development"}
            sizes="(min-width: 1024px) 48vw, 0vw"
            className="absolute bottom-0 right-[6%] z-10 h-auto w-[min(48vw,760px)] object-contain opacity-100"
          />
        </section>
      </div>
    </main>
  );
}

function LoginBubbles() {
  return (
    <div aria-hidden="true" className="pointer-events-none absolute inset-0 z-10">
      <BubbleLayer tone="left" />
      <BubbleLayer tone="right" />
    </div>
  );
}

function BubbleLayer({ tone }: { tone: "left" | "right" }) {
  return (
    <div className={`login-bubble-layer login-bubble-layer--${tone}`}>
      {bubbles.map((bubble, index) => (
        <span
          className={`login-bubble login-bubble--${tone}`}
          key={`${tone}-${bubble.x}-${index}`}
          style={
            {
              "--bubble-x": bubble.x,
              "--bubble-size": bubble.size,
              "--bubble-duration": bubble.duration,
              "--bubble-delay": bubble.delay,
              "--bubble-drift": bubble.drift,
            } as CSSProperties
          }
        />
      ))}
    </div>
  );
}
