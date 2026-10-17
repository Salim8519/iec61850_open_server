// Shows the phone buy bar after the page's first screen and hides it while the page's own
// buying buttons ([data-cta-zone]) or the footer are on screen, so it never doubles them.
export function startCtaBar(): void {
  const bar = document.querySelector<HTMLElement>('[data-cta-bar]');
  if (!bar || !('IntersectionObserver' in window)) return;
  const phone = matchMedia('(max-width: 759px)');
  const zones = Array.from(document.querySelectorAll<HTMLElement>('[data-cta-zone], .site-footer'));
  const visibleZones = new Set<Element>();
  let pastTop = false;

  const update = () => {
    const show = phone.matches && pastTop && visibleZones.size === 0;
    bar.hidden = !phone.matches;
    bar.classList.toggle('shown', show);
    bar.toggleAttribute('inert', !show);
    document.documentElement.classList.toggle('cta-bar', phone.matches);
  };

  const zoneObserver = new IntersectionObserver((entries) => {
    for (const entry of entries) {
      if (entry.isIntersecting) visibleZones.add(entry.target); else visibleZones.delete(entry.target);
    }
    update();
  });
  zones.forEach((zone) => zoneObserver.observe(zone));

  const onScroll = () => {
    const next = window.scrollY > window.innerHeight * 0.8;
    if (next !== pastTop) { pastTop = next; update(); }
  };
  addEventListener('scroll', onScroll, { passive: true });
  phone.addEventListener('change', update);
  onScroll();
  update();
}
