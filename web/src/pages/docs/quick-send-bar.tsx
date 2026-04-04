import { useCallback, useState } from "react";
import { ChevronDown, ChevronUp, Copy, ExternalLink, Loader2, Play } from "lucide-react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { ApiEndpoint } from "@/types/docs";
import { usePlaygroundAuth } from "./use-playground-auth";
import { ResourceSelect } from "./resource-select";
import { highlightJSON } from "./json-highlight";
import {
	buildRequestUrl,
	formatBytes,
	generateCurl,
	isQuickEligible,
	MAX_DISPLAY_BYTES,
	parsePathParams,
	statusColor,
} from "./playground-utils";

// ── Props ──────────────────────────────────────────────────────────────────

interface QuickSendBarProps {
	endpoint: ApiEndpoint;
	onOpenPlayground: () => void;
}

// ── Component ──────────────────────────────────────────────────────────────

export function QuickSendBar({ endpoint, onOpenPlayground }: QuickSendBarProps) {
	const { authHeaders: playgroundAuthHeaders } = usePlaygroundAuth();
	const [paramValue, setParamValue] = useState("");
	const [loading, setLoading] = useState(false);
	const [response, setResponse] = useState<{
		status: number;
		statusText: string;
		body: string;
		duration: number;
		size: number;
	} | null>(null);
	const [showBody, setShowBody] = useState(false);
	const [error, setError] = useState<string | null>(null);

	const pathParamNames = parsePathParams(endpoint.path);
	const paramName = pathParamNames[0] ?? null;

	const send = useCallback(async () => {
		setLoading(true);
		setError(null);
		setResponse(null);

		const origin = window.location.origin;
		const pParams: Record<string, string> = {};
		if (paramName) pParams[paramName] = paramValue;

		const url = buildRequestUrl(origin, endpoint.path, pParams, {});
		const headers: Record<string, string> = { ...playgroundAuthHeaders };

		const t0 = performance.now();

		try {
			const res = await fetch(url, {
				method: endpoint.method,
				headers,
			});

			const duration = Math.round(performance.now() - t0);
			const raw = await res.text();
			const size = new Blob([raw]).size;
			const display = size > MAX_DISPLAY_BYTES ? raw.slice(0, MAX_DISPLAY_BYTES) : raw;

			let prettyBody = display;
			try {
				prettyBody = JSON.stringify(JSON.parse(display), null, 2);
			} catch {
				// not JSON
			}

			setResponse({
				status: res.status,
				statusText: res.statusText,
				body: prettyBody,
				duration,
				size,
			});
		} catch (err: unknown) {
			const msg = err instanceof Error ? err.message : "Could not connect";
			setError(msg);
		} finally {
			setLoading(false);
		}
	}, [endpoint, paramName, paramValue, playgroundAuthHeaders]);

	if (!isQuickEligible(endpoint)) return null;

	// cURL for copy
	const origin = window.location.origin;
	const pParams: Record<string, string> = {};
	if (paramName) pParams[paramName] = paramValue || `{${paramName}}`;
	const curlUrl = buildRequestUrl(origin, endpoint.path, pParams, {});
	const curlHeaders: Record<string, string> = { ...playgroundAuthHeaders };

	return (
		<div className="mt-3 space-y-2 rounded-lg border border-border/60 bg-muted/20 p-3">
			{/* Bar */}
			<div className="flex items-center gap-2">
				{paramName && (
					<ResourceSelect
						endpointPath={endpoint.path}
						paramName={paramName}
						value={paramValue}
						onChange={setParamValue}
						className="h-8 flex-1 max-w-[220px]"
					/>
				)}

				<Button
					size="sm"
					className="h-8"
					disabled={loading || (!!paramName && !paramValue)}
					onClick={send}
				>
					{loading ? (
						<Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
					) : (
						<Play className="mr-1 h-3.5 w-3.5" />
					)}
					Send
				</Button>

				<Button
					variant="ghost"
					size="sm"
					className="h-8 text-xs text-muted-foreground"
					onClick={() =>
						navigator.clipboard.writeText(generateCurl(endpoint.method, curlUrl, curlHeaders))
					}
					aria-label="Copy as cURL"
				>
					<Copy className="mr-1 h-3 w-3" /> cURL
				</Button>

				<Button
					variant="ghost"
					size="sm"
					className="ml-auto h-8 text-xs text-muted-foreground"
					onClick={onOpenPlayground}
				>
					Full Playground <ExternalLink className="ml-1 h-3 w-3" />
				</Button>
			</div>

			{/* Error */}
			{error && <p className="text-xs text-red-400">{error}</p>}

			{/* Inline response */}
			{response && (
				<div className="space-y-1.5">
					<div className="flex items-center gap-2">
						<Badge
							variant="outline"
							className={cn("font-mono text-[10px] font-bold", statusColor(response.status))}
						>
							{response.status}
						</Badge>
						<span className="text-[10px] text-muted-foreground">{response.duration}ms</span>
						<span className="text-[10px] text-muted-foreground">{formatBytes(response.size)}</span>
						<Button
							variant="ghost"
							size="sm"
							className="ml-auto h-6 text-[10px]"
							onClick={() => setShowBody(!showBody)}
						>
							{showBody ? (
								<>
									<ChevronUp className="mr-1 h-2.5 w-2.5" /> Hide
								</>
							) : (
								<>
									<ChevronDown className="mr-1 h-2.5 w-2.5" /> Show body
								</>
							)}
						</Button>
					</div>

					{showBody && (
						<pre className="max-h-48 overflow-auto rounded-lg bg-muted/50 p-2 text-[11px] font-mono leading-relaxed">
							<code>{highlightJSON(response.body)}</code>
						</pre>
					)}
				</div>
			)}
		</div>
	);
}
