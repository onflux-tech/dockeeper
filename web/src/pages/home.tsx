import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Container } from "lucide-react";

export default function Home() {
	return (
		<main className="flex min-h-screen flex-col items-center justify-center gap-6 p-8">
			<div className="flex items-center gap-3">
				<Container className="h-10 w-10 text-primary" />
				<h1 className="text-4xl font-bold tracking-tight">Dockeeper</h1>
			</div>

			<Card className="w-full max-w-md">
				<CardHeader>
					<CardTitle>Welcome</CardTitle>
					<CardDescription>Docker container management &amp; monitoring platform</CardDescription>
				</CardHeader>
				<CardContent className="flex gap-2">
					<Button>Get Started</Button>
					<Button variant="outline">Documentation</Button>
				</CardContent>
			</Card>
		</main>
	);
}
