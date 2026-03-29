import { useCallback, useEffect, useRef, useState } from "react";

export interface ScrollSpyOptions {
  rootMargin?: string;
  threshold?: number;
}

const DEFAULT_ROOT_MARGIN = "-10% 0px -50% 0px";
const DEFAULT_THRESHOLD = 0;
const INITIAL_SCROLL_DELAY_MS = 100;

export function useScrollSpy(
  sectionIds: string[],
  options?: ScrollSpyOptions,
) {
  const [activeId, setActiveId] = useState<string | null>(null);
  const observerRef = useRef<IntersectionObserver | null>(null);

  const rootMargin = options?.rootMargin ?? DEFAULT_ROOT_MARGIN;
  const threshold = options?.threshold ?? DEFAULT_THRESHOLD;

  // Stable serialization for dependency tracking
  const idsKey = sectionIds.join(",");

  useEffect(() => {
    const ids = idsKey.split(",").filter(Boolean);
    if (ids.length === 0) return;

    observerRef.current = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            setActiveId(entry.target.id);
            history.replaceState(null, "", "#" + entry.target.id);
          }
        }
      },
      { rootMargin, threshold },
    );

    for (const id of ids) {
      const el = document.getElementById(id);
      if (el) observerRef.current.observe(el);
    }

    // Scroll to initial hash on mount
    const hash = window.location.hash.slice(1);
    if (hash) {
      const timer = setTimeout(() => {
        const target = document.getElementById(hash);
        if (target) {
          target.scrollIntoView({ behavior: "smooth" });
          setActiveId(hash);
        }
      }, INITIAL_SCROLL_DELAY_MS);
      return () => {
        clearTimeout(timer);
        observerRef.current?.disconnect();
      };
    }

    return () => {
      observerRef.current?.disconnect();
    };
  }, [idsKey, rootMargin, threshold]);

  const scrollTo = useCallback((id: string) => {
    const el = document.getElementById(id);
    if (!el) return;
    el.scrollIntoView({ behavior: "smooth" });
    history.pushState(null, "", "#" + id);
    setActiveId(id);
  }, []);

  return { activeId, scrollTo };
}
