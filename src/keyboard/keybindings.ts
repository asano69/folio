// src/keyboard/keybindings.ts

// ── Keyboard command registry ──────────────────────────────────

// KeyBinding describes a command that can be triggered by a keyboard shortcut.
interface KeyBinding {
  id: string;
  description: string;
  keys: string[];  // e.g. ['ctrl+h', 'ctrl+shift+h']
  handler: (e: KeyboardEvent) => void;
  // shouldFire determines whether this binding should be active in the current context.
  // Returns true if the binding should be active, false otherwise.
  shouldFire: (context: KeyBindingContext) => boolean;
}

// KeyBindingContext provides information about the current UI state.
interface KeyBindingContext {
  isDrawPaneOpen: boolean;
  isEditPaneOpen: boolean;
  isTOCPaneOpen: boolean;
  isViewerPage: boolean;
  focusedElement: HTMLElement | null;
}

// KeyBindingManager centrally manages all keyboard shortcuts.
class KeyBindingManager {
  private bindings: KeyBinding[] = [];
  private keyMap: Map<string, KeyBinding[]> = new Map();

  register(binding: KeyBinding): void {
    this.bindings.push(binding);
    for (const key of binding.keys) {
      const normalizedKey = this.normalizeKey(key);
      if (!this.keyMap.has(normalizedKey)) {
        this.keyMap.set(normalizedKey, []);
      }
      this.keyMap.get(normalizedKey)!.push(binding);
    }
  }

  // handleKeydown is the single entry point for all keyboard events.
  handleKeydown(e: KeyboardEvent, context: KeyBindingContext): void {
    const key = this.eventToKey(e);
    const candidates = this.keyMap.get(key);

    if (!candidates) return;

    for (const binding of candidates) {
      if (binding.shouldFire(context)) {
        e.preventDefault();
        binding.handler(e);
        return; // Stop propagation after first match.
      }
    }
  }

  private eventToKey(e: KeyboardEvent): string {
    const parts: string[] = [];
    if (e.ctrlKey) parts.push('ctrl');
    if (e.altKey) parts.push('alt');
    if (e.shiftKey) parts.push('shift');
    parts.push(e.key.toLowerCase());
    return parts.join('+');
  }

  private normalizeKey(key: string): string {
    return key.toLowerCase();
  }

  getAllBindings(): KeyBinding[] {
    return this.bindings;
  }
}

// ── Global instance ────────────────────────────────────────────

export const keyboardManager = new KeyBindingManager();

// ── Helper functions for common shouldFire predicates ──────────

export function onlyOnViewerPage(ctx: KeyBindingContext): boolean {
  return ctx.isViewerPage;
}

export function notWhenEditingText(ctx: KeyBindingContext): boolean {
  const elem = ctx.focusedElement;
  return elem?.tagName !== 'INPUT' && elem?.tagName !== 'TEXTAREA';
}

export function whenDrawPaneOpen(ctx: KeyBindingContext): boolean {
  return ctx.isDrawPaneOpen;
}

export function whenEditPaneOpen(ctx: KeyBindingContext): boolean {
  return ctx.isEditPaneOpen;
}

export function whenTOCPaneOpen(ctx: KeyBindingContext): boolean {
  return ctx.isTOCPaneOpen;
}

export function whenAnyPaneOpen(ctx: KeyBindingContext): boolean {
  return ctx.isDrawPaneOpen || ctx.isEditPaneOpen || ctx.isTOCPaneOpen;
}

export function whenNoPaneOpen(ctx: KeyBindingContext): boolean {
  return !whenAnyPaneOpen(ctx);
}

export { KeyBinding, KeyBindingContext };
