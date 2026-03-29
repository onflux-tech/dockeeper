import { useEffect, useMemo, useState } from "react";
import { apiGroups } from "@/data/api-reference";
import { useScrollSpy } from "@/hooks/use-scroll-spy";
import { DocsLayout } from "./docs-layout";
import { DocsSidebar } from "./docs-sidebar";
import { DocsContent } from "./docs-content";
import "./print.css";

// ── Page ────────────────────────────────────────────────────────────────────

export default function DocsPage() {
  const [mobileOpen, setMobileOpen] = useState(false);
  const [changelogOpen, setChangelogOpen] = useState(false);

  // Collect all section IDs for scroll-spy (group ids + endpoint ids)
  const sectionIds = useMemo(() => {
    const ids: string[] = [];
    for (const group of apiGroups) {
      if (group.endpoints.length === 0) {
        ids.push(group.id); // getting-started
      } else {
        ids.push(group.id); // group header
        for (const ep of group.endpoints) {
          ids.push(ep.id);
        }
      }
    }
    return ids;
  }, []);

  const { activeId, scrollTo } = useScrollSpy(sectionIds);

  // SEO meta tags
  useEffect(() => {
    const prev = document.title;
    document.title = "Dockeeper API Reference";

    const metas: HTMLMetaElement[] = [];
    const setMeta = (attr: string, key: string, content: string) => {
      const el = document.createElement("meta");
      el.setAttribute(attr, key);
      el.content = content;
      document.head.appendChild(el);
      metas.push(el);
    };

    setMeta("name", "description", "Complete REST API reference for Dockeeper — container management and monitoring platform.");
    setMeta("property", "og:title", "Dockeeper API Reference");
    setMeta("property", "og:description", "Complete REST API reference for Dockeeper — container management and monitoring platform.");
    setMeta("property", "og:type", "website");
    setMeta("property", "og:url", window.location.href);

    return () => {
      document.title = prev;
      metas.forEach((el) => el.remove());
    };
  }, []);

  const handleNavigate = (id: string) => {
    scrollTo(id);
    setMobileOpen(false);
  };

  return (
    <DocsLayout
      mobileOpen={mobileOpen}
      onMobileOpenChange={setMobileOpen}
      changelogOpen={changelogOpen}
      onChangelogOpenChange={setChangelogOpen}
    >
      <DocsSidebar
        groups={apiGroups}
        activeId={activeId}
        onNavigate={handleNavigate}
        mobileOpen={mobileOpen}
        onMobileClose={() => setMobileOpen(false)}
      />
      <DocsContent groups={apiGroups} />
    </DocsLayout>
  );
}
