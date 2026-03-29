import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { BookOpen, ChevronDown, Search } from "lucide-react";
import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
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
  const navRef = useRef<HTMLElement>(null);

  // Track which groups the user has manually toggled
  const [manualToggle, setManualToggle] = useState<Record<string, boolean>>({});

  // Derive which group contains the active endpoint
  const activeGroupId = useMemo(() => {
    if (!activeId) return null;
    for (const group of groups) {
      if (group.endpoints.some((ep) => ep.id === activeId)) return group.id;
    }
    return null;
  }, [activeId, groups]);

  const getIsOpen = useCallback(
    (groupId: string) => {
      if (manualToggle[groupId] !== undefined) return manualToggle[groupId];
      return activeGroupId === groupId;
    },
    [manualToggle, activeGroupId],
  );

  const handleToggle = useCallback((groupId: string, open: boolean) => {
    setManualToggle((prev) => ({ ...prev, [groupId]: open }));
  }, []);

  // Auto-scroll sidebar to keep the active item visible
  useEffect(() => {
    if (!activeId || !navRef.current) return;
    const activeBtn = navRef.current.querySelector<HTMLElement>(
      `[data-sidebar-id="${activeId}"]`,
    );
    if (!activeBtn) return;
    activeBtn.scrollIntoView({ behavior: "smooth", block: "nearest" });
  }, [activeId]);

  const filtered = useMemo(() => {
    const q = search.toLowerCase().trim();
    if (!q) return groups;

    return groups
      .map((group) => {
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

  const isSearching = search.trim().length > 0;

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
      <div className="flex-1 overflow-y-auto">
        <nav ref={navRef} className="space-y-1 p-2">
          {filtered.length === 0 && (
            <p className="px-3 py-8 text-center text-xs text-muted-foreground">
              No results found
            </p>
          )}

          {filtered.map((group) => {
            const Icon = group.icon;
            const hasEndpoints = group.endpoints.length > 0;
            const isOpen = isSearching || getIsOpen(group.id);
            const hasActiveChild =
              hasEndpoints &&
              group.endpoints.some((ep) => ep.id === activeId);

            if (!hasEndpoints) {
              return (
                <button
                  key={group.id}
                  data-sidebar-id={group.id}
                  onClick={() => onNavigate(group.id)}
                  className={cn(
                    "flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-left text-[13px]",
                    activeId === group.id
                      ? "bg-primary/10 font-medium text-primary"
                      : "text-muted-foreground hover:bg-muted/50 hover:text-foreground",
                  )}
                >
                  <BookOpen className="h-4 w-4 shrink-0" />
                  <span>{group.title}</span>
                  {activeId === group.id && (
                    <span className="ml-auto h-1.5 w-1.5 shrink-0 rounded-full bg-primary shadow-[0_0_10px_oklch(0.723_0.219_149.579/0.5)]" />
                  )}
                </button>
              );
            }

            return (
              <Collapsible
                key={group.id}
                open={isOpen}
                onOpenChange={(open) => handleToggle(group.id, open)}
              >
                {/* Group trigger */}
                <CollapsibleTrigger
                  className={cn(
                    "group/trigger flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-left text-[13px] font-medium",
                    hasActiveChild
                      ? "bg-primary/5 text-primary"
                      : "text-muted-foreground hover:bg-muted/50 hover:text-foreground",
                  )}
                >
                  <span
                    className={cn(
                      "flex h-6 w-6 shrink-0 items-center justify-center rounded-md border",
                      hasActiveChild
                        ? "border-primary/30 bg-primary/10 text-primary"
                        : "border-border bg-muted/50 text-muted-foreground group-hover/trigger:border-border group-hover/trigger:text-foreground",
                    )}
                  >
                    <Icon className="h-3.5 w-3.5" />
                  </span>

                  <span className="flex-1 truncate">{group.title}</span>

                  <span
                    className={cn(
                      "mr-0.5 min-w-5 text-center text-[10px] tabular-nums",
                      hasActiveChild
                        ? "text-primary/60"
                        : "text-muted-foreground/60",
                    )}
                  >
                    {group.endpoints.length}
                  </span>

                  <ChevronDown
                    className={cn(
                      "h-3.5 w-3.5 shrink-0 text-muted-foreground/50 transition-transform duration-200",
                      isOpen && "rotate-180",
                    )}
                  />
                </CollapsibleTrigger>

                {/* Endpoint list */}
                <CollapsibleContent>
                  <div className="ml-[22px] mt-0.5 flex flex-col gap-px border-l border-border pl-2.5">
                    {group.endpoints.map((ep) => {
                      const isActive = activeId === ep.id;
                      return (
                        <button
                          key={ep.id}
                          data-sidebar-id={ep.id}
                          onClick={() => onNavigate(ep.id)}
                          className={cn(
                            "flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-[13px]",
                            isActive
                              ? "bg-primary/10 font-medium text-primary"
                              : "text-muted-foreground hover:bg-muted/50 hover:text-foreground",
                          )}
                        >
                          <span
                            className={cn(
                              "h-1.5 w-1.5 shrink-0 rounded-full",
                              METHOD_DOT[ep.method],
                            )}
                          />
                          <span className="truncate">{ep.title}</span>
                          {isActive && (
                            <span className="ml-auto h-1.5 w-1.5 shrink-0 rounded-full bg-primary shadow-[0_0_10px_oklch(0.723_0.219_149.579/0.5)]" />
                          )}
                        </button>
                      );
                    })}
                  </div>
                </CollapsibleContent>
              </Collapsible>
            );
          })}
        </nav>
      </div>
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
      <aside className="docs-sidebar hidden w-72 shrink-0 border-r border-border lg:flex lg:flex-col lg:overflow-hidden">
        <SidebarNav
          groups={groups}
          activeId={activeId}
          onNavigate={onNavigate}
          search={search}
          onSearchChange={setSearch}
        />
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
