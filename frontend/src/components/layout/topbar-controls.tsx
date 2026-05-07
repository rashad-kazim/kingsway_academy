"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useTheme } from "next-themes";
import { useEffect, useState } from "react";
import {
  ChevronDown,
  Languages,
  LogOut,
  Moon,
  Sun,
} from "lucide-react";
import {
  Avatar,
  AvatarFallback,
} from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { Role } from "@/lib/api/types";
import { logoutAction } from "@/lib/auth/actions";
import { cn } from "@/lib/utils";

export type ChromeLanguageCode = "en" | "az" | "ru" | "de";

export type ChromeLabels = {
  brand: string;
  academy: string;
  commandCenter: string;
  signOut: string;
  language: string;
  profileMenu: string;
  theme: string;
  switchToDark: string;
  switchToLight: string;
  roles: Record<Role, string>;
  languages: Record<ChromeLanguageCode, string>;
};

type TopbarControlsProps = {
  labels: ChromeLabels;
  locale: string;
  role?: Role;
  showProfile?: boolean;
  userName?: string;
};

const localeOptions: ChromeLanguageCode[] = ["en", "az", "ru", "de"];

export function TopbarControls({
  labels,
  locale,
  role,
  showProfile = true,
  userName,
}: TopbarControlsProps) {
  const { resolvedTheme, setTheme } = useTheme();
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();
  const [mounted, setMounted] = useState(false);
  const darkMode = mounted ? resolvedTheme === "dark" : false;
  const activeLanguage =
    labels.languages[locale as ChromeLanguageCode] ?? labels.languages.en;

  useEffect(() => {
    const timer = window.setTimeout(() => setMounted(true), 0);
    return () => window.clearTimeout(timer);
  }, []);

  return (
    <div className="flex items-center gap-3">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            aria-label={labels.language}
            className="gap-2 rounded-md border border-white/15 bg-white/10 px-3 text-white hover:bg-white/15 hover:text-white dark:border-[#3b4658] dark:bg-[#202b3a] dark:text-[#f3f6fa] dark:hover:bg-[#263448] dark:hover:text-white"
            type="button"
            variant="ghost"
          >
            <Languages className="size-4" />
            <span className="text-sm font-semibold">{activeLanguage}</span>
            <ChevronDown className="size-4 opacity-80" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="z-[1100] min-w-0">
          {localeOptions.map((nextLocale) => (
            <DropdownMenuItem
              className="cursor-pointer"
              key={nextLocale}
              onClick={() =>
                router.push(localePath(pathname, searchParams, nextLocale))
              }
            >
              {labels.languages[nextLocale]}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>

      <button
        aria-label={darkMode ? labels.switchToLight : labels.switchToDark}
        className={cn(
          "relative grid h-9 w-[74px] cursor-pointer grid-cols-2 items-center rounded-full border border-white/15 px-1 transition-colors hover:bg-white/15 dark:border-[#3b4658] dark:hover:bg-[#263448]",
          darkMode ? "bg-[#0b1622] text-[#e7ecf3]" : "bg-white/10 text-white",
        )}
        type="button"
        onClick={() => setTheme(darkMode ? "light" : "dark")}
      >
        <Sun className="z-10 mx-auto size-4 opacity-70" />
        <Moon className="z-10 mx-auto size-4 opacity-70" />
        <span
          className={cn(
            "absolute top-1 flex size-7 items-center justify-center rounded-full bg-white text-[#0a284b] shadow-sm transition-transform duration-300 ease-in-out dark:bg-[#f3f6fa] dark:text-[#0b1622]",
            darkMode ? "translate-x-[39px]" : "translate-x-1",
          )}
        >
          {darkMode ? <Moon className="size-4" /> : <Sun className="size-4" />}
        </span>
      </button>

      {showProfile && role && userName ? (
        <ProfileMenu
          labels={labels}
          locale={locale}
          role={role}
          userName={userName}
        />
      ) : null}
    </div>
  );
}

export function ProfileMenu({
  labels,
  locale,
  open = true,
  role,
  userName,
  variant = "topbar",
}: {
  labels: ChromeLabels;
  locale: string;
  open?: boolean;
  role: Role;
  userName: string;
  variant?: "topbar" | "sidebar";
}) {
  const sidebar = variant === "sidebar";
  const showText = !sidebar || open;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          aria-label={labels.profileMenu}
          className={cn(
            "flex cursor-pointer items-center gap-3 rounded-md text-left text-white transition-colors hover:bg-white/10 dark:text-[#f3f6fa] dark:hover:bg-[#202d3e]",
            sidebar ? "h-14 w-full px-3" : "px-2 py-1.5",
            sidebar && !open && "h-12 justify-center px-0",
          )}
          type="button"
        >
          <Avatar className="size-10 shrink-0">
            <AvatarFallback className="bg-white text-sm font-black text-[#0a284b]">
              {initials(userName)}
            </AvatarFallback>
          </Avatar>
          {showText ? (
            <>
              <span
                className={cn(
                  "min-w-0 flex-col leading-tight",
                  sidebar ? "flex" : "hidden lg:flex",
                )}
              >
                <span className="max-w-44 truncate text-sm font-bold">
                  {userName}
                </span>
                <span className="mt-0.5 text-xs font-medium text-white/70 dark:text-[#a7b0bf]">
                  {labels.roles[role]}
                </span>
              </span>
              <ChevronDown className="ml-auto size-4 shrink-0 text-white/75 dark:text-[#a7b0bf]" />
            </>
          ) : null}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align={sidebar ? "start" : "end"}
        className="z-[1100] min-w-48"
        side={sidebar ? "right" : "bottom"}
        sideOffset={10}
      >
        <div className="px-2 py-1.5">
          <div className="truncate text-sm font-semibold">{userName}</div>
          <div className="text-xs text-muted-foreground">
            {labels.roles[role]}
          </div>
        </div>
        <DropdownMenuSeparator />
        <form action={logoutAction}>
          <input name="locale" type="hidden" value={locale} />
          <DropdownMenuItem
            asChild
            className="cursor-pointer transition-colors focus:text-[#ef2334] data-[highlighted]:text-[#ef2334] data-[highlighted]:[&_svg]:text-[#ef2334] dark:focus:text-[#ff3b4f] dark:data-[highlighted]:text-[#ff3b4f] dark:data-[highlighted]:[&_svg]:text-[#ff3b4f]"
          >
            <button
              className="flex w-full cursor-pointer items-center gap-2 transition-colors hover:text-[#ef2334] focus:text-[#ef2334] dark:hover:text-[#ff3b4f] dark:focus:text-[#ff3b4f] [&_svg]:stroke-current [&_svg]:transition-colors"
              type="submit"
            >
              <LogOut className="size-4" />
              {labels.signOut}
            </button>
          </DropdownMenuItem>
        </form>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function localePath(
  pathname: string,
  searchParams: URLSearchParams,
  nextLocale: ChromeLanguageCode,
) {
  const nextPath = pathname.replace(
    /^\/(en|az|ru|de)(?=\/|$)/,
    `/${nextLocale}`,
  );
  const query = searchParams.toString();
  return query ? `${nextPath}?${query}` : nextPath;
}

function initials(name: string) {
  const parts = name.trim().split(/\s+/).slice(0, 2);
  return parts.map((part) => part[0]?.toUpperCase()).join("") || "K";
}
