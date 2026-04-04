import { forwardRef } from "react";
import type { ApiEndpoint, ApiGroup } from "@/types/docs";
import { GettingStarted } from "./getting-started";
import { EndpointSection } from "./endpoint-section";

// ── Props ───────────────────────────────────────────────────────────────────

interface DocsContentProps {
	groups: ApiGroup[];
	onOpenPlayground?: (endpoint: ApiEndpoint) => void;
}

// ── Component ───────────────────────────────────────────────────────────────

export const DocsContent = forwardRef<HTMLDivElement, DocsContentProps>(function DocsContent(
	{ groups, onOpenPlayground },
	ref,
) {
	return (
		<div ref={ref} className="flex-1 overflow-y-auto">
			<main className="mx-auto max-w-6xl px-6 py-12 lg:px-8">
				{/* Getting Started always first */}
				<GettingStarted />

				{/* Endpoint groups */}
				{groups.map((group) => {
					if (group.endpoints.length === 0) return null;

					return (
						<div key={group.id} className="mt-24 border-t border-border pt-12">
							{/* Group header */}
							<div id={group.id} className="mb-12 scroll-mt-24">
								<h2 className="text-2xl font-bold mb-2">{group.title}</h2>
								<p className="text-sm text-muted-foreground">{group.description}</p>
							</div>

							{/* Endpoints */}
							<div className="space-y-56">
								{group.endpoints.map((endpoint) => (
									<EndpointSection
										key={endpoint.id}
										endpoint={endpoint}
										onOpenPlayground={onOpenPlayground}
									/>
								))}
							</div>
						</div>
					);
				})}
			</main>
		</div>
	);
});
