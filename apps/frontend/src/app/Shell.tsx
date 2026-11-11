import { LayoutDashboard, LibraryBig, Settings2, Sparkles } from "lucide-react";
import { NavLink, Outlet } from "react-router-dom";
import { ScanStatus } from "../components/ScanStatus";

const navigation = [
  { to: "/", label: "Dashboard", icon: LayoutDashboard, end: true },
  { to: "/library", label: "Library", icon: LibraryBig, end: false },
  { to: "/settings", label: "Settings", icon: Settings2, end: false },
];

function navClass(active: boolean): string {
  return `group flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm transition ${
    active
      ? "bg-brass/10 text-brass-soft"
      : "text-paper-muted hover:bg-white/[0.045] hover:text-paper"
  }`;
}

export function Shell() {
  return (
    <div className="min-h-screen lg:grid lg:grid-cols-[17rem_minmax(0,1fr)]">
      <aside className="bg-ink-950/55 hidden border-r border-white/7 lg:sticky lg:top-0 lg:flex lg:h-screen lg:flex-col lg:px-5 lg:py-6">
        <Brand />
        <nav aria-label="Main navigation" className="mt-10 flex flex-col gap-1">
          {navigation.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) => navClass(isActive)}
            >
              <item.icon size={18} strokeWidth={1.7} />
              {item.label}
            </NavLink>
          ))}
        </nav>

        <div className="mt-auto space-y-3">
          <div className="rounded-2xl border border-white/8 bg-white/[0.025] p-4">
            <div className="text-brass flex items-center gap-2">
              <Sparkles size={15} />
              <span className="text-[0.68rem] font-semibold tracking-[0.18em] uppercase">
                A private shelf
              </span>
            </div>
            <p className="text-paper-muted mt-2 text-xs leading-relaxed">
              Your library stays on your network. The server indexes, your desktop plays.
            </p>
          </div>
          <p className="text-paper-faint px-1 text-[0.65rem] tracking-[0.18em] uppercase">
            Oryn · early access
          </p>
        </div>
      </aside>

      <div className="min-w-0">
        <header className="bg-ink-950/80 sticky top-0 z-30 border-b border-white/7 backdrop-blur-xl lg:hidden">
          <div className="flex items-center justify-between gap-3 px-4 py-3">
            <Brand compact />
            <ScanStatus compact />
          </div>
          <nav aria-label="Mobile navigation" className="flex gap-1 overflow-x-auto px-3 pb-3">
            {navigation.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.end}
                className={({ isActive }) =>
                  `flex shrink-0 items-center gap-2 rounded-full px-3.5 py-2 text-xs font-medium transition ${
                    isActive
                      ? "bg-brass/12 text-brass-soft"
                      : "text-paper-muted hover:text-paper hover:bg-white/5"
                  }`
                }
              >
                <item.icon size={15} strokeWidth={1.8} />
                {item.label}
              </NavLink>
            ))}
          </nav>
        </header>

        <main className="mx-auto w-full max-w-[92rem] px-4 pt-8 pb-20 sm:px-6 lg:px-10 lg:pt-10">
          <Outlet />
        </main>
      </div>
    </div>
  );
}

function Brand({ compact = false }: { compact?: boolean }) {
  return (
    <NavLink
      to="/"
      className="focus-visible:ring-brass flex items-center gap-3 rounded-xl outline-none focus-visible:ring-2"
      aria-label="Oryn home"
    >
      <span className="border-brass/30 bg-brass/10 font-display text-brass-soft grid h-10 w-10 place-items-center rounded-xl border text-xl">
        O
      </span>
      {!compact ? (
        <span>
          <span className="font-display text-paper block text-[1.35rem] leading-none tracking-[0.02em]">
            Oryn
          </span>
          <span className="text-paper-faint mt-1 block text-[0.62rem] tracking-[0.28em] uppercase">
            Visual novel library
          </span>
        </span>
      ) : (
        <span className="font-display text-paper text-xl">Oryn</span>
      )}
    </NavLink>
  );
}
