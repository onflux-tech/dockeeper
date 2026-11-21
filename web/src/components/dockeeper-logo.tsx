import type { SVGProps } from "react";
import { cn } from "@/lib/utils";

interface DockeeperLogoProps extends Omit<SVGProps<SVGSVGElement>, "children"> {
	size?: number;
}

export function DockeeperLogo({ size = 32, className, ...props }: DockeeperLogoProps) {
	return (
		<svg
			xmlns="http://www.w3.org/2000/svg"
			viewBox="0 0 33.1 33.1"
			width={size}
			height={size}
			role="img"
			aria-label="Dockeeper"
			className={cn("shrink-0", className)}
			{...props}
		>
			<path d="M0 0L4.47 0L4.47 5.17L0 5.17L0 0Z" fill="#4ADE80" />
			<path d="M4.47 0L8.95 0L8.95 5.17L4.47 5.17L4.47 0Z" fill="#22C55E" />
			<path d="M8.95 0L13.42 0L13.42 5.17L8.95 5.17L8.95 0Z" fill="#22C55E" />
			<path d="M13.42 0L17.89 0L17.89 5.17L13.42 5.17L13.42 0Z" fill="#16A34A" />
			<path d="M0 5.17L4.47 5.17L4.47 10.34L0 10.34L0 5.17Z" fill="#22C55E" />
			<path d="M4.47 5.17L8.95 5.17L8.95 10.34L4.47 10.34L4.47 5.17Z" fill="#16A34A" />
			<path d="M8.95 5.17L13.42 5.17L13.42 10.34L8.95 10.34L8.95 5.17Z" fill="#15803D" />
			<path d="M13.42 5.17L17.89 5.17L17.89 10.34L13.42 10.34L13.42 5.17Z" fill="#22C55E" />
			<path d="M4.47 11.38L8.95 11.38L8.95 16.55L4.47 16.55L4.47 11.38Z" fill="#4ADE80" />
			<path d="M8.95 11.38L13.42 11.38L13.42 16.55L8.95 16.55L8.95 11.38Z" fill="#22C55E" />
			<path d="M13.42 11.38L17.89 11.38L17.89 16.55L13.42 16.55L13.42 11.38Z" fill="#16A34A" />
			<path d="M4.47 16.55L8.95 16.55L8.95 21.72L4.47 21.72L4.47 16.55Z" fill="#22C55E" />
			<path d="M8.95 16.55L13.42 16.55L13.42 21.72L8.95 21.72L8.95 16.55Z" fill="#16A34A" />
			<path d="M13.42 16.55L17.89 16.55L17.89 21.72L13.42 21.72L13.42 16.55Z" fill="#15803D" />
			<path d="M8.95 22.76L13.42 22.76L13.42 27.93L8.95 27.93L8.95 22.76Z" fill="#4ADE80" />
			<path d="M13.42 22.76L17.89 22.76L17.89 27.93L13.42 27.93L13.42 22.76Z" fill="#22C55E" />
			<path d="M8.95 27.93L13.42 27.93L13.42 33.1L8.95 33.1L8.95 27.93Z" fill="#22C55E" />
			<path d="M13.42 27.93L17.89 27.93L17.89 33.1L13.42 33.1L13.42 27.93Z" fill="#16A34A" />
			<path
				d="M18.79 0C26.69 0 33.1 7.41 33.1 16.55C33.1 25.69 26.69 33.1 18.79 33.1L18.79 0Z"
				fill="#22C55E"
			/>
		</svg>
	);
}
