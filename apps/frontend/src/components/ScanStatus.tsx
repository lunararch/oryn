import { useQueryClient } from "@tanstack/react-query";
import { RefreshCw } from "lucide-react";
import { useEffect, useRef } from "react";
import { useCurrentScan, useStartScan } from "../lib/api/queries";
import { Button } from "./ui/Button";

interface ScanStatusProps {
  compact?: boolean;
}

export function ScanStatus({ compact = false }: ScanStatusProps) {
  const queryClient = useQueryClient();
  const scan = useCurrentScan();
  const startScan = useStartScan();
  const wasRunning = useRef(false);

  useEffect(() => {
    const running = Boolean(scan.data?.running);
    if (wasRunning.current && !running) {
      void queryClient.invalidateQueries({ queryKey: ["games"] });
      void queryClient.invalidateQueries({ queryKey: ["stats"] });
      void queryClient.invalidateQueries({ queryKey: ["libraries"] });
      void queryClient.invalidateQueries({ queryKey: ["scans"] });
    }
    wasRunning.current = running;
  }, [queryClient, scan.data?.running]);

  const running = Boolean(scan.data?.running);
  const status = scan.data?.status;
  const hasResult = Boolean(scan.data?.id);
  const failed = status === "failed" || status === "partial" || status === "cancelled";

  if (compact) {
    return (
      <div className="flex items-center gap-2.5">
        <span
          className={`h-2 w-2 rounded-full ${
            scan.isError ? "bg-rust" : running ? "bg-brass animate-pulse" : "bg-emerald-400/80"
          }`}
          aria-hidden="true"
        />
        <span className="text-paper-muted hidden text-xs sm:inline">
          {scan.isError ? "Server offline" : running ? "Scanning library" : "Library ready"}
        </span>
        <Button
          size="sm"
          variant="secondary"
          onClick={() => startScan.mutate(false)}
          disabled={running || startScan.isPending}
        >
          <RefreshCw size={14} className={running ? "animate-spin" : ""} />
          <span className="hidden md:inline">{running ? "Scanning" : "Scan"}</span>
        </Button>
      </div>
    );
  }

  return (
    <div className="surface px-5 py-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <span
            className={`grid h-9 w-9 place-items-center rounded-full border ${
              running
                ? "border-brass/30 bg-brass/10 text-brass"
                : failed
                  ? "border-rust/30 bg-rust/10 text-rust-soft"
                  : "border-emerald-400/20 bg-emerald-400/10 text-emerald-200"
            }`}
          >
            <RefreshCw size={16} className={running ? "animate-spin" : ""} />
          </span>
          <div>
            <p className="text-paper text-sm font-medium">
              {scan.isError
                ? "Server unavailable"
                : running
                  ? "Scanning the library"
                  : hasResult
                    ? scanSummary(status)
                    : "No scan yet"}
            </p>
            <p className="text-paper-muted mt-0.5 text-xs">
              {scan.isError
                ? "Check that the Oryn server is running."
                : running
                  ? "Reading game folders safely. This can take a moment over SMB."
                  : hasResult
                    ? scan.data?.summary || "The library index is up to date."
                    : "Run a scan to discover your games."}
            </p>
          </div>
        </div>
        <Button
          variant={running ? "secondary" : "primary"}
          onClick={() => startScan.mutate(false)}
          disabled={running || startScan.isPending}
        >
          <RefreshCw size={15} className={running ? "animate-spin" : ""} />
          {running ? "Scanning…" : "Scan library"}
        </Button>
      </div>
      {startScan.isError ? (
        <p className="text-rust-soft mt-3 text-xs" role="alert">
          {startScan.error.message}
        </p>
      ) : null}
    </div>
  );
}

function scanSummary(status?: string): string {
  switch (status) {
    case "completed":
      return "Scan complete";
    case "partial":
      return "Scan completed with warnings";
    case "failed":
      return "Scan did not complete";
    case "cancelled":
      return "Scan cancelled";
    default:
      return "Library status";
  }
}
