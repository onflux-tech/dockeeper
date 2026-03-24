import { useState } from "react";
import { useForm, FormProvider } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useNavigate } from "react-router";
import { ArrowLeft, ArrowRight, Loader2, Rocket } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { useCreateContainer } from "@/hooks/use-create-container";
import { StepIndicator } from "./step-indicator";
import { containerWizardSchema, STEPS, type ContainerWizardData, type StepIndex } from "./schema";
import { ImageStep } from "./steps/image-step";
import { GeneralStep } from "./steps/general-step";
import { PortsStep } from "./steps/ports-step";
import { VolumesStep } from "./steps/volumes-step";
import { EnvStep } from "./steps/env-step";
import { LabelsStep } from "./steps/labels-step";
import { ResourcesStep } from "./steps/resources-step";
import { ReviewStep } from "./steps/review-step";

const STEP_COMPONENTS = [
	ImageStep,
	GeneralStep,
	PortsStep,
	VolumesStep,
	EnvStep,
	LabelsStep,
	ResourcesStep,
	ReviewStep,
] as const;

export function CreateContainerWizard() {
	const [step, setStep] = useState<StepIndex>(0);
	const navigate = useNavigate();
	const createMutation = useCreateContainer();

	const methods = useForm<ContainerWizardData>({
		resolver: zodResolver(containerWizardSchema),
		defaultValues: {
			image: "",
			auto_start: true,
			server_id: "",
			name: "",
			network: "",
			restart_policy: "",
			command: "",
			entrypoint: "",
			ports: [],
			volumes: [],
			env: [],
			labels: [],
			cpu_limit: 0,
			mem_limit: 0,
			pids_limit: 0,
		},
		mode: "onTouched",
	});

	const isLastStep = step === STEPS.length - 1;
	const isFirstStep = step === 0;
	const StepComponent = STEP_COMPONENTS[step];

	async function handleNext() {
		const fields = STEPS[step].fields as unknown as (keyof ContainerWizardData)[];
		if (fields.length > 0) {
			const valid = await methods.trigger(fields);
			if (!valid) return;
		}
		setStep((s) => Math.min(s + 1, STEPS.length - 1) as StepIndex);
	}

	function handleBack() {
		setStep((s) => Math.max(s - 1, 0) as StepIndex);
	}

	function handleStepClick(target: StepIndex) {
		// Allow clicking any previous step or the current step
		if (target <= step) {
			setStep(target);
		}
	}

	async function onSubmit(data: ContainerWizardData) {
		await createMutation.mutateAsync(data);
		navigate("/containers");
	}

	return (
		<FormProvider {...methods}>
			<form onSubmit={methods.handleSubmit(onSubmit)} className="flex flex-col gap-6">
				<StepIndicator currentStep={step} onStepClick={handleStepClick} />

				<Card>
					<CardHeader>
						<CardTitle>{STEPS[step].label}</CardTitle>
						<CardDescription>Step {step + 1} of {STEPS.length}</CardDescription>
					</CardHeader>
					<CardContent>
						<StepComponent />
					</CardContent>
				</Card>

				<div className="flex items-center justify-between">
					<Button
						type="button"
						variant="outline"
						onClick={handleBack}
						disabled={isFirstStep}
					>
						<ArrowLeft className="mr-1.5 h-4 w-4" />
						Back
					</Button>

					{isLastStep ? (
						<Button type="submit" disabled={createMutation.isPending}>
							{createMutation.isPending ? (
								<Loader2 className="mr-1.5 h-4 w-4 animate-spin" />
							) : (
								<Rocket className="mr-1.5 h-4 w-4" />
							)}
							Create Container
						</Button>
					) : (
						<Button type="button" onClick={handleNext}>
							Next
							<ArrowRight className="ml-1.5 h-4 w-4" />
						</Button>
					)}
				</div>
			</form>
		</FormProvider>
	);
}
