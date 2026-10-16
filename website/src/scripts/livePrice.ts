import { SITE } from '../data/site';

// The license server owns the price. Pages are built with the last known price and
// then show the live one, so a price change on the server needs no website rebuild.
export function showLivePrice(): void {
  const targets = document.querySelectorAll<HTMLElement>('[data-price]');
  if (targets.length === 0) return;
  fetch(`${SITE.licenseServer}/api/v1/product`, { mode: 'cors', credentials: 'omit' })
    .then((response) => (response.ok ? response.json() : null))
    .then((product: { price?: string; currency?: string } | null) => {
      if (!product?.price || !/^\d+(\.\d{2})?$/.test(product.price)) return;
      const short = product.price.endsWith('.00') ? product.price.slice(0, -3) : product.price;
      for (const target of targets) target.textContent = short;
    })
    .catch(() => undefined);
}
