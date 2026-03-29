import { type RefObject, useCallback, useEffect, useRef, useState } from "react";

export interface ScrollSpyOptions {
  rootMargin?: string;
  threshold?: number;
}

const DEFAULT_ROOT_MARGIN = "-5% 0px -70% 0px";
const DEFAULT_THRESHOLD = 0;

/**
 * Scroll-spy hook that tracks which section is active based on visibility
 * within a scroll container. Supports programmatic scrollTo with observer
 * suppression that waits for scroll to actually finish (not a fixed timeout).
 */
export function useScrollSpy(
  sectionIds: string[],
  scrollContainerRef: RefObject<HTMLElement | null>,
  options?: ScrollSpyOptions,
) {
  const [activeId, setActiveId] = useState<string | null>(null);
  const observerRef = useRef<IntersectionObserver | null>(null);
  const isScrollingRef = useRef(false);
  /** Map of currently visible section IDs (maintained across observer batches) */
  const visibleRef = useRef<Map<string, IntersectionObserverEntry>>(new Map());

  const rootMargin = options?.rootMargin ?? DEFAULT_ROOT_MARGIN;
  const threshold = options?.threshold ?? DEFAULT_THRESHOLD;

  // Stable serialization for dependency tracking
  const idsKey = sectionIds.join(",");

  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    const ids = idsKey.split(",").filter(Boolean);
    if (ids.length === 0) return;

    const visible = visibleRef.current;
    visible.clear();

    observerRef.current = new IntersectionObserver(
      (entries) => {
        // Always update the visible map so it stays current
        for (const entry of entries) {
          if (entry.isIntersecting) {
            visible.set(entry.target.id, entry);
          } else {
            visible.delete(entry.target.id);
          }
        }

        // Skip picking active during programmatic scrolling
        if (isScrollingRef.current) return;

        // Pick the topmost visible section immediately
        pickActive(visible, setActiveId);
      },
      { root: container, rootMargin, threshold },
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
          target.scrollIntoView({ behavior: "smooth", block: "start" });
          setActiveId(hash);
        }
      }, 150);
      return () => {
        clearTimeout(timer);
        observerRef.current?.disconnect();
      };
    }

    return () => {
      observerRef.current?.disconnect();
    };
  }, [idsKey, rootMargin, threshold, scrollContainerRef]);

  const scrollTo = useCallback((id: string) => {
    const el = document.getElementById(id);
    if (!el) return;

    // Lock observer, set active, scroll instantly
    isScrollingRef.current = true;
    setActiveId(id);
    history.pushState(null, "", "#" + id);
    el.scrollIntoView({ behavior: "instant", block: "start" });

    // Unlock after a frame so the observer ignores the jump
    requestAnimationFrame(() => {
      isScrollingRef.current = false;
    });
  }, [scrollContainerRef]);

  return { activeId, scrollTo };
}

/** Pick the topmost visible section from the map */
function pickActive(
  visible: Map<string, IntersectionObserverEntry>,
  setActiveId: (id: string) => void,
) {
  let best: IntersectionObserverEntry | null = null;
  for (const entry of visible.values()) {
    if (!best || entry.boundingClientRect.top < best.boundingClientRect.top) {
      best = entry;
    }
  }
  if (best) {
    setActiveId(best.target.id);
  }
}
