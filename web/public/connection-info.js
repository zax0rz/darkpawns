// Connection hints come from sockets the server actually opened, never defaults.
export function showConnections(info, hostname, document) {
  for (const [key, rowId, textId] of [
    ['telnet', 'telnet-connection', 'telnet-host'],
    ['telnet_tls', 'telnet-tls-connection', 'telnet-tls-host'],
  ]) {
    const row = document.getElementById(rowId);
    const text = document.getElementById(textId);
    if (!row || !text) continue;
    const endpoint = info?.[key];
    const valid = endpoint && typeof endpoint.bind === 'string' &&
      Number.isInteger(endpoint.port) && endpoint.port > 0 && endpoint.port <= 65535;
    row.hidden = !valid;
    if (!valid) { text.textContent = ''; continue; }
    const host = ['', '::', '0.0.0.0'].includes(endpoint.bind) ? hostname : endpoint.bind;
    text.textContent = `${host} port ${endpoint.port}`;
  }
}
