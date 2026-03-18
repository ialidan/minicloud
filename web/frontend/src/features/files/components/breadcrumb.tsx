import { ChevronRight, Home } from "lucide-react";

interface BreadcrumbProps {
  path: string;
  onNavigate: (path: string) => void;
}

export function Breadcrumb({ path, onNavigate }: BreadcrumbProps) {
  const segments = path ? path.split("/").filter(Boolean) : [];

  return (
    <nav aria-label="Breadcrumb" className="flex items-center gap-1 text-sm">
      <button
        onClick={() => onNavigate("")}
        className={`flex items-center gap-1 rounded-md px-1.5 py-1 transition-colors cursor-pointer ${
          segments.length === 0
            ? "text-foreground font-medium"
            : "text-muted-foreground hover:text-foreground hover:bg-surface-hover"
        }`}
        aria-current={segments.length === 0 ? "page" : undefined}
      >
        <Home className="h-3.5 w-3.5" />
        <span>Files</span>
      </button>

      {segments.map((segment, i) => {
        const segmentPath = "/" + segments.slice(0, i + 1).join("/");
        const isLast = i === segments.length - 1;

        return (
          <span key={segmentPath} className="flex items-center gap-1">
            <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
            {isLast ? (
              <span
                className="rounded-md px-1.5 py-1 font-medium text-foreground"
                aria-current="page"
              >
                {segment}
              </span>
            ) : (
              <button
                onClick={() => onNavigate(segmentPath)}
                className="rounded-md px-1.5 py-1 text-muted-foreground transition-colors hover:text-foreground hover:bg-surface-hover cursor-pointer"
              >
                {segment}
              </button>
            )}
          </span>
        );
      })}
    </nav>
  );
}
