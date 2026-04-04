import type { ReactNode } from "react";
import { DocsHeader } from "./docs-header";
import { ChangelogDialog } from "./changelog-dialog";

interface DocsLayoutProps {
	children: ReactNode;
	mobileOpen: boolean;
	onMobileOpenChange: (open: boolean) => void;
	changelogOpen: boolean;
	onChangelogOpenChange: (open: boolean) => void;
	historySlot?: ReactNode;
}

export function DocsLayout({
	children,
	mobileOpen,
	onMobileOpenChange,
	changelogOpen,
	onChangelogOpenChange,
	historySlot,
}: DocsLayoutProps) {
	return (
		<div className="flex h-screen flex-col bg-background">
			<DocsHeader
				onMobileToggle={() => onMobileOpenChange(!mobileOpen)}
				onChangelogOpen={() => onChangelogOpenChange(true)}
				historySlot={historySlot}
			/>

			<div className="flex flex-1 overflow-hidden">{children}</div>

			<ChangelogDialog open={changelogOpen} onOpenChange={onChangelogOpenChange} />
		</div>
	);
}
