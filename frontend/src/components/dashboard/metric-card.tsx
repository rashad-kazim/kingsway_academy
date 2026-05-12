import type { LucideIcon } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

type MetricCardProps = {
  label: string;
  value: number | string;
  icon: LucideIcon;
};

export function MetricCard({ label, value, icon: Icon }: MetricCardProps) {
  return (
    <Card className="rounded-lg">
      <CardHeader className="flex-row items-center justify-between space-y-0">
        <CardTitle className="text-sm text-muted-foreground">{label}</CardTitle>
        <Icon className="size-4 text-kw-c-ef2334 dark:text-kw-c-ff3b4f" />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-semibold tabular-nums">{value}</div>
      </CardContent>
    </Card>
  );
}
