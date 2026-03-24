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