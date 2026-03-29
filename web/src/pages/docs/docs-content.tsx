import { ScrollArea } from "@/components/ui/scroll-area";
import type { ApiGroup } from "@/types/docs";
import { GettingStarted } from "./getting-started";
import { EndpointSection } from "./endpoint-section";

// ── Props ───────────────────────────────────────────────────────────────────

interface DocsContentProps {
  groups: ApiGroup[];
}

// ── Component ───────────────────────────────────────────────────────────────

export function DocsContent({ groups }: DocsContentProps) {
  return (
    <ScrollArea className="flex-1">
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
              <div className="space-y-16">
                {group.endpoints.map((endpoint) => (
                  <EndpointSection key={endpoint.id} endpoint={endpoint} />
                ))}
              </div>
            </div>
          );
        })}
      </main>
    </ScrollArea>
  );
}
