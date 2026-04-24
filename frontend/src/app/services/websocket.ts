import { Injectable, OnDestroy } from '@angular/core';
import { Subject } from 'rxjs';
import { environment } from '../../environments/environment';

// Match the Go "Task" struct
export interface Task {
  id: string;
  url: string;
  fileName: string;
  status: string;
  totalSize: number;
  downloaded: number;
  progress?: number;
  speed?: number;
  eta?: number;
}

export interface ProgressMessage {
  event: string
  id: string
  fileName: string
  percent: number
  totalSize: number
  speed: number
  eta: number
}

export interface ErrorMessage {
  event: string
  id: string
  message: string
}

@Injectable({
  providedIn: 'root',
})
export class Websocket implements OnDestroy {
  private socket: WebSocket | undefined;
  private reconnectAttempts = 0;
  private readonly maxReconnectAttempts = 20;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private intentionalClose = false;

  // Existing channel for live updates
  public progressUpdates$ = new Subject<ProgressMessage>();

  // Channel for loading history
  public historyUpdates$ = new Subject<Task[]>();

  // Channel for error events
  public errorUpdates$ = new Subject<ErrorMessage>();

  // Connection status
  public connectionStatus$ = new Subject<'connected' | 'disconnected' | 'reconnecting'>();

  constructor() { }

  connect() {
    this.intentionalClose = false;

    if (this.socket && (this.socket.readyState === WebSocket.OPEN || this.socket.readyState === WebSocket.CONNECTING)) {
      return; // Already connected or connecting
    }

    this.socket = new WebSocket(`${environment.wsBaseUrl}/ws`);

    this.socket.onopen = () => {
      console.log('✅ WS Connected');
      this.reconnectAttempts = 0;
      this.connectionStatus$.next('connected');
    };

    this.socket.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);

        // 1. Handle Live Progress
        if (data.event === 'progress') {
          this.progressUpdates$.next(data);
        }
        // 2. Handle Initial History (Load on Refresh)
        else if (data.event === 'initial_state') {
          // data.tasks comes from the Go backend
          this.historyUpdates$.next(data.tasks);
        }
        // 3. Handle Download Errors
        else if (data.event === 'error') {
          this.errorUpdates$.next(data);
        }
      } catch (e) {
        console.error('WS Parse Error', e);
      }
    };

    this.socket.onclose = () => {
      console.log('❌ Disconnected');
      this.connectionStatus$.next('disconnected');
      if (!this.intentionalClose) {
        this.scheduleReconnect();
      }
    };

    this.socket.onerror = (err) => {
      console.error('⚠️ WS Error', err);
      // onclose will fire after onerror, so reconnect is handled there
    };
  }

  private scheduleReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('🚫 Max reconnect attempts reached. Please refresh the page.');
      return;
    }

    this.reconnectAttempts++;
    // Exponential backoff: 1s, 2s, 4s, 8s, ... capped at 30s
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts - 1), 30000);
    console.log(`🔄 Reconnecting in ${delay / 1000}s (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
    this.connectionStatus$.next('reconnecting');

    this.reconnectTimer = setTimeout(() => {
      this.connect();
    }, delay);
  }

  disconnect() {
    this.intentionalClose = true;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.socket) {
      this.socket.close();
      this.socket = undefined;
    }
  }

  ngOnDestroy() {
    this.disconnect();
    this.progressUpdates$.complete();
    this.historyUpdates$.complete();
    this.errorUpdates$.complete();
    this.connectionStatus$.complete();
  }
}
