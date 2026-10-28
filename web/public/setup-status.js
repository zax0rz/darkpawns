// Preserve the last known state on failure; the initial conditional warning
// stays visible until a successful status read.
export function showSetupStatus(info, document) {
  const notice = document.getElementById('setup-notice');
  const heading = document.getElementById('setup-heading');
  const message = document.getElementById('setup-message');
  const next = document.getElementById('setup-next');
  if (!notice || !heading || !message || !next) return;
  if (typeof info?.first_character_pending !== 'boolean' || typeof info?.started_empty !== 'boolean') return;
  if (info.first_character_pending) {
    notice.hidden = false;
    heading.textContent = 'Create your immortal first';
    message.textContent = 'The first character you create becomes the immortal (level 40) and unlocks public access. Create yours before you let anyone else in.';
    next.textContent = 'Play locally to create your character, then restart the server to use the public addresses you configured.';
  } else if (info.started_empty) {
    notice.hidden = false;
    heading.textContent = 'Your first character is ready';
    message.textContent = 'Sign in to the admin console with your immortal character.';
    next.textContent = 'This server started without characters. Restart it to use the public addresses you configured.';
  } else {
    notice.hidden = true;
  }
}
