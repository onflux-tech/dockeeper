import { useMemo } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { changelog } from "@/data/changelog";
import type { ChangeArea, ChangeType, ChangelogChange } from "@/types/docs";

// ── Type display config ─────────────────────────────────────────────────────

const TYPE_META: Record<ChangeType, { emoji: string; label: string }> = {
  feat: { emoji: "\u{1F680}", label: "Features" },
  fix: { emoji: "\u{1F41B}", label: "Fixes" },
  perf: { emoji: "\u26A1", label: "Performance" },
  breaking: { emoji: "\u{1F4A5}", label: "Breaking Changes" },
  refactor: { emoji: "\u{1F527}", label: "Refactoring" },
  docs: { emoji: "\u{1F4DD}", label: "Documentation" },
};

const TYPE_ORDER: ChangeType[] = ["breaking", "feat", "fix", "perf", "refactor", "docs"];

// ── Area badge colors ───────────────────────────────────────────────────────

const AREA_STYLE: Record<ChangeArea, string> = {
  manager: "border-blue-400/30 text-blue-400",
  agent: "border-purple-400/30 text-purple-400",
  web: "border-amber-400/30 text-amber-400",
  api: "border-green-400/30 text-green-400",
  compose: "border-teal-400/30 text-teal-400",
  infra: "border-zinc-400/30 text-zinc-400",
};

// ── Props ───────────────────────────────────────────────────────────────────

interface ChangelogDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

// ── Helpers ─────────────────────────────────────────────────────────────────

function groupByType(changes: ChangelogChange[]) {
  const groups = new Map<ChangeType, ChangelogChange[]>();
  for (const c of changes) {
    const existing = groups.get(c.type);
    if (existing) {
      existing.push(c);
    } else {
      groups.set(c.type, [c]);
    }
  }
  return groups;
}

// ── Component ───────────────────────────────────────────────────────────────

export function ChangelogDialog({ open, onOpenChange }: ChangelogDialogProps) {
  const sorted = useMemo(
    () => [...changelog].sort((a, b) => b.version.localeCompare(a.version, undefined, { numeric: true })),
    [],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl p-0">
        <DialogHeader className="px-6 pt-6 pb-4">
          <DialogTitle>Changelog</DialogTitle>
        </DialogHeader>

        <ScrollArea className="max-h-[70vh] px-6 pb-6">
          {sorted.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">
              No changelog entries yet
            </p>
          ) : (
            sorted.map((entry, i) => {
              const grouped = groupByType(entry.changes);

              return (
                <div key={entry.version}>
                  {i > 0 && <Separator className="my-6" />}

                  {/* Version heading */}
                  <div className="mb-4 flex items-baseline gap-3">
                    <h3 className="text-lg font-bold">v{entry.version}</h3>
                    <span className="text-sm text-muted-foreground">{entry.date}</span>
                  </div>

                  {/* Changes grouped by type */}
                  <div className="space-y-4">
                    {TYPE_ORDER.filter((t) => grouped.has(t)).map((type) => {
                      const meta = TYPE_META[type];
                      const items = grouped.get(type)!;

                      return (
                        <div key={type}>
                          <h4 className="mb-2 text-sm font-semibold">
                            {meta.emoji} {meta.label}
                          </h4>
                          <ul className="space-y-1.5">
                            {items.map((change, ci) => (
                              <li key={ci} className="flex items-start gap-2 text-sm">
                                <Badge
                                  variant="outline"
                                  className={`mt-0.5 shrink-0 text-[10px] ${AREA_STYLE[change.area]}`}
                                >
                                  {change.area}
                                </Badge>
                                <span className="text-muted-foreground">
                                  {change.description}
                                </span>
                              </li>
                            ))}
                          </ul>
                        </div>
                      );
                    })}
                  </div>
                </div>
              );
            })
          )}
        </ScrollArea>
      </DialogContent>
    </Dialog>
  );
}
