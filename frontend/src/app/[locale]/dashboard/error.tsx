"use client";

import { RotateCcw } from "lucide-react";
import { useEffect } from "react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  errorToTelemetryPayload,
  reportClientError,
} from "@/lib/telemetry/client-error";

export default function DashboardError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    reportClientError(errorToTelemetryPayload("app-error-boundary", error));
  }, [error]);

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
