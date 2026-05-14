"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";
import type { ReactNode } from "react";
import { useMemo, useState } from "react";
import {
  BadgeDollarSign,
  Bell,
  Building2,
  CalendarDays,
  CalendarCheck,
  CreditCard,
  Files,
  Gauge,
  GraduationCap,
  ShieldCheck,
  UserCog,
  UserRound,
  Users,
  type LucideIcon,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  ProfileMenu,
  TopbarControls,
  type ChromeLabels,
} from "@/components/layout/topbar-controls";
import type { Branch, Role, Session } from "@/lib/api/types";
import { cn } from "@/lib/utils";
import { dashboardPathForRole } from "@/lib/navigation/roles";
import { useDevAssetSrc } from "@/lib/use-dev-asset-src";

type NavLabelKey =
  | "globalDashboard"
  | "branchManagement"
  | "teacherFinanceHr"
  | "receptionistManagement"
  | "studentAssignmentHub"
  | "schedulingRooms"
  | "paymentHub"
  | "eventsExams"
  | "students"
  | "reception"
  | "classes"
  | "progress"
  | "schedule"
  | "finance"
  | "files"
  | "alerts";

export type AppShellLabels = {
  common: ChromeLabels;
  shell: {
    openSidebar: string;
    closeSidebar: string;
    footer: string;
    navigation: Record<NavLabelKey, string>;
  };
};

type AppShellProps = {
  children: ReactNode;
  labels: AppShellLabels;
  locale: string;
  selectedBranch?: Branch;
  session: Session;
};

type NavItem = {
  href: string;
  icon: LucideIcon;
  labelKey: NavLabelKey;
  roles: Role[];
};

const navigation: NavItem[] = [
  {
    href: "/dashboard/owner",
    labelKey: "globalDashboard",
    icon: Gauge,
    roles: ["owner"],
  },
  {
    href: "/dashboard/owner",
    labelKey: "branchManagement",
    icon: Building2,
    roles: ["owner"],
  },
  {
    href: "/dashboard/owner",
    labelKey: "teacherFinanceHr",
    icon: BadgeDollarSign,
    roles: ["owner"],
  },
  {
    href: "/dashboard/owner",
    labelKey: "receptionistManagement",
    icon: UserCog,
    roles: ["owner"],
  },
  {
    href: "/dashboard/owner",
    labelKey: "studentAssignmentHub",
    icon: Users,
    roles: ["owner"],
  },
  {
    href: "/dashboard/owner",
    labelKey: "schedulingRooms",
    icon: CalendarDays,
    roles: ["owner"],
  },
  {
    href: "/dashboard/owner",
    labelKey: "paymentHub",
    icon: CreditCard,
    roles: ["owner"],
  },
  {
    href: "/dashboard/owner",
    labelKey: "eventsExams",
    icon: CalendarCheck,
    roles: ["owner"],
  },
  {
    href: "/dashboard/owner",
    labelKey: "students",
    icon: Users,
    roles: ["receptionist"],
  },
  {
    href: "/dashboard/receptionist",
    labelKey: "reception",
    icon: ShieldCheck,
    roles: ["receptionist"],
  },
  {
    href: "/dashboard/teacher",
    labelKey: "classes",
    icon: GraduationCap,
    roles: ["teacher"],
  },
  {
    href: "/dashboard/student",
    labelKey: "progress",
    icon: UserRound,
    roles: ["student"],
  },
  {
    href: "/dashboard",
    labelKey: "schedule",
    icon: CalendarDays,
    roles: ["receptionist", "teacher", "student"],
  },
  {
    href: "/dashboard",
    labelKey: "finance",
    icon: CreditCard,
    roles: ["receptionist", "student"],
  },
  {
    href: "/dashboard",
    labelKey: "files",
    icon: Files,
    roles: ["receptionist", "teacher"],
  },
  {
    href: "/dashboard",
    labelKey: "alerts",
    icon: Bell,
    roles: ["receptionist", "teacher", "student"],
  },
];

export function AppShell({
  children,
  labels,
  locale,
  selectedBranch,
  session,
}: AppShellProps) {
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [labelsReady, setLabelsReady] = useState(true);
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const logoSrc = useDevAssetSrc("/images/kingsway-mark.png");
  const roleHome = dashboardPathForRole(session.user.role);
  const homeHref = selectedBranch
    ? `/${locale}${roleHome}?branch_id=${selectedBranch.id}`
    : `/${locale}${roleHome}`;
  const userName = `${session.user.first_name} ${session.user.last_name}`;

  const visibleNavigation = useMemo(
    () => navigation.filter((item) => item.roles.includes(session.user.role)),
    [session.user.role],
  );

  function toggleSidebar() {
    setLabelsReady(false);
    setSidebarOpen((current) => !current);
    window.setTimeout(() => setLabelsReady(true), 280);
  }

  function isNavigationActive(item: NavItem, href: string) {
    const activeView = searchParams.get("view");

    if (item.labelKey === "globalDashboard") {
      return pathname === `/${locale}${href}` && !activeView;
    }

    if (item.labelKey === "branchManagement") {
      return pathname === `/${locale}/dashboard/owner` && activeView === "branches";
    }

    if (item.labelKey === "teacherFinanceHr") {
      return (
        (pathname === `/${locale}/dashboard/owner` &&
          (activeView === "teacher-finance" ||
            activeView === "teacher-add" ||
            activeView === "teacher-edit")) ||
        pathname.startsWith(`/${locale}/dashboard/owner/teacher`)
      );
    }

    if (item.labelKey === "receptionistManagement") {
      return pathname === `/${locale}/dashboard/owner` && activeView === "receptionists";
    }

    if (item.labelKey === "studentAssignmentHub") {
      return (
        pathname === `/${locale}/dashboard/owner` &&
        (activeView === "student-assignment" || activeView === "student-add")
      );
    }

    return false;
  }

  return (
    <div className="h-svh overflow-hidden bg-kw-c-f7f8fb text-kw-c-0a284b transition-colors dark:bg-kw-c-0b1622 dark:text-kw-c-f3f6fa">
      <header className="fixed inset-x-0 top-0 z-[1000] flex h-20 items-center justify-between border-b border-transparent bg-kw-c-0a284b px-6 text-white shadow-sm dark:border-kw-c-293445 dark:bg-kw-c-121f2d dark:text-kw-c-f3f6fa">
        <div className="flex w-72 items-center gap-3">
          <Button
            aria-label={
              sidebarOpen ? labels.shell.closeSidebar : labels.shell.openSidebar
            }
            className="relative h-9 w-11 overflow-hidden rounded-md text-white hover:bg-white/10 hover:text-white dark:text-kw-c-f3f6fa dark:hover:bg-kw-c-202d3e"
            size="icon"
            type="button"
            variant="ghost"
            onClick={toggleSidebar}
          >
            <HamburgerMorph open={sidebarOpen} />
          </Button>
        </div>

        <Link
          className="absolute left-1/2 flex -translate-x-1/2 cursor-pointer items-center gap-3 text-white dark:text-kw-c-f3f6fa"
          href={homeHref}
        >
          <Image
            src={logoSrc}
            alt={labels.common.brand}
            width={501}
            height={499}
            priority
            unoptimized={process.env.NODE_ENV === "development"}
            className="size-12 object-contain drop-shadow-kw-logo"
          />
          <div className="leading-[0.92]">
            <div className="text-kw-18 font-black uppercase tracking-[0.14em] text-white">
              {labels.common.brand}
            </div>
            <div className="mt-1 text-kw-12 font-black uppercase tracking-[0.36em] text-kw-c-ff3b4f">
              {labels.common.academy}
            </div>
          </div>
        </Link>

        <TopbarControls
          labels={labels.common}
          locale={locale}
          showProfile={false}
        />
      </header>

      <aside
        className={cn(
          "fixed bottom-0 left-0 top-0 z-[999] border-r border-kw-c-dce3ee bg-kw-c-0a284b pt-20 text-white transition-all duration-300 ease-in-out dark:border-kw-c-293445 dark:bg-kw-c-121f2d dark:text-kw-c-f3f6fa",
          sidebarOpen ? "w-80" : "w-20",
        )}
      >
        <div className="flex h-full flex-col">
          <nav className="flex flex-1 flex-col gap-1 p-3 pt-5">
            {visibleNavigation.map((item) => {
              const href = item.href === "/dashboard" ? roleHome : item.href;
              const selectedBranchID = selectedBranch?.id;
              const shouldScopeToBranch =
                Boolean(selectedBranchID) &&
                item.labelKey !== "branchManagement";
              const fullHref =
                item.labelKey === "globalDashboard" &&
                session.user.role === "owner"
                  ? `/${locale}${href}?branch_id=${selectedBranchID ?? "all"}`
                  : item.labelKey === "branchManagement" &&
                      session.user.role === "owner"
                    ? `/${locale}${href}?view=branches`
                  : item.labelKey === "teacherFinanceHr" &&
                      session.user.role === "owner"
                    ? `/${locale}${href}?view=teacher-finance`
                  : item.labelKey === "receptionistManagement" &&
                      session.user.role === "owner"
                    ? `/${locale}${href}?view=receptionists`
                  : item.labelKey === "studentAssignmentHub" &&
                      session.user.role === "owner"
                    ? `/${locale}${href}?view=student-assignment`
                  : shouldScopeToBranch
                    ? `/${locale}${href}?branch_id=${selectedBranchID}`
                    : `/${locale}${href}`;
              return (
                <SidebarLink
                  active={isNavigationActive(item, href)}
                  key={`${item.labelKey}-${href}`}
                  href={fullHref}
                  icon={item.icon}
                  label={labels.shell.navigation[item.labelKey]}
                  labelsReady={labelsReady}
                  open={sidebarOpen}
                />
              );
            })}
          </nav>
          <div className="border-t border-white/10 p-3 dark:border-kw-c-293445">
            <ProfileMenu
              labels={labels.common}
              locale={locale}
              open={sidebarOpen}
              role={session.user.role}
              userName={userName}
              variant="sidebar"
            />
          </div>
        </div>
      </aside>

      <div
        className={cn(
          "fixed bottom-0 right-0 top-20 transition-all duration-300 ease-in-out",
          sidebarOpen ? "left-80" : "left-20",
        )}
      >
        <div className="h-full overflow-y-auto">
          <div className="flex min-h-full flex-col">
            <main className="mx-auto flex w-full max-w-7xl flex-1 flex-col gap-6 p-6">
              {children}
            </main>
            <footer className="w-full border-t border-kw-c-dce3ee bg-white/55 px-6 py-4 dark:border-kw-c-293445 dark:bg-kw-c-121f2d/55">
              <div className="flex w-full flex-wrap items-center justify-between gap-3 text-xs text-kw-c-687386 dark:text-kw-c-6f7a8a">
                <span className="font-semibold">{labels.shell.footer}</span>
                <span>&copy; 2026 Kingsway Academy. Internal operation system.</span>
              </div>
            </footer>
          </div>
        </div>
      </div>
    </div>
  );
}

function HamburgerMorph({ open }: { open: boolean }) {
  return (
    <span
      aria-hidden="true"
      className="relative block h-5 w-7 text-current"
    >
      <span
        className={cn(
          "absolute left-[10px] top-0 h-0.5 w-[17px] rounded-full bg-current transition-all duration-300 ease-in-out",
          open && "left-[3px] top-[9px] w-[22px] rotate-45",
        )}
      />
      <span
        className={cn(
          "absolute left-0 top-[9px] h-0.5 w-7 rounded-full bg-current transition-all duration-300 ease-in-out",
          open && "opacity-0",
        )}
      />
      <span
        className={cn(
          "absolute left-0 top-[18px] h-0.5 w-[15px] rounded-full bg-current transition-all duration-300 ease-in-out",
          open && "left-[3px] top-[9px] w-[22px] -rotate-45",
        )}
      />
    </span>
  );
}

function SidebarLink({
  active,
  href,
  icon: Icon,
  label,
  labelsReady,
  open,
}: {
  active: boolean;
  href: string;
  icon: LucideIcon;
  label: string;
  labelsReady: boolean;
  open: boolean;
}) {
  const showLabel = open && labelsReady;
  const content = (
    <Link
      className={cn(
        "relative flex h-12 cursor-pointer items-center gap-3 overflow-hidden rounded-md px-3 text-kw-15 font-semibold text-white transition-colors hover:bg-white/10 hover:text-white dark:text-kw-c-f3f6fa dark:hover:bg-kw-c-202d3e",
        active &&
          "bg-kw-c-306186 text-white before:absolute before:left-0 before:top-2 before:h-8 before:w-1 before:rounded-r-full before:bg-kw-c-ff3b4f dark:bg-kw-c-306186",
      )}
      href={href}
    >
      <span className="flex w-10 shrink-0 items-center justify-center">
        <Icon className="size-[22px] shrink-0" />
      </span>
      <span
        className={cn(
          "whitespace-nowrap transition-[opacity,transform] duration-150",
          showLabel
            ? "translate-x-0 opacity-100"
            : "pointer-events-none -translate-x-1 opacity-0",
        )}
      >
        {label}
      </span>
    </Link>
  );

  if (open) {
    return content;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>{content}</TooltipTrigger>
      <TooltipContent side="right" sideOffset={10}>
        {label}
      </TooltipContent>
    </Tooltip>
  );
}
