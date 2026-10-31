// Share one liveness request across all status indicators on the page.
const statuses = document.querySelectorAll<HTMLElement>('[data-server-status]');
if (statuses.length) {
  const settle = (state: 'online' | 'offline' | null, text: string) => {
    statuses.forEach(element => {
      element.classList.remove('online', 'offline');
      if (state) element.classList.add(state);
      const label = element.querySelector<HTMLElement>('[data-server-status-text]');
      if (label) label.textContent = text;
    });
  };
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 5000);
  fetch('/health', { signal: controller.signal, cache: 'no-store' })
    .then(response => settle(response.ok ? 'online' : 'offline', response.ok ? 'server online' : 'server offline'))
    .catch(() => settle(null, 'status unavailable'))
    .finally(() => clearTimeout(timeout));
}
