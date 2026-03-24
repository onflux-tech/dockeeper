import { Link } from "react-router";
import { ArrowLeft, Box } from "lucide-react";
import { Button } from "@/components/ui/button";
import { CreateContainerWizard } from "@/components/container/wizard/create-container-wizard";

export default function ContainerCreatePage() {
	return (
		<div className="flex flex-col gap-6">
			<div className="flex items-center gap-3">
				<Button variant="ghost" size="icon" asChild>
					<Link to="/containers">
						<ArrowLeft className="h-4 w-4" />
					</Link>
				</Button>
				<div className="flex items-center gap-2">
					<Box className="h-5 w-5 text-muted-foreground" />
					<h1 className="text-2xl font-semibold">Create Container</h1>
				</div>
			</div>

			<CreateContainerWizard />
		</div>
	);
}
