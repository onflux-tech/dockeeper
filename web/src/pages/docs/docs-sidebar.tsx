import { useMemo, useState } from "react";
import { BookOpen, Search } from "lucide-react";
import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import type { ApiGroup, HttpMethod } from "@/types/docs";

// ── Method dot colors ───────────────────────────────────────────────────────

const METHOD_DOT: Record<HttpMethod, string> = {
  GET: "bg-blue-400",
  POST: "bg-green-400",
  PUT: "bg-amber-400",
  PATCH: "bg-purple-400",
  DELETE: "bg-red-400",
};

// ── Props ───────────────────────────────────────────────────────────────────

interface DocsSidebarProps {
  groups: ApiGroup[];
  activeId: string | null;
  onNavigate: (id: string) => void;
  mobileOpen: boolean;
  onMobileClose: () => void;
}

// ── Sidebar content (shared between desktop & mobile) ───────────────────────

function SidebarNav({
  groups,
  activeId,
  onNavigate,
  search,
  onSearchChange,
}: {
  groups: ApiGroup[];
  activeId: string | null;
  onNavigate: (id: string) => void;
  search: string;
  onSearchChange: (value: string) => void;
}) {
  const filtered = useMemo(() => {
    const q = search.toLowerCase().trim();
    if (!q) return groups;

    return groups
      .map((group) => {
        // Groups with no endpoints (e.g. Getting Started) — match group title
        if (group.endpoints.length === 0) {
          return group.title.toLowerCase().includes(q) ? group : null;
        }

        const matchingEndpoints = group.endpoints.filter(
          (ep) =>
            ep.title.toLowerCase().includes(q) ||
            ep.path.toLowerCase().includes(q) ||
            ep.method.toLowerCase().includes(q),
        );

        if (matchingEndpoints.length === 0) return null;
        return { ...group, endpoints: matchingEndpoints };
      })
      .filter(Boolean) as ApiGroup[];
  }, [groups, search]);

  return (
    <div className="flex h-full flex-col">
      {/* Search */}
      <div className="border-b border-border p-3">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search endpoints..."
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
            className="h-8 pl-8 text-xs"
          />
        </div>
      </div>

      {/* Groups */}
      <ScrollArea className="flex-1">
        <nav className="p-3">
          {filtered.length === 0 && (
            <p className="px-3 py-8 text-center text-xs text-muted-foreground">
              No results found
            </p>
          )}

          {filtered.map((group, gi) => (
            <div key={group.id}>
              {gi > 0 && <Separator className="my-3" />}

              {/* Group with endpoints → header + items */}
              {group.endpoints.length > 0 ? (
                <>
                  <p className="mb-2 px-3 text-xs font-bold uppercase tracking-widest text-muted-foreground">
                    {group.title}
                  </p>
                  <div className="flex flex-col gap-0.5">
                    {group.endpoints.map((ep) => {
                      const isActive = activeId === ep.id;
                      return (
                        <button
                          key={ep.id}
                          onClick={() => onNavigate(ep.id)}
                          className={cn(
                            "flex w-full items-center gap-2.5 rounded-md px-3 py-1.5 text-left text-sm transition-all duration-150",
                            isActive
                              ? "bg-primary/10 font-medium text-primary"
                              : "text-muted-foreground hover:bg-primary/5 hover:text-foreground",
                          )}
                        >
                          {/* Method dot */}
                          <span
                            className={cn(
                              "h-1.5 w-1.5 shrink-0 rounded-full",
                              METHOD_DOT[ep.method],
                            )}
                          />

                          <span className="truncate">{ep.title}</span>

                          {/* Active glow dot */}
                          {isActive && (
                            <span className="ml-auto h-1.5 w-1.5 shrink-0 rounded-full bg-primary shadow-[0_0_10px_oklch(0.723_0.219_149.579/0.5)]" />
                          )}
                        </button>
                      );
                    })}
                  </div>
                </>
              ) : (
                /* Group with no endpoints (Getting Started) → clickable item */
                <button
                  onClick={() => onNavigate(group.id)}
                  className={cn(
                    "flex w-full items-center gap-2.5 rounded-md px-3 py-1.5 text-left text-sm transition-all duration-150",
                    activeId === group.id
                      ? "bg-primary/10 font-medium text-primary"
                      : "text-muted-foreground hover:bg-primary/5 hover:text-foreground",
                  )}
                >
                  <BookOpen className="h-3.5 w-3.5 shrink-0" />
                  <span>{group.title}</span>
                  {activeId === group.id && (
                    <span className="ml-auto h-1.5 w-1.5 shrink-0 rounded-full bg-primary shadow-[0_0_10px_oklch(0.723_0.219_149.579/0.5)]" />
                  )}
                </button>
              )}
            </div>
          ))}
        </nav>
      </ScrollArea>
    </div>
  );
}

// ── Main component ──────────────────────────────────────────────────────────

export function DocsSidebar({
  groups,
  activeId,
  onNavigate,
  mobileOpen,
  onMobileClose,
}: DocsSidebarProps) {
  const [search, setSearch] = useState("");

  const handleNavigate = (id: string) => {
    onNavigate(id);
    onMobileClose();
  };

  return (
    <>
      {/* Desktop sidebar */}
      <aside className="docs-sidebar hidden w-72 shrink-0 border-r border-border lg:block">
        <div className="sticky top-16 h-[calc(100vh-4rem)]">
          <SidebarNav
            groups={groups}
            activeId={activeId}
            onNavigate={onNavigate}
            search={search}
            onSearchChange={setSearch}
          />
        </div>
      </aside>

      {/* Mobile sidebar */}
      <Sheet open={mobileOpen} onOpenChange={(open) => !open && onMobileClose()}>
        <SheetContent side="left" className="w-72 p-0">
          <SheetHeader className="flex h-14 items-center gap-2 border-b border-border px-4">
            <SheetTitle className="text-sm font-semibold">
              API Reference
            </SheetTitle>
          </SheetHeader>
          <div className="h-[calc(100vh-3.5rem)]">
            <SidebarNav
              groups={groups}
              activeId={activeId}
              onNavigate={handleNavigate}
              search={search}
              onSearchChange={setSearch}
            />
          </div>
        </SheetContent>
      </Sheet>
    </>
  );
}
