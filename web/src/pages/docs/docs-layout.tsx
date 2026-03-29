import type { ReactNode } from "react";
import { DocsHeader } from "./docs-header";

interface DocsLayoutProps {
  children: ReactNode;
  mobileOpen: boolean;
  onMobileOpenChange: (open: boolean) => void;
  changelogOpen: boolean;
  onChangelogOpenChange: (open: boolean) => void;
}

export function DocsLayout({
  children,
  mobileOpen,
  onMobileOpenChange,
  changelogOpen,
  onChangelogOpenChange,
}: DocsLayoutProps) {
  return (
    <div className="flex h-screen flex-col bg-background">
      <DocsHeader
        onMobileToggle={() => onMobileOpenChange(!mobileOpen)}
        onChangelogOpen={() => onChangelogOpenChange(true)}
      />

      <div className="flex flex-1 overflow-hidden">{children}</div>

      {/* ChangelogDialog will be wired here once built (Task 9) */}
      {changelogOpen && (
        <div className="hidden" data-changelog-placeholder>
          <button onClick={() => onChangelogOpenChange(false)} />
        </div>
      )}
    </div>
  );
}
