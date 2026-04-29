import Link from "next/link";
import {
  Bell,
  Building2,
  CalendarDays,
  CreditCard,
  Files,
  GraduationCap,
  LayoutDashboard,
  LogOut,
  ShieldCheck,
  UserRound,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import type { Session } from "@/lib/api/types";
import { logoutAction } from "@/lib/auth/actions";
import { dashboardPathForRole, roleLabel } from "@/lib/navigation/roles";

type AppShellProps = {
  children: React.ReactNode;
  locale: string;
  session: Session;
};

const navigation = [
  { href: "/dashboard/owner", label: "Owner", icon: ShieldCheck, roles: ["owner"] },
  {
    href: "/dashboard/receptionist",
    label: "Reception",
    icon: LayoutDashboard,
    roles: ["receptionist"],
  },
  { href: "/dashboard/teacher", label: "Classes", icon: GraduationCap, roles: ["teacher"] },
  { href: "/dashboard/student", label: "Progress", icon: UserRound, roles: ["student"] },
  {
    href: "/dashboard",
    label: "Schedule",
    icon: CalendarDays,
    roles: ["owner", "receptionist", "teacher", "student"],
  },
  {
    href: "/dashboard",
    label: "Finance",
    icon: CreditCard,
    roles: ["owner", "receptionist", "student"],
  },
  {
    href: "/dashboard",
    label: "Files",
    icon: Files,
    roles: ["owner", "receptionist", "teacher"],
  },
  {
    href: "/dashboard",
    label: "Alerts",
    icon: Bell,
    roles: ["owner", "receptionist", "teacher", "student"],
  },
] as const;

export function AppShell({ children, locale, session }: AppShellProps) {
  const userName = `${session.user.first_name} ${session.user.last_name}`;
  const roleHome = dashboardPathForRole(session.user.role);

  return (
    <div className="min-h-svh bg-background text-foreground">
      <aside className="fixed inset-y-0 left-0 hidden w-64 border-r bg-sidebar text-sidebar-foreground lg:flex lg:flex-col">
        <div className="flex h-14 items-center gap-2 px-4">
          <div className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <Building2 className="size-4" />
          </div>
          <div className="min-w-0">
            <div className="truncate text-sm font-semibold">Kingsway</div>
            <div className="truncate text-xs text-muted-foreground">
              {session.branch?.name ?? roleLabel(session.user.role)}
            </div>
          </div>
        </div>
        <Separator />
        <nav className="flex flex-1 flex-col gap-1 p-3">
          {navigation
            .filter((item) =>
              (item.roles as readonly string[]).includes(session.user.role),
            )
            .map((item) => {
              const Icon = item.icon;
              const href = item.href === "/dashboard" ? roleHome : item.href;
              return (
                <Button
                  key={`${item.label}-${href}`}
                  asChild
                  variant="ghost"
                  className="justify-start"
                >
                  <Link href={`/${locale}${href}`}>
                    <Icon />
                    {item.label}
                  </Link>
                </Button>
              );
            })}
        </nav>
        <div className="space-y-3 border-t p-4">
          <div className="min-w-0">
            <div className="truncate text-sm font-medium">{userName}</div>
            <div className="truncate text-xs text-muted-foreground">
              {session.user.email}
            </div>
          </div>
          <form action={logoutAction}>
            <input name="locale" type="hidden" value={locale} />
            <Button className="w-full justify-start" type="submit" variant="outline">
              <LogOut />
              Sign out
            </Button>
          </form>
        </div>
      </aside>

      <div className="lg:pl-64">
        <header className="sticky top-0 z-20 flex h-14 items-center justify-between border-b bg-background/95 px-4 backdrop-blur lg:px-6">
          <div className="min-w-0">
            <div className="truncate text-sm font-semibold">Kingsway</div>
            <div className="truncate text-xs text-muted-foreground lg:hidden">
              {session.branch?.name ?? roleLabel(session.user.role)}
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Badge variant="outline">{roleLabel(session.user.role)}</Badge>
            <form action={logoutAction} className="lg:hidden">
              <input name="locale" type="hidden" value={locale} />
              <Button size="icon" type="submit" variant="ghost" aria-label="Sign out">
                <LogOut />
              </Button>
            </form>
          </div>
        </header>
        <nav className="flex gap-2 overflow-x-auto border-b p-2 lg:hidden">
          {navigation
            .filter((item) =>
              (item.roles as readonly string[]).includes(session.user.role),
            )
            .map((item) => {
              const Icon = item.icon;
              const href = item.href === "/dashboard" ? roleHome : item.href;
              return (
                <Button
                  key={`mobile-${item.label}-${href}`}
                  asChild
                  size="sm"
                  variant="ghost"
                >
                  <Link href={`/${locale}${href}`}>
                    <Icon />
                    {item.label}
                  </Link>
                </Button>
              );
            })}
        </nav>
        <main className="mx-auto flex w-full max-w-7xl flex-col gap-6 p-4 lg:p-6">
          {children}
        </main>
      </div>
    </div>
  );
}
