import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router";
import {
	Container,
	Globe,
	HardDrive,
	Layers,
	Loader2,
	Network,
	Rocket,
	Server,
} from "lucide-react";
import {
	CommandDialog,
	CommandEmpty,
	CommandGroup,
	CommandInput,
	CommandItem,
	CommandList,
} from "@/components/ui/command";
import { navigation } from "@/lib/navigation";
import { useGlobalSearch, type SearchResultGroup } from "@/hooks/use-global-search";
import type { ComponentType } from "react";

const typeIcons: Record<string, ComponentType<{ className?: string }>> = {
	servers: Server,
	containers: Container,
	images: Layers,
	volumes: HardDrive,
	networks: Network,
	compose_stacks: Rocket,
	traefik_routes: Globe,
};

export function CommandMenu() {
	const [open, setOpen] = useState(false);
	const [search, setSearch] = useState("");
	const navigate = useNavigate();
	const { results, isLoading } = useGlobalSearch(open ? search : "");

	useEffect(() => {
		function onKeyDown(e: KeyboardEvent) {
			if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
				e.preventDefault();
				setOpen((prev) => !prev);
			}
		}
		document.addEventListener("keydown", onKeyDown);
		return () => document.removeEventListener("keydown", onKeyDown);
	}, []);

	const handleOpenChange = useCallback((value: boolean) => {
		setOpen(value);
		if (!value) setSearch("");
	}, []);

	const runCommand = useCallback(
		(href: string) => {
			setOpen(false);
			navigate(href);
		},
		[navigate],
	);

	const nonEmptyGroups = results.filter((g: SearchResultGroup) => g.items.length > 0);
	const hasSearch = search.trim().length >= 2;

	return (
		<CommandDialog open={open} onOpenChange={handleOpenChange} shouldFilter={!hasSearch}>
			<CommandInput
				placeholder="Type a command or search..."
				value={search}
				onValueChange={setSearch}
			/>
			<CommandList>
				<CommandEmpty>
					{isLoading ? "Searching..." : "No results found."}
				</CommandEmpty>

				{/* Static navigation items – only in navigation mode */}
				{!hasSearch &&
					navigation.map((group) => (
						<CommandGroup key={group.label} heading={group.label}>
							{group.items.map((item) => (
								<CommandItem key={item.href} onSelect={() => runCommand(item.href)}>
									<item.icon className="mr-2 h-4 w-4" />
									{item.label}
								</CommandItem>
							))}
						</CommandGroup>
					))}

				{/* Live entity search results – only in search mode */}
				{hasSearch && isLoading && (
					<CommandGroup heading="Searching...">
						<CommandItem disabled>
							<Loader2 className="mr-2 h-4 w-4 animate-spin" />
							Loading results...
						</CommandItem>
					</CommandGroup>
				)}

				{nonEmptyGroups.map((group: SearchResultGroup) => {
					const Icon = typeIcons[group.type] ?? Server;
					return (
						<CommandGroup key={group.type} heading={group.label}>
							{group.items.map((item) => (
								<CommandItem key={item.id} onSelect={() => runCommand(item.href)}>
									<Icon className="mr-2 h-4 w-4" />
									<span>{item.name}</span>
									{item.subtitle && (
										<span className="ml-2 text-xs text-muted-foreground">
											{item.subtitle}
										</span>
									)}
								</CommandItem>
							))}
							{group.total > group.items.length && (
								<CommandItem onSelect={() => runCommand(group.viewAllHref)}>
									<span className="text-xs text-muted-foreground">
										View all {group.total} {group.label.toLowerCase()}
									</span>
								</CommandItem>
							)}
						</CommandGroup>
					);
				})}
			</CommandList>
		</CommandDialog>
	);
}
