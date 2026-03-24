import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router";
import {
	CommandDialog,
	CommandEmpty,
	CommandGroup,
	CommandInput,
	CommandItem,
	CommandList,
} from "@/components/ui/command";
import { navigation } from "@/lib/navigation";

export function CommandMenu() {
	const [open, setOpen] = useState(false);
	const navigate = useNavigate();

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

	const runCommand = useCallback(
		(href: string) => {
			setOpen(false);
			navigate(href);
		},
		[navigate],
	);

	return (
		<CommandDialog open={open} onOpenChange={setOpen}>
			<CommandInput placeholder="Type a command or search..." />
			<CommandList>
				<CommandEmpty>No results found.</CommandEmpty>
				{navigation.map((group) => (
					<CommandGroup key={group.label} heading={group.label}>
						{group.items.map((item) => (
							<CommandItem key={item.href} onSelect={() => runCommand(item.href)}>
								<item.icon className="mr-2 h-4 w-4" />
								{item.label}
							</CommandItem>
						))}
					</CommandGroup>
				))}
			</CommandList>
		</CommandDialog>
	);
}

export function useCommandMenu() {
	return {
		open: () => {
			document.dispatchEvent(
				new KeyboardEvent("keydown", {
					key: "k",
					metaKey: true,
					bubbles: true,
				}),
			);
		},
	};
}
