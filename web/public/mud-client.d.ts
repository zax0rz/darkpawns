// Types for the shared MUD client. The module itself is plain ESM because the
// self-host splash serves it verbatim with no build step; this file exists so
// the two bundled hosts get a checked contract instead of `any`.

export interface MudPlayerState {
  health: number;
  maxHealth: number;
  mana: number;
  maxMana: number;
  move: number;
  maxMove: number;
  level: number;
  gold: number;
  /** Present once the player is in a room; drives the minimap. */
  roomVnum?: number;
}

export type MudConnectionState = 'connected' | 'disconnected';

export interface MudClientOptions {
  /** An opened xterm Terminal. The host constructs it: the three surfaces run
   *  different xterm versions and mount differently. */
  terminal: {
    write(data: string): void;
    writeln(data: string): void;
    onData(handler: (data: string) => void): unknown;
    /** xterm's column count and resize event. When present, server output is
     *  held while the terminal is narrower than 20 columns (not yet fitted). */
    cols?: number;
    onResize?(handler: (size: { cols: number; rows: number }) => void): unknown;
  };
  /** Defaults to /ws on the current origin, or the `host` query parameter. */
  wsUrl?: string;
  /** Document queried for the optional panel elements. Defaults to `document`. */
  doc?: Document;
  /** Answer the name prompt with this character. The admin console already
   *  knows who is signed in; the server still asks for the password. Nothing
   *  else should set this. */
  autoLogin?: string;
  /** Connection state, for hosts that render their own indicator. */
  onStatus?(state: MudConnectionState): void;
  /** Player vitals, for hosts whose status bar is not DOM this module can write
   *  to. Fires on every `state` and `vars` message. */
  onPlayerState?(state: MudPlayerState): void;
}

export interface MudClient {
  /** Reset session state and open a new socket. */
  connect(): void;
  /** Close the socket without the reconnect notice. Hosts with a lifecycle —
   *  a React component, say — must call this on teardown or leak the socket. */
  disconnect(): void;
}

export function createMudClient(options: MudClientOptions): MudClient;

export function pct(cur: number, max: number): number;
export function hpColor(p: number): string;
export function manaColor(p: number): string;
export function moveColor(p: number): string;
