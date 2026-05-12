import type { LucideIcon } from "lucide-react";
import {
  Avatar,
  AvatarFallback,
  AvatarGroup,
  AvatarImage,
} from "@/components/ui/avatar";
import { Card, CardContent } from "@/components/ui/card";
import type { Branch, StaffMember } from "@/lib/api/types";

export function MiniStat({
  icon: Icon,
  label,
  value,
}: {
  icon: LucideIcon;
  label: string;
  value: string;
}) {
  return (
    <Card className="rounded-lg border-kw-c-dce3ee dark:border-kw-c-3a4658 dark:bg-kw-c-1b2635">
      <CardContent className="p-5">
        <div className="flex items-center justify-between">
          <div className="text-sm font-bold text-kw-c-59667a dark:text-kw-c-a7b0bf">
            {label}
          </div>
          <Icon className="size-5 text-kw-c-ef2334 dark:text-kw-c-ff3b4f" />
        </div>
        <div className="mt-4 text-2xl font-black tabular-nums">{value}</div>
      </CardContent>
    </Card>
  );
}

export function StaffCell({
  label,
  receptionistLabel,
  staff,
}: {
  label: string;
  receptionistLabel: string;
  staff: StaffMember[];
}) {
  if (staff.length > 0) {
    return (
      <div className="flex items-center gap-3">
        <AvatarGroup>
          {staff.slice(0, 3).map((member) => (
            <Avatar
              className="border border-white/60 dark:border-kw-c-414d60"
              key={member.id}
              size="sm"
            >
              {member.profile_photo_url ? (
                <AvatarImage
                  alt={`${member.first_name} ${member.last_name}`}
                  src={member.profile_photo_url}
                />
              ) : null}
              <AvatarFallback className="bg-kw-c-eef2f7 text-kw-10 font-black text-kw-c-0a284b dark:bg-kw-c-2a3444 dark:text-kw-c-f3f6fa">
                {staffInitials(member)}
              </AvatarFallback>
            </Avatar>
          ))}
        </AvatarGroup>
        <span className="text-sm font-semibold text-kw-c-59667a dark:text-kw-c-a7b0bf">
          {staff.length} {receptionistLabel}
        </span>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-3">
      <AvatarGroup>
        <Avatar size="sm">
          <AvatarFallback className="bg-kw-c-eef2f7 text-kw-10 font-black text-kw-c-0a284b dark:bg-kw-c-2a3444 dark:text-kw-c-f3f6fa">
            --
          </AvatarFallback>
        </Avatar>
      </AvatarGroup>
      <span className="text-sm text-kw-c-687386 dark:text-kw-c-6f7a8a">
        {label}
      </span>
    </div>
  );
}

export function BranchProfileAvatar({ branch }: { branch: Branch }) {
  return (
    <Avatar className="size-11 border border-white/50 shadow-sm dark:border-kw-c-414d60">
      {branch.photo_url ? (
        <AvatarImage alt={branch.name} src={branch.photo_url} />
      ) : null}
      <AvatarFallback className="bg-kw-c-e8edf5 text-sm font-black text-kw-c-0a284b dark:bg-kw-c-2a3444 dark:text-kw-c-f3f6fa">
        {branchInitials(branch.name)}
      </AvatarFallback>
    </Avatar>
  );
}

function staffInitials(member: StaffMember) {
  return `${member.first_name[0] ?? ""}${member.last_name[0] ?? ""}`
    .trim()
    .toUpperCase();
}

function branchInitials(name: string) {
  return name
    .split(/\s+/)
    .map((part) => part[0])
    .join("")
    .slice(0, 2)
    .toUpperCase();
}
