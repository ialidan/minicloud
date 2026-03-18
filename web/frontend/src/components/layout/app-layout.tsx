import { Outlet, Link, useLocation } from "react-router-dom";
import { useState } from "react";
import { Cloud, Moon, Sun, Users } from "lucide-react";
import { Button } from "@/components/ui/button";
import { UserMenu } from "@/components/layout/user-menu";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";

export function AppLayout() {
  const { user } = useAuth();
  const location = useLocation();

  return (
    <div className="min-h-screen bg-background text-foreground">
      <header className="sticky top-0 z-50 border-b border-border bg-background/80 backdrop-blur-sm">
        <div className="mx-auto flex h-14 max-w-[1200px] items-center justify-between px-4 sm:px-6">
          <div className="flex items-center gap-4">
            <Link to="/files" className="flex items-center gap-2">
              <Cloud className="h-5 w-5 text-primary" />
              <span className="text-base font-semibold tracking-tight">
                MiniCloud
              </span>
            </Link>
            {user?.role === "admin" && (
              <Link
                to="/admin/users"
                className={cn(
                  "flex items-center gap-1.5 text-sm transition-colors",
                  location.pathname.startsWith("/admin")
                    ? "text-foreground font-medium"
                    : "text-muted-foreground hover:text-foreground",
                )}
              >
                <Users className="h-4 w-4" />
                Users
              </Link>
            )}
          </div>
          <div className="flex items-center gap-2">
            <ThemeToggle />
            <UserMenu />
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-[1200px] px-4 py-6 sm:px-6">
        <Outlet />
      </main>
    </div>
  );
}

function ThemeToggle() {
  const [dark, setDark] = useState(() =>
    document.documentElement.classList.contains("dark"),
  );

  function toggle() {
    const next = !dark;
    setDark(next);
    document.documentElement.classList.toggle("dark", next);
    localStorage.setItem("minicloud-theme", next ? "dark" : "light");
  }

  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={toggle}
      aria-label={dark ? "Switch to light mode" : "Switch to dark mode"}
    >
      {dark ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
    </Button>
  );
}
