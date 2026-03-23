import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod/v4";
import { zodResolver } from "@hookform/resolvers/zod";
import { Navigate } from "react-router";
import { Container, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useAuthStore } from "@/stores/auth-store";
import { ApiRequestError } from "@/lib/api";

const loginSchema = z.object({
	email: z.email("Enter a valid email address"),
	password: z.string().min(1, "Password is required"),
});

type LoginForm = z.infer<typeof loginSchema>;

export default function LoginPage() {
	const { isAuthenticated, login } = useAuthStore();
	const [error, setError] = useState<string | null>(null);

	const {
		register,
		handleSubmit,
		formState: { errors, isSubmitting },
	} = useForm<LoginForm>({
		resolver: zodResolver(loginSchema),
	});

	if (isAuthenticated) {
		return <Navigate to="/" replace />;
	}

	const onSubmit = async (data: LoginForm) => {
		setError(null);
		try {
			await login(data.email, data.password);
		} catch (err) {
			if (err instanceof ApiRequestError) {
				setError(err.status === 429 ? "Too many attempts. Try again later." : err.message);
			} else {
				setError("An unexpected error occurred");
			}
		}
	};

	return (
		<div
			className="flex min-h-screen items-center justify-center p-4"
			style={
				{
					"--primary": "oklch(0.723 0.219 149.579)",
					"--primary-foreground": "oklch(0.982 0.018 155.826)",
					"--ring": "oklch(0.723 0.219 149.579)",
				} as React.CSSProperties
			}
		>
			<Card className="w-full max-w-sm">
				<CardHeader className="text-center">
					<div className="mx-auto mb-2 flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10">
						<Container className="h-6 w-6 text-primary" />
					</div>
					<CardTitle className="text-2xl">Dockeeper</CardTitle>
					<CardDescription>Sign in to manage your containers</CardDescription>
				</CardHeader>
				<CardContent>
					<form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4">
						{error && (
							<div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
								{error}
							</div>
						)}

						<div className="flex flex-col gap-1.5">
							<label htmlFor="email" className="text-sm font-medium">
								Email
							</label>
							<Input
								id="email"
								type="email"
								placeholder="admin@example.com"
								autoComplete="email"
								aria-invalid={!!errors.email}
								{...register("email")}
							/>
							{errors.email && <p className="text-xs text-destructive">{errors.email.message}</p>}
						</div>

						<div className="flex flex-col gap-1.5">
							<label htmlFor="password" className="text-sm font-medium">
								Password
							</label>
							<Input
								id="password"
								type="password"
								placeholder="••••••••"
								autoComplete="current-password"
								aria-invalid={!!errors.password}
								{...register("password")}
							/>
							{errors.password && (
								<p className="text-xs text-destructive">{errors.password.message}</p>
							)}
						</div>

						<Button type="submit" disabled={isSubmitting} className="mt-1 w-full">
							{isSubmitting ? <Loader2 className="animate-spin" /> : "Sign in"}
						</Button>
					</form>
				</CardContent>
			</Card>
		</div>
	);
}
