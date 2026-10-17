// Tap-to-enlarge for product screenshots. Any element with data-zoom="<large image url>"
// opens that image in a native dialog; it scrolls and pinch-zooms, and closes on tap or Esc.
export function startLightbox(): void {
  const triggers = document.querySelectorAll<HTMLElement>('[data-zoom]');
  if (!triggers.length || typeof HTMLDialogElement === 'undefined') return;
  const dialog = document.createElement('dialog');
  dialog.className = 'lightbox';
  dialog.setAttribute('aria-label', 'Enlarged screenshot');
  dialog.innerHTML = '<button type="button" class="lightbox-close" aria-label="Close">×</button>'
    + '<div class="lightbox-scroll"><img alt="" decoding="async" /></div>'
    + '<p class="lightbox-hint">Pinch or scroll to look closer. Tap to close.</p>';
  document.body.append(dialog);
  const image = dialog.querySelector('img')!;
  let opener: HTMLElement | null = null;

  const open = (trigger: HTMLElement) => {
    opener = trigger;
    image.src = trigger.dataset.zoom!;
    image.alt = trigger.dataset.zoomAlt || '';
    dialog.showModal();
    document.documentElement.classList.add('menu-open');
  };
  dialog.addEventListener('close', () => {
    document.documentElement.classList.remove('menu-open');
    opener?.focus();
  });
  dialog.addEventListener('click', (event) => {
    const scroller = dialog.querySelector('.lightbox-scroll')!;
    const wide = scroller.scrollWidth > scroller.clientWidth + 4;
    if (event.target === image && wide) return;
    dialog.close();
  });
  triggers.forEach((trigger) => {
    trigger.setAttribute('role', 'button');
    trigger.tabIndex = 0;
    trigger.addEventListener('click', () => open(trigger));
    trigger.addEventListener('keydown', (event) => {
      if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); open(trigger); }
    });
  });
}
