"use client";

import { RotateCcw } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export default function DashboardError({ reset }: { reset: () => void }) {
  return (
    <Card className="rounded-lg">
      <CardHeader>
        <CardTitle>Dashboard unavailable</CardTitle>
        <CardDescription>
          The backend request did not complete successfully.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Button onClick={() => reset()} type="button">
          <RotateCcw />
          Retry
        </Button>
      </CardContent>
    </Card>
  );
}
