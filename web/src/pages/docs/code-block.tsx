import { useState } from "react";
import { Check, Clipboard } from "lucide-react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { highlightCurl, highlightJS } from "./syntax-highlight";

// ── Props ───────────────────────────────────────────────────────────────────

interface CodeBlockProps {
  curlCode: string;
  jsCode: string;
}

// ── Component ───────────────────────────────────────────────────────────────

export function CodeBlock({ curlCode, jsCode }: CodeBlockProps) {
  const [copied, setCopied] = useState(false);
  const [activeTab, setActiveTab] = useState("curl");

  const handleCopy = async () => {
    const raw = activeTab === "curl" ? curlCode : jsCode;
    await navigator.clipboard.writeText(raw);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <div className="rounded-xl border border-border bg-muted/30 shadow-lg overflow-hidden">
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <div className="flex items-center justify-between border-b border-border px-3">
          <TabsList variant="line" className="h-10 bg-transparent p-0">
            <TabsTrigger value="curl" className="text-xs">
              cURL
            </TabsTrigger>
            <TabsTrigger value="js" className="text-xs">
              JavaScript
            </TabsTrigger>
          </TabsList>

          <Button
            variant="ghost"
            size="icon-xs"
            onClick={handleCopy}
            aria-label={copied ? "Copied" : "Copy code"}
          >
            {copied ? (
              <Check className="text-green-400" />
            ) : (
              <Clipboard className="text-muted-foreground" />
            )}
          </Button>
        </div>

        <TabsContent value="curl" className="m-0">
          <pre className="p-4 overflow-x-auto text-xs font-mono leading-relaxed">
            <code>{highlightCurl(curlCode)}</code>
          </pre>
        </TabsContent>

        <TabsContent value="js" className="m-0">
          <pre className="p-4 overflow-x-auto text-xs font-mono leading-relaxed">
            <code>{highlightJS(jsCode)}</code>
          </pre>
        </TabsContent>
      </Tabs>
    </div>
  );
}
