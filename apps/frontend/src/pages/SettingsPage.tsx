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
        <p className="mt-3 max-w-2xl text-sm leading-relaxed text-paper-muted">
          Oryn keeps server configuration in environment variables so deployment stays explicit.
          This page shows the active library and lets you run safe, repeatable scans.
        </p>
      </div>

      <section className="grid gap-4 md:grid-cols-3">
        <InfoCard
          icon={HardDrive}
          label="Library"
          value={library?.name ?? "Not configured"}
          detail={library ? `${formatNumber(library.gameCount)} games indexed` : "Set ORYN_LIBRARY_ROOT"}
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
            <h2 className="font-display text-2xl text-paper">Manual scan</h2>
            <p className="mt-2 max-w-2xl text-sm leading-relaxed text-paper-muted">
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

        <label className="mt-5 flex items-start gap-3 rounded-2xl border border-rust/20 bg-rust/[0.045] px-4 py-3 text-sm">
          <input
            type="checkbox"
            checked={allowMissing}
            onChange={(event) => setAllowMissing(event.target.checked)}
            className="mt-1 h-4 w-4 accent-[#c9a86a]"
          />
          <span>
            <span className="font-medium text-paper">Allow mass-missing reconciliation</span>
            <span className="mt-1 block leading-relaxed text-paper-muted">
              Only use this after deliberately removing a large part of the library. Without it,
              Oryn refuses to mark more than 25% of known games missing in one scan.
            </span>
          </span>
        </label>

        {startScan.isError ? (
          <p className="mt-4 text-sm text-rust-soft" role="alert">
            {startScan.error.message}
          </p>
        ) : null}
      </section>

      <section className="surface overflow-hidden">
        <div className="border-b border-white/8 px-6 py-5">
          <h2 className="font-display text-2xl text-paper">Scan history</h2>
          <p className="mt-1 text-sm text-paper-muted">The most recent 25 scan runs.</p>
        </div>

        {scans.isLoading ? (
          <p className="px-6 py-8 text-sm text-paper-muted">Loading scan history…</p>
        ) : scans.isError ? (
          <p className="px-6 py-8 text-sm text-rust-soft" role="alert">
            Could not load scan history: {scans.error.message}
          </p>
        ) : scans.data && scans.data.items.length === 0 ? (
          <p className="px-6 py-8 text-sm text-paper-muted">No scans have run yet.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[42rem] text-left text-sm">
              <thead className="border-b border-white/8 text-[0.67rem] uppercase tracking-[0.13em] text-paper-faint">
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
                    <td className="whitespace-nowrap px-6 py-3.5 text-paper">
                      {formatDate(run.startedAt)}
                    </td>
                    <td className="px-4 py-3.5">
                      <ScanBadge status={run.status} />
                    </td>
                    <td className="px-4 py-3.5 text-right tabular-nums text-paper-muted">
                      {run.discovered}
                    </td>
                    <td className="px-4 py-3.5 text-right tabular-nums text-paper-muted">
                      {run.added}
                    </td>
                    <td className="px-4 py-3.5 text-right tabular-nums text-paper-muted">
                      {run.missing}
                    </td>
                    <td className="px-4 py-3.5 text-right tabular-nums text-paper-muted">
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
        <h2 className="font-display text-2xl text-paper">About this build</h2>
        <dl className="mt-4 grid gap-4 text-sm sm:grid-cols-2">
          <div>
            <dt className="text-[0.67rem] uppercase tracking-[0.13em] text-paper-faint">API</dt>
            <dd className="mt-1 font-mono text-xs text-paper">/api/v1</dd>
          </div>
          <div>
            <dt className="text-[0.67rem] uppercase tracking-[0.13em] text-paper-faint">Storage</dt>
            <dd className="mt-1 text-paper">SQLite + local artwork cache</dd>
          </div>
          <div>
            <dt className="text-[0.67rem] uppercase tracking-[0.13em] text-paper-faint">
              Desktop client
            </dt>
            <dd className="mt-1 text-paper-muted">Planned · Wails 3</dd>
          </div>
          <div>
            <dt className="text-[0.67rem] uppercase tracking-[0.13em] text-paper-faint">Network</dt>
            <dd className="mt-1 text-paper-muted">Trusted LAN only · no public exposure</dd>
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
      <div className="flex items-center gap-2 text-brass">
        <Icon size={16} />
        <span className="text-[0.67rem] font-semibold uppercase tracking-[0.15em]">{label}</span>
      </div>
      <p className="mt-3 font-display text-xl text-paper">{value}</p>
      <p className="mt-1 text-xs text-paper-muted">{detail}</p>
    </div>
  );
}
