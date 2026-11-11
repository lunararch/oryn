import { Database, HardDrive, RefreshCw, ShieldCheck } from "lucide-react";
import { useState } from "react";
import { ScanStatus } from "../components/ScanStatus";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { useLibraries, useScanHistory, useStartScan } from "../lib/api/queries";
import { formatDate, formatNumber } from "../lib/format";

export function SettingsPage() {
  const libraries = useLibraries();
  const scans = useScanHistory();
  const startScan = useStartScan();
  const [allowMissing, setAllowMissing] = useState(false);
  const library = libraries.data?.items[0];

  return (
    <div className="space-y-8">
      <div>
        <p className="eyebrow">Server & library</p>
        <h1 className="page-title mt-2">Settings</h1>
        <p className="text-paper-muted mt-3 max-w-2xl text-sm leading-relaxed">
          Oryn keeps server configuration in environment variables so deployment stays explicit.
          This page shows the active library and lets you run safe, repeatable scans.
        </p>
      </div>

      <section className="grid gap-4 md:grid-cols-3">
        <InfoCard
          icon={HardDrive}
          label="Library"
          value={library?.name ?? "Not configured"}
          detail={
            library ? `${formatNumber(library.gameCount)} games indexed` : "Set ORYN_LIBRARY_ROOT"
          }
        />
        <InfoCard
          icon={Database}
          label="Last successful scan"
          value={library?.lastScanAt ? formatDate(library.lastScanAt) : "Never"}
          detail="SQLite remains the source of truth"
        />
        <InfoCard
          icon={ShieldCheck}
          label="Game share"
          value="Read-only"
          detail="Oryn never writes to game folders"
        />
      </section>

      <ScanStatus />

      <section className="surface px-6 py-6">
        <div className="flex flex-wrap items-start justify-between gap-5">
          <div>
            <h2 className="font-display text-paper text-2xl">Manual scan</h2>
            <p className="text-paper-muted mt-2 max-w-2xl text-sm leading-relaxed">
              Scans are idempotent: adding, renaming, or updating game folders does not duplicate
              records or erase your metadata. Missing games are only reconciled after a clean scan
              with no traversal errors.
            </p>
          </div>
          <Button
            variant="primary"
            onClick={() => startScan.mutate(allowMissing)}
            disabled={startScan.isPending}
          >
            <RefreshCw size={15} className={startScan.isPending ? "animate-spin" : ""} />
            {startScan.isPending ? "Starting…" : "Start scan"}
          </Button>
        </div>

        <label className="border-rust/20 bg-rust/[0.045] mt-5 flex items-start gap-3 rounded-2xl border px-4 py-3 text-sm">
          <input
            type="checkbox"
            checked={allowMissing}
            onChange={(event) => setAllowMissing(event.target.checked)}
            className="mt-1 h-4 w-4 accent-[#c9a86a]"
          />
          <span>
            <span className="text-paper font-medium">Allow mass-missing reconciliation</span>
            <span className="text-paper-muted mt-1 block leading-relaxed">
              Only use this after deliberately removing a large part of the library. Without it,
              Oryn refuses to mark more than 25% of known games missing in one scan.
            </span>
          </span>
        </label>

        {startScan.isError ? (
          <p className="text-rust-soft mt-4 text-sm" role="alert">
            {startScan.error.message}
          </p>
        ) : null}
      </section>

      <section className="surface overflow-hidden">
        <div className="border-b border-white/8 px-6 py-5">
          <h2 className="font-display text-paper text-2xl">Scan history</h2>
          <p className="text-paper-muted mt-1 text-sm">The most recent 25 scan runs.</p>
        </div>

        {scans.isLoading ? (
          <p className="text-paper-muted px-6 py-8 text-sm">Loading scan history…</p>
        ) : scans.isError ? (
          <p className="text-rust-soft px-6 py-8 text-sm" role="alert">
            Could not load scan history: {scans.error.message}
          </p>
        ) : scans.data && scans.data.items.length === 0 ? (
          <p className="text-paper-muted px-6 py-8 text-sm">No scans have run yet.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[42rem] text-left text-sm">
              <thead className="text-paper-faint border-b border-white/8 text-[0.67rem] tracking-[0.13em] uppercase">
                <tr>
                  <th className="px-6 py-3 font-medium">Started</th>
                  <th className="px-4 py-3 font-medium">Status</th>
                  <th className="px-4 py-3 text-right font-medium">Found</th>
                  <th className="px-4 py-3 text-right font-medium">Added</th>
                  <th className="px-4 py-3 text-right font-medium">Missing</th>
                  <th className="px-4 py-3 text-right font-medium">Errors</th>
                </tr>
              </thead>
              <tbody>
                {scans.data?.items.map((run) => (
                  <tr key={run.id} className="border-b border-white/6 last:border-0">
                    <td className="text-paper px-6 py-3.5 whitespace-nowrap">
                      {formatDate(run.startedAt)}
                    </td>
                    <td className="px-4 py-3.5">
                      <ScanBadge status={run.status} />
                    </td>
                    <td className="text-paper-muted px-4 py-3.5 text-right tabular-nums">
                      {run.discovered}
                    </td>
                    <td className="text-paper-muted px-4 py-3.5 text-right tabular-nums">
                      {run.added}
                    </td>
                    <td className="text-paper-muted px-4 py-3.5 text-right tabular-nums">
                      {run.missing}
                    </td>
                    <td className="text-paper-muted px-4 py-3.5 text-right tabular-nums">
                      {run.errors}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="surface px-6 py-6">
        <h2 className="font-display text-paper text-2xl">About this build</h2>
        <dl className="mt-4 grid gap-4 text-sm sm:grid-cols-2">
          <div>
            <dt className="text-paper-faint text-[0.67rem] tracking-[0.13em] uppercase">API</dt>
            <dd className="text-paper mt-1 font-mono text-xs">/api/v1</dd>
          </div>
          <div>
            <dt className="text-paper-faint text-[0.67rem] tracking-[0.13em] uppercase">Storage</dt>
            <dd className="text-paper mt-1">SQLite + local artwork cache</dd>
          </div>
          <div>
            <dt className="text-paper-faint text-[0.67rem] tracking-[0.13em] uppercase">
              Desktop client
            </dt>
            <dd className="text-paper-muted mt-1">Planned · Wails 3</dd>
          </div>
          <div>
            <dt className="text-paper-faint text-[0.67rem] tracking-[0.13em] uppercase">Network</dt>
            <dd className="text-paper-muted mt-1">Trusted LAN only · no public exposure</dd>
          </div>
        </dl>
      </section>
    </div>
  );
}

function ScanBadge({ status }: { status: string }) {
  switch (status) {
    case "completed":
      return <Badge tone="success">Completed</Badge>;
    case "partial":
      return <Badge tone="brass">Partial</Badge>;
    case "failed":
      return <Badge tone="danger">Failed</Badge>;
    case "cancelled":
      return <Badge tone="quiet">Cancelled</Badge>;
    default:
      return <Badge tone="quiet">{status}</Badge>;
  }
}

function InfoCard({
  icon: Icon,
  label,
  value,
  detail,
}: {
  icon: typeof HardDrive;
  label: string;
  value: string;
  detail: string;
}) {
  return (
    <div className="surface px-5 py-5">
      <div className="text-brass flex items-center gap-2">
        <Icon size={16} />
        <span className="text-[0.67rem] font-semibold tracking-[0.15em] uppercase">{label}</span>
      </div>
      <p className="font-display text-paper mt-3 text-xl">{value}</p>
      <p className="text-paper-muted mt-1 text-xs">{detail}</p>
    </div>
  );
}
