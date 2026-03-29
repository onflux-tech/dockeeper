import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { ApiEndpoint, ApiParameter, AuthType, HttpMethod } from "@/types/docs";
import { CodeBlock } from "./code-block";

// ── Method badge styles ─────────────────────────────────────────────────────

const METHOD_STYLE: Record<HttpMethod, string> = {
  GET: "border-blue-400/30 text-blue-400 bg-blue-400/10",
  POST: "border-green-400/30 text-green-400 bg-green-400/10",
  PUT: "border-amber-400/30 text-amber-400 bg-amber-400/10",
  PATCH: "border-purple-400/30 text-purple-400 bg-purple-400/10",
  DELETE: "border-red-400/30 text-red-400 bg-red-400/10",
};

// ── Auth labels ─────────────────────────────────────────────────────────────

const AUTH_LABEL: Record<AuthType, { text: string; className: string }> = {
  required: { text: "Auth Required", className: "border-amber-400/30 text-amber-400" },
  "api-key": { text: "API Key", className: "border-blue-400/30 text-blue-400" },
  public: { text: "Public", className: "border-green-400/30 text-green-400" },
  superuser: { text: "Superuser Only", className: "border-red-400/30 text-red-400" },
};

// ── Parameter row ───────────────────────────────────────────────────────────

function ParamRow({ param }: { param: ApiParameter }) {
  return (
    <div className="flex flex-col gap-0.5 py-2">
      <div className="flex items-baseline gap-2">
        <span className="font-mono font-semibold text-sm">{param.name}</span>
        <span className="font-mono text-xs text-muted-foreground">{param.type}</span>
        {param.required ? (
          <span className="text-[10px] font-bold uppercase tracking-wider text-red-400">
            required
          </span>
        ) : (
          <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
            optional
          </span>
        )}
      </div>
      <p className="text-xs text-muted-foreground">{param.description}</p>
    </div>
  );
}

// ── Parameter list ──────────────────────────────────────────────────────────

function ParamList({
  title,
  params,
}: {
  title: string;
  params: ApiParameter[];
}) {
  if (params.length === 0) return null;

  return (
    <div className="mt-6">
      <h4 className="text-xs font-bold uppercase tracking-widest text-muted-foreground mb-2">
        {title}
      </h4>
      <div className="divide-y divide-border">
        {params.map((p) => (
          <ParamRow key={p.name} param={p} />
        ))}
      </div>
    </div>
  );
}

// ── Props ───────────────────────────────────────────────────────────────────

interface EndpointSectionProps {
  endpoint: ApiEndpoint;
}

// ── Component ───────────────────────────────────────────────────────────────

export function EndpointSection({ endpoint }: EndpointSectionProps) {
  const authInfo = AUTH_LABEL[endpoint.auth];

  return (
    <section id={endpoint.id} className="scroll-mt-24">
      <div className="flex flex-col xl:flex-row gap-8 xl:gap-12">
        {/* Left column — description */}
        <div className="flex-1 min-w-0">
          {/* Method + Title */}
          <div className="flex items-center gap-3 mb-3">
            <Badge
              variant="outline"
              className={cn("font-mono text-xs font-bold", METHOD_STYLE[endpoint.method])}
            >
              {endpoint.method}
            </Badge>
            <h3 className="text-lg font-bold">{endpoint.title}</h3>
          </div>

          {/* Path */}
          <div className="rounded-lg bg-muted/50 px-3 py-2 mb-4">
            <code className="font-mono text-sm text-muted-foreground">
              {endpoint.path}
            </code>
          </div>

          {/* Description */}
          <p className="text-sm text-muted-foreground leading-relaxed mb-4">
            {endpoint.description}
          </p>

          {/* Auth badge */}
          <div className="flex items-center gap-2 mb-2">
            <span className="text-xs font-bold uppercase tracking-widest text-muted-foreground">
              Auth
            </span>
            <Badge variant="outline" className={cn("text-xs", authInfo.className)}>
              {authInfo.text}
            </Badge>
            {endpoint.capability && (
              <span className="font-mono text-xs text-muted-foreground">
                ({endpoint.capability})
              </span>
            )}
          </div>

          {/* URL Parameters */}
          {endpoint.parameters && endpoint.parameters.length > 0 && (
            <ParamList title="URL Parameters" params={endpoint.parameters} />
          )}

          {/* Query Parameters */}
          {endpoint.queryParams && endpoint.queryParams.length > 0 && (
            <ParamList title="Query Parameters" params={endpoint.queryParams} />
          )}

          {/* Request Body */}
          {endpoint.requestBody && endpoint.requestBody.fields.length > 0 && (
            <div className="mt-6">
              <h4 className="text-xs font-bold uppercase tracking-widest text-muted-foreground mb-1">
                Request Body
              </h4>
              <p className="text-xs text-muted-foreground mb-2">
                {endpoint.requestBody.contentType}
              </p>
              <div className="divide-y divide-border">
                {endpoint.requestBody.fields.map((f) => (
                  <ParamRow key={f.name} param={f} />
                ))}
              </div>
            </div>
          )}

          {/* Response Example */}
          {endpoint.responseExample && (
            <div className="mt-6">
              <h4 className="text-xs font-bold uppercase tracking-widest text-muted-foreground mb-2">
                Response
              </h4>
              <pre className="rounded-lg bg-muted/50 p-3 overflow-x-auto text-xs font-mono leading-relaxed text-muted-foreground">
                <code>{endpoint.responseExample}</code>
              </pre>
            </div>
          )}
        </div>

        {/* Right column — code examples */}
        <div className="flex-1 xl:max-w-xl">
          <div className="xl:sticky xl:top-24">
            <CodeBlock
              curlCode={endpoint.curlExample}
              jsCode={endpoint.jsExample}
            />
          </div>
        </div>
      </div>
    </section>
  );
}
