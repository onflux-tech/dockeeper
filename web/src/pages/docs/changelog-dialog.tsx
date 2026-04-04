import { useMemo } from "react";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import { changelog } from "@/data/changelog";
import type { ChangeArea, ChangeType, ChangelogChange } from "@/types/docs";

// ── Type badge config (pill style, no emojis) ──────────────────────────────

const TYPE_BADGE: Record<ChangeType, { label: string; className: string }> = {
	breaking: {
		label: "Breaking",
		className: "border-red-400/20 bg-red-400/10 text-red-400",
	},
	feat: {
		label: "New",
		className: "border-green-400/20 bg-green-400/10 text-green-400",
	},
	fix: {
		label: "Fix",
		className: "border-orange-400/20 bg-orange-400/10 text-orange-400",
	},
	perf: {
		label: "Perf",
		className: "border-purple-400/20 bg-purple-400/10 text-purple-400",
	},
	refactor: {
		label: "Refactor",
		className: "border-blue-400/20 bg-blue-400/10 text-blue-400",
	},
	docs: {
		label: "Docs",
		className: "border-zinc-400/20 bg-zinc-400/10 text-zinc-400",
	},
};

// ── Area display labels ─────────────────────────────────────────────────────

const AREA_LABEL: Record<ChangeArea, string> = {
	manager: "Manager",
	agent: "Agent",
	web: "Web",
	api: "API",
	compose: "Compose",
	infra: "Infrastructure",
};

const AREA_ORDER: ChangeArea[] = ["manager", "agent", "web", "api", "compose", "infra"];

// ── Props ───────────────────────────────────────────────────────────────────

interface ChangelogDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
}

// ── Helpers ─────────────────────────────────────────────────────────────────

function groupByArea(changes: ChangelogChange[]) {
	const groups = new Map<ChangeArea, ChangelogChange[]>();
	for (const c of changes) {
		const existing = groups.get(c.area);
		if (existing) {
			existing.push(c);
		} else {
			groups.set(c.area, [c]);
		}
	}
	return groups;
}

function formatDate(dateStr: string) {
	const d = new Date(dateStr + "T00:00:00");
	return d.toLocaleDateString("en-US", {
		year: "numeric",
		month: "long",
		day: "numeric",
	});
}

// ── Component ───────────────────────────────────────────────────────────────

export function ChangelogDialog({ open, onOpenChange }: ChangelogDialogProps) {
	const sorted = useMemo(
		() =>
			[...changelog].sort((a, b) =>
				b.version.localeCompare(a.version, undefined, { numeric: true }),
			),
		[],
	);

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="max-w-[90vw] gap-0 p-0 xl:max-w-5xl">
				<DialogHeader className="shrink-0 border-b border-border px-8 py-6">
					<DialogTitle className="text-2xl font-bold tracking-tight">Updates</DialogTitle>
					<DialogDescription className="mt-1 text-sm text-muted-foreground">
						New features, improvements and fixes in Dockeeper.
					</DialogDescription>
				</DialogHeader>

				<div className="max-h-[80vh] overflow-y-auto px-8 py-8">
					{sorted.length === 0 ? (
						<p className="py-12 text-center text-sm text-muted-foreground">
							No changelog entries yet.
						</p>
					) : (
						<div className="space-y-0">
							{sorted.map((entry, i) => {
								const grouped = groupByArea(entry.changes);

								return (
									<div
										key={entry.version}
										className={cn(
											"grid gap-6 pb-10 sm:grid-cols-[140px_1fr] sm:gap-8",
											i > 0 && "border-t border-border pt-10",
										)}
									>
										{/* Left column — version meta */}
										<div className="sm:sticky sm:top-0 sm:self-start">
											<span className="inline-block rounded-md border border-border bg-muted/50 px-2.5 py-1 font-mono text-sm font-semibold">
												v{entry.version}
											</span>
											<time className="mt-1.5 block text-xs text-muted-foreground">
												{formatDate(entry.date)}
											</time>
										</div>

										{/* Right column — content */}
										<div className="min-w-0">
											{/* Title */}
											{entry.title && (
												<h3 className="mb-2 text-lg font-semibold tracking-tight">{entry.title}</h3>
											)}

											{/* Summary */}
											{entry.summary && (
												<p className="mb-6 text-sm leading-relaxed text-muted-foreground">
													{entry.summary}
												</p>
											)}

											{/* Changes grouped by area */}
											<div className="space-y-6">
												{AREA_ORDER.filter((a) => grouped.has(a)).map((area) => {
													const items = grouped.get(area)!;
													return (
														<section key={area}>
															{/* Category header */}
															<h4 className="mb-3 border-b border-border pb-2 text-xs font-semibold uppercase tracking-widest text-muted-foreground">
																{AREA_LABEL[area]}
															</h4>

															{/* Change items */}
															<div className="divide-y divide-border/40">
																{items.map((change, ci) => {
																	const badge = TYPE_BADGE[change.type];
																	return (
																		<div key={ci} className="flex items-start gap-3 py-2.5">
																			<span
																				className={cn(
																					"mt-0.5 inline-flex shrink-0 items-center rounded-full border px-2 py-0.5 text-[11px] font-semibold uppercase leading-none tracking-wide",
																					badge.className,
																				)}
																			>
																				{badge.label}
																			</span>
																			<p className="text-sm leading-relaxed text-muted-foreground">
																				{change.description}
																			</p>
																		</div>
																	);
																})}
															</div>
														</section>
													);
												})}
											</div>
										</div>
									</div>
								);
							})}
						</div>
					)}
				</div>
			</DialogContent>
		</Dialog>
	);
}
