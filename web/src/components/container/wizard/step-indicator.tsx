import { Check } from "lucide-react";
import { cn } from "@/lib/utils";
import { STEPS, type StepIndex } from "./schema";

interface StepIndicatorProps {
	currentStep: StepIndex;
	onStepClick: (step: StepIndex) => void;
}

export function StepIndicator({ currentStep, onStepClick }: StepIndicatorProps) {
	return (
		<nav aria-label="Wizard steps" className="flex items-center gap-1 overflow-x-auto pb-2">
			{STEPS.map((step, i) => {
				const isCompleted = i < currentStep;
				const isCurrent = i === currentStep;

				return (
					<button
						key={step.label}
						type="button"
						onClick={() => onStepClick(i as StepIndex)}
						className={cn(
							"flex shrink-0 items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors",
							isCurrent && "bg-primary text-primary-foreground",
							isCompleted && "text-primary hover:bg-accent",
							!isCurrent && !isCompleted && "text-muted-foreground hover:bg-accent",
						)}
					>
						<span
							className={cn(
								"flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-xs font-medium",
								isCurrent && "bg-primary-foreground text-primary",
								isCompleted && "bg-primary text-primary-foreground",
								!isCurrent && !isCompleted && "border border-muted-foreground/40",
							)}
						>
							{isCompleted ? <Check className="h-3.5 w-3.5" /> : i + 1}
						</span>
						<span className="hidden sm:inline">{step.label}</span>
					</button>
				);
			})}
		</nav>
	);
}
