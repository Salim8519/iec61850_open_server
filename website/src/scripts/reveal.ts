// Fade sections in as they enter the viewport. Content is visible without JS and
// with reduced motion; only elements below the first screen start hidden.
export function startReveal(): void {
  const items = Array.from(document.querySelectorAll<HTMLElement>('.reveal'));
  if (!('IntersectionObserver' in window) || matchMedia('(prefers-reduced-motion: reduce)').matches) return;
  const fold = window.innerHeight * 0.92;
  const observer = new IntersectionObserver((entries) => {
    for (const entry of entries) {
      if (entry.isIntersecting) {
        entry.target.classList.remove('pending');
        observer.unobserve(entry.target);
      }
    }
  }, { rootMargin: '0px 0px -8% 0px' });
  for (const item of items) {
    if (item.getBoundingClientRect().top > fold) {
      item.classList.add('pending');
      observer.observe(item);
    }
  }
}
