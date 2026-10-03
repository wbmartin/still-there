// A page's header block (its <h1> and the buttons beside it) stays pinned
// under the shell's fixed header while a long page scrolls. index.css does
// the pinning with the `.app-page` rule; this hook only measures.
//
// A page's own sticky rows (components/sticky-filters.tsx) must land below
// the pinned block, not under it, and the block's height varies by page: a
// wrapped title, a button row, a title that renders after its data loads.
// This writes that height to --app-page-header-height on the page element,
// re-read on every change under the page, every window resize and every
// resize of the block itself.
import { useEffect, type RefObject } from "react";

// The child of the page's root that is, or holds, the page's <h1>. Every
// page opens its root Container with a <Title order={1}>, alone or in a
// Group with its buttons, and has one <h1>. Kept in step with the
// `.app-page` rule in index.css.
export const PAGE_HEADER_SELECTOR = ".app-page > * > :is(h1, :has(h1))";

export const PAGE_HEADER_HEIGHT_VARIABLE = "--app-page-header-height";

// The page element must stay mounted for the life of the caller (AppLayout
// renders it on every route), so one effect run covers every route change:
// the MutationObserver sees the new page's nodes arrive.
export function usePinnedPageHeader(
  pageRef: RefObject<HTMLElement | null>,
) {
  useEffect(() => {
    const page = pageRef.current;
    if (!page) return;
    const headerResizeObserver = new ResizeObserver(() => read());
    let observedHeader: Element | null = null;
    const read = () => {
      const header = page.querySelector(PAGE_HEADER_SELECTOR);
      if (header !== observedHeader) {
        if (observedHeader) headerResizeObserver.unobserve(observedHeader);
        if (header) headerResizeObserver.observe(header);
        observedHeader = header;
      }
      const height = header
        ? `${Math.ceil(header.getBoundingClientRect().height)}px`
        : "0px";
      if (page.style.getPropertyValue(PAGE_HEADER_HEIGHT_VARIABLE) !== height) {
        page.style.setProperty(PAGE_HEADER_HEIGHT_VARIABLE, height);
      }
    };
    const pageMutationObserver = new MutationObserver(read);
    pageMutationObserver.observe(page, {
      childList: true,
      subtree: true,
      characterData: true,
    });
    window.addEventListener("resize", read);
    read();
    return () => {
      pageMutationObserver.disconnect();
      headerResizeObserver.disconnect();
      window.removeEventListener("resize", read);
    };
  }, [pageRef]);
}
