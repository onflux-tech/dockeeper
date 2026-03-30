import { Link } from "react-router";
import { ArrowLeft, KeyRound, Menu } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { DockeeperLogo } from "@/components/dockeeper-logo";
import { changelog } from "@/data/changelog";

interface DocsHeaderProps {
  onMobileToggle: () => void;
  onChangelogOpen: () => void;
}

const latestVersion = changelog.length > 0 ? changelog[0].version : "0.0.0";

export function DocsHeader({
  onMobileToggle,
  onChangelogOpen,
}: DocsHeaderProps) {
  return (
    <header className="docs-header sticky top-0 z-50 flex h-14 items-center gap-3 border-b border-border bg-background/90 px-4 backdrop-blur-md">
      {/* Mobile menu toggle */}
      <Button
        variant="ghost"
        size="icon-sm"
        onClick={onMobileToggle}
        className="docs-mobile-toggle lg:hidden"
        aria-label="Toggle sidebar"
      >
        <Menu className="h-5 w-5" />
      </Button>

      {/* Brand */}
      <div className="flex items-center gap-2">
        <DockeeperLogo size={32} />
        <span className="hidden text-lg font-semibold sm:inline">
          Dockeeper
        </span>
        <Separator orientation="vertical" className="!h-5" />
        <span className="text-sm text-muted-foreground">API Reference</span>
      </div>

      {/* Version badge */}
      <button
        onClick={onChangelogOpen}
        className="cursor-pointer"
        aria-label="Open changelog"
      >
        <Badge
          variant="outline"
          className="border-primary/30 text-primary hover:bg-primary/10"
        >
          v{latestVersion}
        </Badge>
      </button>

      <div className="flex-1" />

      {/* Actions */}
      <div className="flex items-center gap-2">
        <Button variant="outline" size="sm" asChild>
          <Link to="/">
            <ArrowLeft className="mr-1.5 h-3.5 w-3.5" />
            <span className="hidden sm:inline">Dashboard</span>
          </Link>
        </Button>
        <Button size="sm" asChild>
          <Link to="/users?tab=api-keys">
            <KeyRound className="mr-1.5 h-3.5 w-3.5" />
            <span className="hidden sm:inline">Create API Key</span>
          </Link>
        </Button>
      </div>
    </header>
  );
}
