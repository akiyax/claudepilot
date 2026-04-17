import { WSMessage } from '../types/ws';

type MessageHandler = (msg: WSMessage) => void;

// Message types that require ACK confirmation
const ACK_REQUIRED_TYPES = new Set([
  'permission.respond',
  'question.answer',
  'plan.approve',
]);

export class WebSocketService {
  private ws: WebSocket | null = null;
  private url: string = '';
  private token: string = '';
  private handler: MessageHandler | null = null;
  private reconnectAttempts: number = 0;
  private maxReconnectAttempts: number = 3;
  private reconnectDelays: number[] = [2000, 4000, 8000]; // exponential backoff
  private heartbeatInterval: ReturnType<typeof setInterval> | null = null;
  private pendingAcks: Map<string, { resolve: (msg: WSMessage) => void; reject: (err: Error) => void; timer: ReturnType<typeof setTimeout> }> = new Map();

  connect(url: string, token?: string): Promise<void> {
    this.url = url;
    this.token = token || '';
    this.reconnectAttempts = 0;

    return new Promise((resolve, reject) => {
      try {
        const wsUrl = token ? `${url}?token=${token}` : url;
        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
          console.log('[WS] Connected');
          this.startHeartbeat();
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const msg: WSMessage = JSON.parse(event.data);
            this.handleIncomingMessage(msg);
          } catch (err) {
            console.warn('[WS] Failed to parse message:', err);
          }
        };

        this.ws.onerror = (event) => {
          console.error('[WS] Error:', event);
          reject(new Error('WebSocket connection error'));
        };

        this.ws.onclose = (event) => {
          console.log('[WS] Closed:', event.code, event.reason);
          this.stopHeartbeat();
          this.tryReconnect();
        };
      } catch (err) {
        reject(err);
      }
    });
  }

  send(type: string, payload?: any): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn('[WS] Cannot send, not connected');
      return;
    }

    const msg: WSMessage = {
      type,
      timestamp: Date.now(),
      payload,
    };

    this.ws.send(JSON.stringify(msg));
  }

  sendWithAck(type: string, payload?: any, timeoutMs: number = 5000): Promise<WSMessage> {
    return new Promise((resolve, reject) => {
      const id = `ack-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
      const msg: WSMessage = {
        type,
        id,
        timestamp: Date.now(),
        payload,
      };

      const timer = setTimeout(() => {
        this.pendingAcks.delete(id);
        reject(new Error(`ACK timeout for ${type}`));
      }, timeoutMs);

      this.pendingAcks.set(id, { resolve, reject, timer });

      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        clearTimeout(timer);
        this.pendingAcks.delete(id);
        reject(new Error('Not connected'));
        return;
      }

      this.ws.send(JSON.stringify(msg));
    });
  }

  onMessage(handler: MessageHandler): void {
    this.handler = handler;
  }

  disconnect(): void {
    this.reconnectAttempts = this.maxReconnectAttempts; // Prevent reconnect
    this.stopHeartbeat();
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    // Clear pending ACKs
    for (const [, entry] of this.pendingAcks) {
      clearTimeout(entry.timer);
      entry.reject(new Error('Disconnected'));
    }
    this.pendingAcks.clear();
  }

  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }

  private handleIncomingMessage(msg: WSMessage): void {
    // Handle ACK responses
    if (msg.type === 'system.ack' && msg.payload?.refId) {
      const pending = this.pendingAcks.get(msg.payload.refId);
      if (pending) {
        clearTimeout(pending.timer);
        this.pendingAcks.delete(msg.payload.refId);
        pending.resolve(msg);
      }
    }

    // Forward to handler
    if (this.handler) {
      this.handler(msg);
    }
  }

  private tryReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.log('[WS] Max reconnect attempts reached');
      return;
    }

    const delay = this.reconnectDelays[this.reconnectAttempts] || 8000;
    this.reconnectAttempts++;

    console.log(`[WS] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);

    setTimeout(() => {
      this.connect(this.url, this.token).catch(err => {
        console.warn('[WS] Reconnect failed:', err);
      });
    }, delay);
  }

  private startHeartbeat(): void {
    this.stopHeartbeat();
    this.heartbeatInterval = setInterval(() => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: 'ping', timestamp: Date.now() }));
      }
    }, 30000);
  }

  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }
}

// Singleton instance
export const wsService = new WebSocketService();
