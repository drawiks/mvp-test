import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";
import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";
import { cn } from "@/lib/utils";

const components: Components = {
  a({ href, children }) {
    return (
      <a
        href={href}
        className="text-gold underline underline-offset-2"
        onClick={(e) => {
          e.preventDefault();
          if (href) BrowserOpenURL(href);
        }}
      >
        {children}
      </a>
    );
  },
  p({ children }) {
    return <p className="my-1.5 leading-relaxed">{children}</p>;
  },
  ul({ children }) {
    return <ul className="my-1.5 list-disc space-y-0.5 pl-5">{children}</ul>;
  },
  ol({ children }) {
    return <ol className="my-1.5 list-decimal space-y-0.5 pl-5">{children}</ol>;
  },
  li({ children }) {
    return <li className="leading-relaxed">{children}</li>;
  },
  strong({ children }) {
    return <strong className="font-semibold text-foreground">{children}</strong>;
  },
  em({ children }) {
    return <em className="italic">{children}</em>;
  },
  code({ className, children }) {
    return (
      <code className={cn("rounded bg-muted px-1 py-0.5 font-mono text-[0.92em]", className)}>
        {children}
      </code>
    );
  },
  pre({ children }) {
    return (
      <pre className="my-2 max-h-64 overflow-auto rounded-md border border-border bg-background p-2 font-mono text-[11px] whitespace-pre-wrap break-all">
        {children}
      </pre>
    );
  },
  blockquote({ children }) {
    return <blockquote className="my-2 border-l-2 border-border pl-3 text-muted-foreground">{children}</blockquote>;
  },
  table({ children }) {
    return <table className="my-2 w-full border-collapse text-[11px]">{children}</table>;
  },
  th({ children }) {
    return <th className="border border-border px-2 py-1 text-left font-semibold">{children}</th>;
  },
  td({ children }) {
    return <td className="border border-border px-2 py-1">{children}</td>;
  },
  hr() {
    return <hr className="my-3 border-border" />;
  },
};

export default function MarkdownMessage({ text }: { text: string }) {
  return (
    <div className="text-sm">
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
        {text}
      </ReactMarkdown>
    </div>
  );
}
