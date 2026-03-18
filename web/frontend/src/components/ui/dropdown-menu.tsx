import { useState, useRef, useEffect, useCallback, type ReactNode } from "react";
import { createPortal } from "react-dom";

interface DropdownMenuProps {
  trigger: ReactNode;
  children: ReactNode;
}

export function DropdownMenu({ trigger, children }: DropdownMenuProps) {
  const [open, setOpen] = useState(false);
  const triggerRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState({ top: 0, left: 0 });

  // Compute position when opening
  useEffect(() => {
    if (!open || !triggerRef.current) return;

    const rect = triggerRef.current.getBoundingClientRect();
    setPos({
      top: rect.bottom + 4,
      left: rect.right,
    });
  }, [open]);

  // Reposition if the menu would overflow the viewport bottom
  useEffect(() => {
    if (!open || !listRef.current) return;

    const menuRect = listRef.current.getBoundingClientRect();
    const viewportH = window.innerHeight;

    if (menuRect.bottom > viewportH - 8) {
      // Flip above the trigger
      const triggerRect = triggerRef.current!.getBoundingClientRect();
      setPos((prev) => ({
        ...prev,
        top: triggerRect.top - menuRect.height - 4,
      }));
    }
  }, [open, pos.top]);

  useEffect(() => {
    if (!open) return;

    function handleClickOutside(e: MouseEvent) {
      const target = e.target as Node;
      if (
        triggerRef.current && !triggerRef.current.contains(target) &&
        listRef.current && !listRef.current.contains(target)
      ) {
        setOpen(false);
      }
    }

    function handleEscape(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }

    function handleScroll() {
      setOpen(false);
    }

    document.addEventListener("mousedown", handleClickOutside);
    document.addEventListener("keydown", handleEscape);
    window.addEventListener("scroll", handleScroll, true);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleEscape);
      window.removeEventListener("scroll", handleScroll, true);
    };
  }, [open]);

  // Focus first item when opened
  useEffect(() => {
    if (open && listRef.current) {
      const firstItem = listRef.current.querySelector<HTMLElement>('[role="menuitem"]');
      firstItem?.focus();
    }
  }, [open]);

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (!listRef.current) return;
    const items = Array.from(listRef.current.querySelectorAll<HTMLElement>('[role="menuitem"]'));
    const currentIndex = items.indexOf(document.activeElement as HTMLElement);

    if (e.key === "ArrowDown") {
      e.preventDefault();
      const next = currentIndex < items.length - 1 ? currentIndex + 1 : 0;
      items[next]?.focus();
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      const prev = currentIndex > 0 ? currentIndex - 1 : items.length - 1;
      items[prev]?.focus();
    } else if (e.key === "Home") {
      e.preventDefault();
      items[0]?.focus();
    } else if (e.key === "End") {
      e.preventDefault();
      items[items.length - 1]?.focus();
    }
  }, []);

  return (
    <div ref={triggerRef}>
      <div onClick={() => setOpen(!open)}>{trigger}</div>
      {open &&
        createPortal(
          <div
            ref={listRef}
            role="menu"
            style={{
              position: "fixed",
              top: pos.top,
              left: pos.left,
              transform: "translateX(-100%)",
            }}
            className="z-[100] min-w-[160px] rounded-lg border border-border bg-surface p-1 shadow-lg animate-in zoom-in-95"
            onClick={() => setOpen(false)}
            onKeyDown={handleKeyDown}
          >
            {children}
          </div>,
          document.body,
        )}
    </div>
  );
}

interface DropdownItemProps {
  onClick: () => void;
  children: ReactNode;
  variant?: "default" | "danger";
}

export function DropdownItem({
  onClick,
  children,
  variant = "default",
}: DropdownItemProps) {
  return (
    <button
      role="menuitem"
      onClick={onClick}
      className={`flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors cursor-pointer outline-none focus-visible:ring-2 focus-visible:ring-ring ${
        variant === "danger"
          ? "text-danger hover:bg-danger/10 focus-visible:bg-danger/10"
          : "text-foreground hover:bg-surface-hover focus-visible:bg-surface-hover"
      }`}
    >
      {children}
    </button>
  );
}
