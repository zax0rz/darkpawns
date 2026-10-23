// The command remains selectable when JavaScript or clipboard access is unavailable.
for (const container of document.querySelectorAll<HTMLElement>('[data-copy-command]')) {
  const input = container.querySelector<HTMLInputElement | HTMLTextAreaElement>('input, textarea');
  const button = container.querySelector('button');
  const feedback = container.querySelector<HTMLElement>('[role="status"]');
  if (!input || !button || !feedback) continue;
  button.hidden = false;
  input.addEventListener('click', () => input.select());
  button.addEventListener('click', async () => {
    button.disabled = true;
    try {
      await navigator.clipboard.writeText(input.value);
      button.textContent = 'Copied';
      feedback.textContent = 'Command copied.';
    } catch {
      input.focus();
      input.select();
      button.textContent = 'Copy';
      feedback.textContent = 'Copy unavailable. The command is selected; copy it from the text box.';
    } finally {
      button.disabled = false;
    }
  });
}
